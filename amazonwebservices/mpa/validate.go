package mpa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go/service/textract"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	doitRole   = "doitintl_cmp"
	doitPolicy = "doitintl_cmp"

	TimeGranularityHourly = costandusagereportservice.TimeUnitHourly
	ReportFormats         = map[string]string{
		costandusagereportservice.ReportFormatTextOrcsv: costandusagereportservice.CompressionFormatGzip,
		costandusagereportservice.ReportFormatParquet:   costandusagereportservice.CompressionFormatParquet,
	}
	SchemaElementsResources = costandusagereportservice.SchemaElementResources

	errorRole          = errors.New("no valid Role was found")
	errorPolicy        = errors.New("no valid Policy was found")
	errorBucket        = errors.New("no valid S3 Bucket was found")   // validation fallback error
	errorBucketEmpty   = errors.New("given S3 bucket has no content") // validation fallback error
	errorCUR           = errors.New("no valid CUR was found")
	errorCURPath       = errors.New("CURs has wrong bucket or path")
	errorCURProperties = errors.New("CURs has wrong properties (compression, time unit, format or schema)")
	errorCURNotFound   = errors.New("given CUR not found on s3 bucket") // validation fallback error
	curErrors          = []error{errorCUR, errorCURPath, errorCURProperties, errorCURNotFound, errorBucket, errorBucketEmpty}

	errorMissingPermissions = "missing permissions"

	accessDeniedExceptionError = textract.ErrCodeAccessDeniedException
	accessDeniedError          = "AccessDenied"
)

// ValidateMPA validates MPA account to clarify it is accessible from our platform
func (s *MasterPayerAccountService) ValidateMPA(ctx context.Context, req *ValidateMPARequest) error {
	l := s.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		"accountId": req.AccountID,
		"service":   "mpa",
		"flow":      "validate",
	})

	if err := s.validateCUR(req.AccountID, req.CURBucket, req.CURPath); err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if errCode := awsErr.Code(); errCode == accessDeniedError || errCode == accessDeniedExceptionError { // fallback in case of missing permissions
				l.Printf("MPA validation fallback for account %s, missing permission 'cur:DescribeReportDefinitions'. error: %s", req.AccountID, awsErr.Error())
				err = s.validateCURFileExistence(req.AccountID, req.CURBucket, req.CURPath)
			}
		}

		/* fails if:
		1. has cur:DescribeReportDefinitions permission but fail to find a valid CUR
		2. missing cur:DescribeReportDefinitions permission and fails to find existence of a given CUR
		*/
		if err != nil {
			for _, curError := range curErrors {
				if err == curError {
					return err
				}
			}

			l.Error(err)

			return errorCUR
		}
	}

	if _, err := s.validateRole(req.AccountID, req.RoleArn, false); err != nil {
		l.Error(err)
		return errorRole
	}

	if _, err := s.validatePolicy(ctx, req.AccountID, req.RoleArn, req.CURBucket, false, nil); err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if errCode := awsErr.Code(); errCode == accessDeniedError || errCode == accessDeniedExceptionError { // fallback in case of missing permissions
				l.Printf("MPA validation fallback for account %s, missing permission 'iam:GetPolicy'. error: %s", req.AccountID, awsErr.Error())
				return nil
			}
		}

		// fails if - has iam:GetPolicy permission but fail to validate the given policy
		if strings.Contains(err.Error(), errorMissingPermissions) {
			return err
		}

		l.Error(err)

		return errorPolicy
	}

	return nil
}

// ensureNoOrganizationAccountAccessInFirestore sets the features.no-assume-role in Firestore for the given MPA account ID
//
// WARN: you may want to make this function more generic to account for all future features to set in FS
func (s *MasterPayerAccountService) ensureNoOrganizationAccountAccessInFirestore(accountID string, permissions *PolicyPermissions) error {
	ctx := context.Background()

	if permissions == nil {
		err := fmt.Errorf("provided permissions were nil")
		s.loggerProvider(ctx).Error(err)

		return err
	}

statementLoop:
	for i, statement := range permissions.Statement {
		if slices.Contains(toStringSlice(statement.Resource), "arn:aws:iam::*:role/OrganizationAccountAccessRole") {
			for _, action := range toStringSlice(statement.Action) {
				if action == "sts:AssumeRole" {
					break statementLoop
				}
			}
			// if no sts:AssumeRole action has been found in the statements, set the no-assume-role feature in Firestore.
			if i == len(permissions.Statement)-1 {
				if err := s.mpaDAL.UpdateMPAField(ctx, accountID, []firestore.Update{
					{Path: "features.no-assume-role", Value: true},
				}); err != nil {
					err = fmt.Errorf("failed to update mpa %s with noAssumeRole: %w", accountID, err)
					s.loggerProvider(ctx).Error(err)
					return err
				}
			}
		}
	}

	return nil
}

// validatePolicyHasRequiredPermissions compares given policy with DoiT's policy stored in a json file
func (s *MasterPayerAccountService) validatePolicyHasRequiredPermissions(accountID, curBucket, policyDocument string) error {
	// get the actual permissions from the policy document
	actualPolicyBytes := []byte(policyDocument)
	actualPermissions := new(PolicyPermissions)

	if err := json.Unmarshal(actualPolicyBytes, actualPermissions); err != nil {
		return err
	}

	// get the expected permissions from the DoiT managed template
	expectedPermissions, err := s.getExpectedPermissionsFromTemplate(accountID, curBucket)
	if err != nil {
		return err
	}

	// get the optional permissions mask from the DoiT managed mask template
	optionalPermissions, err := s.getOptionalPermissionsFromTemplate(accountID, curBucket)
	if err != nil {
		return err
	}

	// apply the permissions mask to both actual and expected permissions before comparing them
	actualPermissionsMasked := applyPermissionMask(actualPermissions, optionalPermissions)
	expectedPermissionsMasked := applyPermissionMask(expectedPermissions, optionalPermissions)

	if !cmp.Equal(expectedPermissionsMasked, actualPermissionsMasked, cmpopts.IgnoreUnexported(PolicyPermissions{})) {
		return fmt.Errorf("%s: %s", errorMissingPermissions, cmp.Diff(expectedPermissionsMasked, actualPermissionsMasked, cmpopts.IgnoreUnexported(PolicyPermissions{})))
	}

	return s.ensureNoOrganizationAccountAccessInFirestore(accountID, actualPermissions)
}

// getExpectedPermissionsFromTemplate returns a pointer to a PolicyPermissions object, read from the template file on our filesystem.
func (s *MasterPayerAccountService) getExpectedPermissionsFromTemplate(accountID, curBucket string) (*PolicyPermissions, error) {
	if s.doitPolicyTemplateFile == nil {
		return nil, fmt.Errorf("doitPolicyTemplateFile was nil")
	}

	return getPolicyPermissionsFromTemplate(*s.doitPolicyTemplateFile, accountID, curBucket)
}

// getOptionalPermissionsFromTemplate returns a pointer to a PolicyPermissions object, read from the template file on our filesystem.
//
// WARN: this is where you can introduce another function to get the masks from a FS collection instead.
func (s *MasterPayerAccountService) getOptionalPermissionsFromTemplate(accountID, curBucket string) (*PolicyPermissions, error) {
	if s.doitPolicyOptionalPermissionsMaskFile == nil {
		// no provided mask, which can be expected
		return nil, nil
	}

	return getPolicyPermissionsFromTemplate(*s.doitPolicyOptionalPermissionsMaskFile, accountID, curBucket)
}

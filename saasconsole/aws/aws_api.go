package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go/service/organizations"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
)

// getReportsDefinitions if not succesful returns error and whether to retry (true if not retry, false if retry)
func (s *AWSSaaSConsoleOnboardService) getReportsDefinitions(accountID string) ([]*costandusagereportservice.ReportDefinition, bool, error) {
	session, err := s.awsAccessService.GetAWSSession(accountID, access.FunctionDescribeReportDefinitions)
	if err != nil {
		return nil, false, s.handleAWSError(err)
	}

	config := aws.Config{Region: aws.String(endpoints.UsEast1RegionID)}
	curService := costandusagereportservice.New(session, &config)

	output, err := curService.DescribeReportDefinitions(nil)
	if err != nil {
		return nil, false, s.handleAWSError(err)
	}

	if output == nil || len(output.ReportDefinitions) < 1 {
		return nil, true, saasconsole.BuildStandaloneError(pkg.OnboardingErrorTypeCUR, errorCUR)
	}

	return output.ReportDefinitions, true, nil
}

/*
	validateSingleCUR

conditions:
- bucket —> report's S3Bucket = ‘s3Bucket’ (given by the user)
- path —> report's S3Prefix = ‘curPath’ (given by the user) OR report's ReportName = last part of ‘curPath’ (given by the user)
- properties —> report's TimeUnit = hourly AND report's Format = CSV AND report's AdditionalSchemaElements contains RESOURCES

examples:
  - report prefix = "CUR", name = "doitintl-awsops102".
    Valid curPaths:
  - prefix "CUR"
  - name "doitintl-awsops102"
  - full path "CUR/doitintl-awsops102"
  - report prefix = "CUR/doitintl-awsops102", name = "doitintl-awsops102".
    valid curPaths:
  - prefix "CUR/doitintl-awsops102"
  - name "doitintl-awsops102"
  - full path "CUR/doitintl-awsops102/doitintl-awsops102"
*/
func (s *AWSSaaSConsoleOnboardService) validateSingleCUR(report *costandusagereportservice.ReportDefinition, s3Bucket string) (bool, *pkg.AWSSaaSConsoleCURPath) {
	if *report.S3Bucket != s3Bucket {
		return false, nil
	}

	path := pkg.AWSSaaSConsoleCURPath{
		Bucket:     *report.S3Bucket,
		ReportName: *report.ReportName,
		PathPrefix: *report.S3Prefix,
	}

	if *report.TimeUnit != mpa.TimeGranularityHourly {
		path.Granularity = *report.TimeUnit
	}

	if compression, ok := mpa.ReportFormats[*report.Format]; ok {
		if compression != *report.Compression {
			path.Format = *report.Format
		}
	}

	validSchema := false

	for _, element := range report.AdditionalSchemaElements {
		if *element == mpa.SchemaElementsResources {
			validSchema = true
		}
	}

	if !validSchema {
		path.NoResource = true
	}

	if path.Format != "" || path.Granularity != "" || path.NoResource {
		return false, &path
	}

	return true, &path
}

// getAssetProperties populate asset properties given AWS account
func (s *AWSSaaSConsoleOnboardService) getAssetProperties(accountID string) (*assets.AWSProperties, error) {
	session, err := s.awsAccessService.GetAWSSession(accountID, access.FunctionDescribeAccount)
	if err != nil {
		return nil, s.handleAWSError(err)
	}

	organizationsService := organizations.New(session)
	describeAccountInput := &organizations.DescribeAccountInput{
		AccountId: aws.String(accountID),
	}

	output, err := organizationsService.DescribeAccount(describeAccountInput)
	if err != nil {
		return nil, s.handleAWSError(err)
	}

	if output == nil || output.Account == nil {
		return nil, errorEmptyDescribeAccount
	}

	awsAccount := output.Account

	return &assets.AWSProperties{
		AccountID:    *awsAccount.Id,
		Name:         *awsAccount.Name,
		FriendlyName: *awsAccount.Id,
		OrganizationInfo: &assets.OrganizationInfo{
			PayerAccount: &domain.PayerAccount{
				AccountID:   *awsAccount.Id,
				DisplayName: s.getPayerAccountName(*awsAccount.Id),
			},
			Status: *awsAccount.Status,
			Email:  *awsAccount.Email,
		},
	}, nil
}

// handleAWSError detects pre-known aws error types
func (s *AWSSaaSConsoleOnboardService) handleAWSError(err error) error {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		switch string(awsErr.Code()) {
		case "AccessDenied", "AccessDeniedException":
			return saasconsole.BuildStandaloneError(pkg.OnboardingErrorTypePermissions, err)
		case "ValidationException":
			if strings.Contains(awsErr.Error(), "LINKED") {
				return saasconsole.BuildStandaloneError(pkg.OnboardingErrorTypeLinkedAccount, err)
			}
		default:
			return err
		}
	}

	return err
}

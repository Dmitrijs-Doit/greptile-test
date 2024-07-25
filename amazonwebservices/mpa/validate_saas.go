package mpa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const (
	logPrefix string = "SaaS Console - AWS Permissions Validator - "

	timeBetweenAlerts = 3 * 24 * time.Hour
)

var (
	criticalPermissions = []string{
		"sts:AssumeRole",
		"s3:GetObject",
		"s3:ListBucket",
		"organizations:ListAccounts*",
		"organizations:ListHandshakes*",
		"organizations:DescribeOrganization",
	}
)

// ValidateSaaS validates SaaS account to clarify it is accessible from our platform
func (s *MasterPayerAccountService) ValidateSaaS(ctx context.Context, req *ValidateSaaSRequest) error {
	l := s.loggerProvider(ctx)

	status := &pkg.AWSCloudConnectStatus{
		Status: pkg.AWSCloudConnectStatusValid,
	}

	customerRef := s.customersDAL.GetRef(ctx, req.CustomerID)

	oldCloudConnect, err := s.cloudConnectDAL.GetAWSCloudConnect(ctx, customerRef, common.Assets.AmazonWebServices, req.AccountID)
	if err != nil {
		l.Errorf("%scouldn't get cloud connect doc for customer %s, account id: %s, %s", logPrefix, req.CustomerID, req.AccountID, err)
	}

	// validate cur definition only for non-custom account onboard
	if customerCurInfo, ok := saasconsole.AWSCustomCustomerCurMap[req.CustomerID]; ok {
		if _, ok := customerCurInfo[req.AccountID]; !ok {
			s.setCURStatus(ctx, req.AccountID, req.CURBucket, req.CURPath, status)
		}
	} else {
		s.setCURStatus(ctx, req.AccountID, req.CURBucket, req.CURPath, status)
	}

	s.setRoleStatus(ctx, req.AccountID, req.RoleArn, status)

	s.setPoliciesStatus(ctx, req.AccountID, req.RoleArn, req.CURBucket, status, oldCloudConnect)

	if err := s.sendAlert(ctx, customerRef, req.CustomerID, req.AccountID, status, oldCloudConnect.StatusInfo); err != nil {
		l.Errorf("%scouldn't publish slack notification, %s", logPrefix, err)
	}

	if err := s.cloudConnectDAL.SetAWSCloudConnectConnectionStatus(ctx, customerRef, common.Assets.AmazonWebServices, req.AccountID, status); err != nil {
		l.Errorf("%scouldn't update cloud connect doc with account status, %s", logPrefix, err)
		return err
	}

	return nil
}

func (s *MasterPayerAccountService) setCURStatus(ctx context.Context, accountID, bucket, path string, status *pkg.AWSCloudConnectStatus) {
	l := s.loggerProvider(ctx)

	if err := s.validateCUR(accountID, bucket, path); err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if errCode := awsErr.Code(); errCode == accessDeniedError || errCode == accessDeniedExceptionError { // fallback in case of missing permissions
				l.Printf("SaaS validation fallback for account %s, missing permission 'cur:DescribeReportDefinitions'. error: %s", accountID, awsErr.Error())
				err = s.validateCURFileExistence(accountID, bucket, path)
			}
		}

		status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{}

		if err != nil {
			for _, curError := range curErrors {
				if err == curError {
					status.InvalidInfo.CURError = err.Error()
					break
				}
			}

			if status.Status == pkg.AWSCloudConnectStatusValid {
				l.Error(err)
				status.InvalidInfo.CURError = purifyError(err)
			}

			status.Status = pkg.AWSCloudConnectStatusCritical
		}
	}
}

func (s *MasterPayerAccountService) setRoleStatus(ctx context.Context, accountID, roleArn string, status *pkg.AWSCloudConnectStatus) {
	l := s.loggerProvider(ctx)

	if missingPermissions, err := s.validateRole(accountID, roleArn, true); err != nil {
		l.Errorf("%sfailed to validate role for account id: %s, role arn: %s with error: %s", logPrefix, accountID, roleArn, err)

		status.Status = pkg.AWSCloudConnectStatusPartial
		if err == errorRole {
			status.Status = pkg.AWSCloudConnectStatusCritical
		}

		if status.InvalidInfo == nil {
			status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{}
		}

		status.InvalidInfo.RoleError = purifyError(err)
	} else if len(missingPermissions) > 0 {
		status.Status = pkg.AWSCloudConnectStatusCritical

		if status.InvalidInfo == nil {
			status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{}
		}

		status.InvalidInfo.MissingPermissions = missingPermissions
	}
}

func (s *MasterPayerAccountService) setPoliciesStatus(ctx context.Context, accountID, roleArn, bucket string, status *pkg.AWSCloudConnectStatus, oldCloudConnect *pkg.AWSCloudConnect) {
	l := s.loggerProvider(ctx)

	missingPermissions, err := s.validatePolicy(ctx, accountID, roleArn, bucket, true, &oldCloudConnect.TimeCreated)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			if errCode := awsErr.Code(); errCode == accessDeniedError || errCode == accessDeniedExceptionError {
				if len(missingPermissions) > 0 {
					missingPermissions = []string{fmt.Sprintf("%s~iam::%s:policy/doitintl_cmp", missingPermissions[0], accountID)}
				}
			} else {
				l.Errorf("%scouldn't get policy for account id: %s, %s", logPrefix, accountID, err)
			}
		}

		status.Status = pkg.AWSCloudConnectStatusPartial
		if err == errorPolicy {
			status.Status = pkg.AWSCloudConnectStatusCritical
		}

		if status.InvalidInfo == nil {
			status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{}
		}

		status.InvalidInfo.PolicyError = purifyError(err)
	}

	if len(missingPermissions) > 0 {
		if status.Status != pkg.AWSCloudConnectStatusCritical {
			status.Status = pkg.AWSCloudConnectStatusPartial

			for _, p := range missingPermissions {
				action := strings.Split(p, "~")[0]
				if slice.Contains(criticalPermissions, action) {
					status.Status = pkg.AWSCloudConnectStatusCritical
					break
				}
			}
		}

		if status.InvalidInfo == nil {
			status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{}
		}

		status.InvalidInfo.MissingPermissions = append(status.InvalidInfo.MissingPermissions, missingPermissions...)
	}
}

// validateSaaSPolicyHasRequiredPermissions compares given policy with DoiT's policy stored in a json file
func (s *MasterPayerAccountService) validateSaaSPolicyHasRequiredPermissions(ctx context.Context, accountID, bucket, policyDocument string, timeCreated *time.Time) ([]string, error) {
	l := s.loggerProvider(ctx)

	requiredPermissions, err := s.getRequiredSaaSPolicyPermissionsFromTemplate(accountID, bucket)
	if err != nil {
		return []string{}, err
	}

	// If any permissions are optional based on the policy's creation date, remove them
	optionalPermissions := getConditionalOptionalPermissionsFromTemplate(l, *s.saasDoitPolicyConditionalPermissionsTemplateFile, *timeCreated)
	if optionalPermissions != nil {
		for _, action := range getSaaSAccountPolicyActions(*optionalPermissions, accountID, bucket) {
			delete(requiredPermissions, action)
		}
	}

	policyByte := []byte(policyDocument)

	var policyPermissions PolicyPermissions

	if err := json.Unmarshal(policyByte, &policyPermissions); err != nil {
		return []string{}, err
	}

	for _, action := range getSaaSAccountPolicyActions(policyPermissions, accountID, bucket) {
		requiredPermissions[action] = true
	}

	missingPermissions := []string{}

	for permissionName, permission := range requiredPermissions {
		if !permission {
			missingPermissions = append(missingPermissions, strings.Replace(permissionName, "arn:aws:", "", -1))
		}
	}

	return missingPermissions, nil
}

func (s *MasterPayerAccountService) validateSaaSRoleHasRequiredPermissions(roleDocument string) ([]string, error) {
	requiredPermissions, err := s.getRequiredSaaSRolePermissionsFromTemplate()
	if err != nil {
		return []string{}, err
	}

	roleByte := []byte(roleDocument)

	var rolePermissions RolePermissions

	if err := json.Unmarshal(roleByte, &rolePermissions); err != nil {
		return []string{}, err
	}

	for _, action := range getSaaSAccountRoleActions(rolePermissions) {
		requiredPermissions[action] = true
	}

	missingPermissions := []string{}

	for permissionName, permission := range requiredPermissions {
		if !permission {
			missingPermissions = append(missingPermissions, strings.Replace(permissionName, "arn:aws:", "", -1))
		}
	}

	return missingPermissions, nil
}

func (s *MasterPayerAccountService) getRequiredSaaSPolicyPermissionsFromTemplate(accountID, bucket string) (map[string]bool, error) {
	fileByte, err := getDoitPolicyFromTemplate(*s.saasDoitPolicyTemplateFile)
	if err != nil {
		return nil, err
	}

	var policyPermissions PolicyPermissions
	if err := json.Unmarshal(fileByte, &policyPermissions); err != nil {
		return nil, err
	}

	permissions := map[string]bool{}
	for _, action := range getSaaSAccountPolicyActions(policyPermissions, accountID, bucket) {
		permissions[action] = false
	}

	return permissions, nil
}

func (s *MasterPayerAccountService) getRequiredSaaSRolePermissionsFromTemplate() (map[string]bool, error) {
	fileByte, err := getDoitPolicyFromTemplate(*s.saasDoitRoleTemplateFile)
	if err != nil {
		return nil, err
	}

	var rolePermissions RolePermissions
	if err := json.Unmarshal(fileByte, &rolePermissions); err != nil {
		return nil, err
	}

	permissions := map[string]bool{}
	for _, action := range getSaaSAccountRoleActions(rolePermissions) {
		permissions[action] = false
	}

	return permissions, nil
}

func getSaaSAccountPolicyActions(policy PolicyPermissions, accountID, bucket string) []string {
	permissions := []string{}

	for _, statement := range policy.Statement {
		actions := getStatementActions(statement.Action)

		for _, action := range actions {
			permissions = append(permissions, getActionResources(action, statement.Resource, accountID, bucket)...)
		}
	}

	return permissions
}

func getSaaSAccountRoleActions(policy RolePermissions) []string {
	permissions := []string{}

	for _, statement := range policy.Statement {
		actions := getStatementActions(statement.Action)

		for _, action := range actions {
			if principalMap, ok := statement.Principal.(map[string]interface{}); ok {
				for key, value := range principalMap {
					permissions = append(permissions, fmt.Sprintf("%s~%s~%s", action, key, value.(string)))
				}
			}
		}
	}

	return permissions
}

func getStatementActions(statementAction interface{}) []string {
	actions := []string{}

	if action, ok := statementAction.(string); ok {
		actions = append(actions, action)
	} else if actionsArr, ok := statementAction.([]string); ok {
		actions = append(actions, actionsArr...)
	} else if actionsArr, ok := statementAction.([]interface{}); ok {
		for _, item := range actionsArr {
			actions = append(actions, item.(string))
		}
	}

	return actions
}

func getActionResources(action string, statementResource interface{}, accountID, bucket string) []string {
	resources := []string{}

	if resource, ok := statementResource.(string); ok {
		resources = append(resources, fmt.Sprintf("%s~%s", action, replaceResourceWithAccountValues(resource, accountID, bucket)))
	} else if resourceArr, ok := statementResource.([]string); ok {
		for _, item := range resourceArr {
			resources = append(resources, fmt.Sprintf("%s~%s", action, replaceResourceWithAccountValues(item, accountID, bucket)))
		}
	} else if resourceArr, ok := statementResource.([]interface{}); ok {
		for _, item := range resourceArr {
			resources = append(resources, fmt.Sprintf("%s~%s", action, replaceResourceWithAccountValues(item.(string), accountID, bucket)))
		}
	}

	return resources
}

func replaceResourceWithAccountValues(resource, accountID, bucket string) string {
	resource = strings.ReplaceAll(resource, "SAAS_S3_CUR_BUCKET", bucket)
	return strings.ReplaceAll(resource, "SAAS_ACCOUNT_ID", accountID)
}

func purifyError(err error) string {
	return strings.Replace(strings.Replace(strings.Replace(err.Error(), "arn:aws:", "", -1), "\n", ". ", -1), "\t", " ", -1)
}

func (s *MasterPayerAccountService) sendAlert(ctx context.Context, customerRef *firestore.DocumentRef, customerID, accountID string, status *pkg.AWSCloudConnectStatus, oldStatus *pkg.AWSCloudConnectStatus) error {
	alertStatus := s.getStatusForAlert(oldStatus, status)
	now := time.Now().UTC()

	if status.InvalidInfo == nil {
		status.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{}
	}

	if oldStatus != nil && oldStatus.InvalidInfo != nil {
		status.InvalidInfo.AlertSentAt = oldStatus.InvalidInfo.AlertSentAt
	}

	if s.shouldSendAlert(ctx, oldStatus, alertStatus, now) {
		if err := saasconsole.PublishPermissionsAlertSlackNotification(ctx, pkg.AWS, s.customersDAL, customerID, accountID, alertStatus, criticalPermissions); err != nil {
			return err
		}

		status.InvalidInfo.AlertSentAt = now
	}

	return nil
}

func (s *MasterPayerAccountService) getStatusForAlert(oldStatus, newStatus *pkg.AWSCloudConnectStatus) *pkg.AWSCloudConnectStatus {
	alertStatus := *newStatus

	if oldStatus != nil && oldStatus.IgnorePolicies {
		newStatus.IgnorePolicies = true

		alertStatus.InvalidInfo = &pkg.AWSCloudConnectInvalidInfo{
			CURError:    newStatus.InvalidInfo.CURError,
			RoleError:   newStatus.InvalidInfo.RoleError,
			PolicyError: newStatus.InvalidInfo.PolicyError,
		}
		alertStatus.InvalidInfo.MissingPermissions = append(alertStatus.InvalidInfo.MissingPermissions, newStatus.InvalidInfo.MissingPermissions...)

		removeGetPolicyVersionPermission(oldStatus)
		removeGetPolicyVersionPermission(&alertStatus)
	}

	return &alertStatus
}

func removeGetPolicyVersionPermission(status *pkg.AWSCloudConnectStatus) {
	missing := status.InvalidInfo.MissingPermissions

	for i, p := range missing {
		if strings.Split(p, "~")[0] == "iam:GetPolicyVersion" {
			missing = append(missing[:i], missing[i+1:]...)
		}
	}

	if len(missing) == 0 && status.InvalidInfo.CURError == "" && status.InvalidInfo.RoleError == "" {
		status.Status = pkg.AWSCloudConnectStatusValid
	}

	status.InvalidInfo.PolicyError = ""
	status.InvalidInfo.MissingPermissions = missing
}

func (s *MasterPayerAccountService) shouldSendAlert(ctx context.Context, oldStatus, newStatus *pkg.AWSCloudConnectStatus, now time.Time) bool {
	if newStatus.Status == pkg.AWSCloudConnectStatusValid {
		return false
	}

	if oldStatus == nil ||
		oldStatus.Status != newStatus.Status ||
		oldStatus.InvalidInfo == nil || newStatus.InvalidInfo == nil || newStatus.InvalidInfo.CURError != oldStatus.InvalidInfo.CURError ||
		oldStatus.InvalidInfo.RoleError != newStatus.InvalidInfo.RoleError ||
		oldStatus.InvalidInfo.PolicyError != newStatus.InvalidInfo.PolicyError ||
		len(oldStatus.InvalidInfo.MissingPermissions) != len(newStatus.InvalidInfo.MissingPermissions) ||
		now.Sub(oldStatus.InvalidInfo.AlertSentAt) > timeBetweenAlerts {
		return true
	}

	for _, p := range newStatus.InvalidInfo.MissingPermissions {
		if !slice.Contains(oldStatus.InvalidInfo.MissingPermissions, p) {
			return true
		}
	}

	return false
}

package aws

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
)

type AWSSaaSConsoleInitRequest struct {
	AccountID string `json:"accountId"`
}

type AWSSaaSConsoleCURDiscoveryRequest struct {
	CustomerID string `json:"external_id"`
	Arn        string `json:"management_arn"`
	AccountID  string `json:"account_id"`
	StackID    string `json:"stack_id"`
	StackName  string `json:"stack_name"`
	S3Bucket   string `json:"s3_bucket"`
}

type AWSSaaSConsoleCURRefreshRequest struct {
	CustomerID string `json:"customerId"`
	AccountID  string `json:"accountId"`
}

type AWSSaaSConsoleActivateRequest struct {
	CustomerID   string `json:"customerId"`
	AccountID    string `json:"accountId"`
	CURPathIndex int    `json:"curPathIndex"`
}

const (
	stackDeletion = "AWS initiated Stack deletion"

	// steps log messages
	stepMessageCURDiscoveryStarted    = "cur discovery"
	stepMessageCURDiscoveryCompleted  = "cur discovery completed successfully"
	stepMessageCURRefreshStarted      = "cur refresh started"
	stepMessageCURRefreshCompleted    = "cur refresh completed"
	stepMessageUpdateBillingStarted   = "update-billing started"
	stepMessageUpdateBillingCompleted = "update-billing completed successfully"

	doitSubString string = "doit"

	stackDeletedRoleError   string = "Stack deleted by customer - no valid Role defined"
	stackDeletedPolicyError string = "Stack deleted by customer - no valid Policy defined"
)

var (
	requetsFields = []string{"account id", "customer id", "s3 bucket"}

	// errors
	errorBadAccountID         = errors.New("customer's account id does not correlates the one supplied from AWS")
	errorEmptyDescribeAccount = errors.New("empty account response returned from AWS")
	errorCUR                  = errors.New("no valid Cost And Usage Report found for AWS account")
	errorInvalidCURIndex      = errors.New("invalid Cost And Usage Report path index")
)

func (s *AWSSaaSConsoleOnboardService) getDocumentID(customerID string) string {
	return pkg.GetDocumentID(customerID, pkg.AWS)
}

func (s *AWSSaaSConsoleOnboardService) composeStandaloneID(customerID, accountID string) string {
	return pkg.ComposeStandaloneID(customerID, accountID, pkg.AWS)
}

// amazon-web-services-standalone...
func (s *AWSSaaSConsoleOnboardService) getAssetID(accountID string) string {
	return saasconsole.GetAssetID(pkg.AWS, accountID)
}

func getRoleArn(role, accountID string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, role)
}

func (s *AWSSaaSConsoleOnboardService) getPayerAccountName(ID string) string {
	return fmt.Sprintf("standalone-payer-%s", ID)
}

func (s *AWSSaaSConsoleOnboardService) extractCUR(curPath string) string {
	pathParts := strings.Split(curPath, "/")
	len := len(pathParts)

	return pathParts[len-1]
}

func (s *AWSSaaSConsoleOnboardService) getLogger(ctx context.Context, customerID string) logger.ILogger {
	logger := s.loggerProvider(ctx)
	saasconsole.EnrichLogger(logger, customerID, pkg.AWS)

	return logger
}

// enableCustomer set enabledSaaSConsole.AWS on customer document
func (s *AWSSaaSConsoleOnboardService) enableCustomer(ctx context.Context, customerID string) error {
	return saasconsole.EnableCustomer(ctx, customerID, pkg.AWS, s.customersDAL)
}

func (s *AWSSaaSConsoleOnboardService) ParseInitRequest(ctx *gin.Context, customerID string) (*AWSSaaSConsoleInitRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepInit

	var req AWSSaaSConsoleInitRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return nil, s.updateAndSlackError(ctx, customerID, req.AccountID, step, err)
	}

	if err := s.validateFields(req.AccountID); err != nil {
		return nil, s.updateAndSlackError(ctx, customerID, req.AccountID, step, err)
	}

	return &req, saasconsole.Success
}

func (s *AWSSaaSConsoleOnboardService) ParseContractRequest(ctx *gin.Context) (*saasconsole.StandaloneContractRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepContract

	req, err := saasconsole.ParseContractRequest(ctx)
	if err != nil {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	if err := s.validateFields(req.AccountID, req.CustomerID); err != nil {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	return req, saasconsole.Success
}

func (s *AWSSaaSConsoleOnboardService) ParseCURDiscoveryRequest(ctx *gin.Context, stackDeletion bool) (*AWSSaaSConsoleCURDiscoveryRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepCURDiscovery

	var req AWSSaaSConsoleCURDiscoveryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		if !stackDeletion {
			return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
		}

		return nil, saasconsole.Failure
	}

	if err := s.validateFields(req.AccountID, req.CustomerID, req.S3Bucket); err != nil {
		if !stackDeletion {
			return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
		}

		return nil, saasconsole.Failure
	}

	return &req, saasconsole.Success
}

func (s *AWSSaaSConsoleOnboardService) ParseCURRefreshRequest(ctx *gin.Context) (*AWSSaaSConsoleCURRefreshRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepCURRefresh

	var req AWSSaaSConsoleCURRefreshRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	if err := s.validateFields(req.AccountID, req.CustomerID); err != nil {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	return &req, saasconsole.Success
}

func (s *AWSSaaSConsoleOnboardService) ParseActivateRequest(ctx *gin.Context) (*AWSSaaSConsoleActivateRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepActivation

	var req AWSSaaSConsoleActivateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	if req.CURPathIndex < 0 {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, errorInvalidCURIndex)
	}

	if err := s.validateFields(req.AccountID, req.CustomerID); err != nil {
		return nil, s.updateAndSlackError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	return &req, saasconsole.Success
}

// validateFields validates mandatory fields has received from the request
func (s *AWSSaaSConsoleOnboardService) validateFields(fields ...string) error {
	for i, f := range fields {
		if len(f) == 0 {
			return saasconsole.GetMissingError(requetsFields[i])
		}
	}

	return nil
}

func (s *AWSSaaSConsoleOnboardService) updateAndSlackError(ctx context.Context, customerID, accountID string, step pkg.StandaloneOnboardingStep, originalError error) *saasconsole.OnboardingResponse {
	s.updateError(ctx, customerID, accountID, step, originalError)

	_ = saasconsole.PublishOnboardErrorSlackNotification(ctx, pkg.AWS, s.customersDAL, customerID, accountID, originalError)

	return saasconsole.Failure
}

// handleError prints & updates error on standalone fs document
func (s *AWSSaaSConsoleOnboardService) updateError(ctx context.Context, customerID, accountID string, step pkg.StandaloneOnboardingStep, err error) {
	logger := s.getLogger(ctx, customerID)

	standaloneID := ""
	if accountID != "" { //	only pass standaloneID when accountID is known
		standaloneID = s.composeStandaloneID(customerID, accountID)
	}

	saasconsole.UpdateError(ctx, logger, pkg.AWS, standaloneID, step, err, s.saasConsoleDAL.UpdateAWSOnboardingError)
}

func (s *AWSSaaSConsoleOnboardService) validateCustomerAccountID(ctx context.Context, customerID, accountID string) error {
	doc, err := s.saasConsoleDAL.GetAWSOnboarding(ctx, s.composeStandaloneID(customerID, accountID))
	if err != nil {
		return err
	}

	if doc.AccountID != accountID {
		return errorBadAccountID
	}

	return nil
}

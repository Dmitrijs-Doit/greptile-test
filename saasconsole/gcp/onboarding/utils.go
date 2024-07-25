package onboarding

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"

	billing "cloud.google.com/go/billing/apiv1"
	"cloud.google.com/go/iam/apiv1/iampb"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
)

type GCPStandaloneRequest struct {
	CustomerID       string `json:"customerId"`
	BillingAccountID string `json:"billingAccountId,omitempty"`
	ProjectID        string `json:"projectId,omitempty"`
	DatasetID        string `json:"datasetId,omitempty"`
	DryRun           bool   `json:"dryRun,omitempty"`
}

const (
	billingAccountID = "billing account id"
	orgID            = "org id"
	projectID        = "project id"
	tableID          = "table id"
	datasetID        = "dataset id"

	// steps logs
	stepMessageActivateStarted   = "activate started"
	stepMessageActivateCompleted = "activate completed successfully"

	// GCP permissions
	roleBillingViewer = "roles/billing.viewer"

	cloudBillingScope  = "https://www.googleapis.com/auth/cloud-billing"
	cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"
)

var (
	// permissions
	rolesBilling = []string{roleBillingViewer}

	// errors
	errorServiceAccount = errors.New("empty service account")
)

func (s *GCPSaaSConsoleOnboardService) getDocumentID(customerID string) string {
	return pkg.GetDocumentID(customerID, pkg.GCP)
}

// google-cloud-standalone...
func (s *GCPSaaSConsoleOnboardService) getAssetID(billingAccountID string) string {
	return saasconsole.GetAssetID(pkg.GCP, billingAccountID)
}

// EnableCustomer set enabledSaaSConsole.GCP on customer document
func (s *GCPSaaSConsoleOnboardService) enableCustomer(ctx context.Context, customerID string) error {
	return saasconsole.EnableCustomer(ctx, customerID, pkg.GCP, s.customersDAL)
}

func (s *GCPSaaSConsoleOnboardService) getLogger(ctx context.Context) logger.ILogger {
	customerID, ok := ctx.Value(saasconsole.CustomerIDKey).(string)
	if !ok {
		customerID = ""
	}

	logger := s.loggerProvider(ctx)
	saasconsole.EnrichLogger(logger, customerID, pkg.GCP)

	return logger
}

// ParseRequest
func (s *GCPSaaSConsoleOnboardService) ParseRequest(ctx *gin.Context) (*GCPStandaloneRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepActivation

	var req GCPStandaloneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return nil, s.updateAndSlackError(ctx, step, "", err)
	}

	if len(req.CustomerID) == 0 {
		return nil, s.updateAndSlackError(ctx, step, req.BillingAccountID, saasconsole.ErrorCustomerID)
	}

	ctxWithCustomerID := context.WithValue(ctx, saasconsole.CustomerIDKey, req.CustomerID)

	if len(req.BillingAccountID) == 0 {
		return nil, s.updateAndSlackError(ctxWithCustomerID, step, req.BillingAccountID, saasconsole.GetMissingError(tableID))
	}

	if len(req.ProjectID) == 0 {
		return nil, s.updateAndSlackError(ctxWithCustomerID, step, req.BillingAccountID, saasconsole.GetMissingError(projectID))
	}

	if len(req.DatasetID) == 0 {
		return nil, s.updateAndSlackError(ctxWithCustomerID, step, req.BillingAccountID, saasconsole.GetMissingError(datasetID))
	}

	return &req, saasconsole.Success
}

func (s *GCPSaaSConsoleOnboardService) ParseContractRequest(ctx *gin.Context) (*saasconsole.StandaloneContractRequest, *saasconsole.OnboardingResponse) {
	step := pkg.OnboardingStepContract

	req, err := saasconsole.ParseContractRequest(ctx)
	ctxWithValue := context.WithValue(ctx, saasconsole.CustomerIDKey, req.CustomerID)

	if err != nil {
		return nil, s.updateAndSlackError(ctxWithValue, step, req.AccountID, err)
	}

	if err := saasconsole.ValidateContractRequest(req); err != nil {
		return nil, s.updateAndSlackError(ctxWithValue, step, req.AccountID, err)
	}

	return req, saasconsole.Success
}

// updateAndSlackError prints & updates error on standalone fs document, and returns 200 with { success: false } payload
func (s *GCPSaaSConsoleOnboardService) updateAndSlackError(ctx context.Context, step pkg.StandaloneOnboardingStep, billingAccountID string, originalError error) *saasconsole.OnboardingResponse {
	customerID, ok := ctx.Value(saasconsole.CustomerIDKey).(string)
	if !ok {
		customerID = ""
	}

	_ = s.updateError(ctx, customerID, billingAccountID, step, originalError)

	_ = saasconsole.PublishOnboardErrorSlackNotification(ctx, pkg.GCP, s.customersDAL, customerID, billingAccountID, originalError)

	return saasconsole.Failure
}

// updateAndSlackError prints & updates error on standalone fs document, and returns 200 with { success: false } payload
func (s *GCPSaaSConsoleOnboardService) updateError(ctx context.Context, customerID, billingAccountID string, step pkg.StandaloneOnboardingStep, originalError error) *saasconsole.OnboardingResponse {
	logger := s.getLogger(ctx)

	standaloneID := pkg.ComposeStandaloneID(customerID, billingAccountID, pkg.GCP)

	saasconsole.UpdateError(ctx, logger, pkg.GCP, standaloneID, step, originalError, s.saasConsoleDAL.UpdateGCPOnboardingError)

	return saasconsole.Failure
}

func (s *GCPSaaSConsoleOnboardService) runAllTests(ctx context.Context, req *GCPStandaloneRequest, doc *pkg.GCPSaaSConsoleAccounts) error {
	if req.CustomerID != "fOjoMysJgzX0eP2kKZnN" { // google.com - remove once onboarded
		permissionsGrp, _ := errgroup.WithContext(ctx)

		permissionsGrp.Go(func() error {
			ok, err := s.testIamPermissions(ctx, req.BillingAccountID, doc.ServiceAccountEmail, rolesBilling)
			if err != nil {
				return err
			}

			if !ok {
				return errors.New("missing permissions")
			}

			return nil
		})

		if err := permissionsGrp.Wait(); err != nil {
			return err
		}
	}

	testingGrp, _ := errgroup.WithContext(ctx)

	testingGrp.Go(func() error {
		return s.billingPipelineService.TestConnection(ctx, req.BillingAccountID, doc.ServiceAccountEmail, &pkg.BillingTablesLocation{
			DetailedProjectID: req.ProjectID,
			DetailedDatasetID: req.DatasetID,
		})
	})

	if err := testingGrp.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *GCPSaaSConsoleOnboardService) runEnablement(ctx context.Context, req *GCPStandaloneRequest, doc *pkg.GCPSaaSConsoleAccounts) error {
	if err := s.serviceAccounts.MarkServiceAccountOnBoardSuccessful(ctx, doc.Customer.ID, doc.ServiceAccountEmail, req.BillingAccountID); err != nil {
		return err
	}

	standaloneID := pkg.ComposeStandaloneID(req.CustomerID, req.BillingAccountID, pkg.GCP)
	if err := s.saasConsoleDAL.CompleteGCPOnboarding(ctx, standaloneID, req.BillingAccountID); err != nil {
		return err
	}

	if err := s.enableCustomer(ctx, req.CustomerID); err != nil {
		return err
	}

	return nil
}

func (s *GCPSaaSConsoleOnboardService) runBillingImport(ctx context.Context, req *GCPStandaloneRequest, doc *pkg.GCPSaaSConsoleAccounts) error {
	if err := s.billingImportStatus.SetStatusPending(ctx, req.CustomerID, req.BillingAccountID); err != nil {
		return err
	}

	if err := s.billingImportStatus.UpdateMaxTimesThresholds(ctx, req.CustomerID, req.BillingAccountID); err != nil {
		return err
	}

	if err := s.billingPipelineService.Onboard(ctx, req.CustomerID, req.BillingAccountID, doc.ServiceAccountEmail,
		&pkg.BillingTablesLocation{
			DetailedProjectID: req.ProjectID,
			DetailedDatasetID: req.DatasetID,
		}); err != nil {
		return err
	}

	return nil
}

func (g *GCPSaaSConsoleOnboardService) cloudBillingServiceWithServiceAccount(ctx context.Context, serviceAccountEmail string) (*billing.CloudBillingClient, error) {
	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: serviceAccountEmail,
		Scopes:          []string{cloudPlatformScope},
	})
	if err != nil {
		return nil, err
	}

	// for now, we are initializing a cloud resource manager service connection on each request
	cloudBilling, err := billing.NewCloudBillingClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, err
	}

	return cloudBilling, nil
}

func (s *GCPSaaSConsoleOnboardService) testIamPermissions(ctx context.Context, billingAccountID, serviceAccountEmail string, roles []string) (bool, error) {
	c, err := s.cloudBillingServiceWithServiceAccount(ctx, serviceAccountEmail)
	if err != nil {
		return false, err
	}
	defer c.Close()

	resource := fmt.Sprintf("billingAccounts/%s", billingAccountID)
	serviceAccountMember := fmt.Sprintf("serviceAccount:%s", serviceAccountEmail)

	req := &iampb.GetIamPolicyRequest{
		Resource: resource,
	}

	resp, err := c.GetIamPolicy(ctx, req)
	if err != nil {
		return false, err
	}

	for _, role := range roles {
		for _, bind := range resp.Bindings {
			if bind.Role == role && slices.Contains(bind.Members, serviceAccountMember) {
				return true, nil
			}
		}
	}

	return false, nil
}

func (g *GCPSaaSConsoleOnboardService) billingServiceWithServiceAccount(ctx context.Context, serviceAccountEmail string) (*cloudbilling.APIService, error) {
	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: serviceAccountEmail,
		Scopes:          []string{cloudBillingScope},
	})
	if err != nil {
		return nil, err
	}

	// for now, we are initializing a billing connection on each request
	billingService, err := cloudbilling.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, err
	}

	return billingService, nil
}

func (s *GCPSaaSConsoleOnboardService) getBillingAccountDisplayName(ctx context.Context, serviceAccountEmail, billingAccountID string) (string, error) {
	logger := s.getLogger(ctx)

	billingService, err := s.billingServiceWithServiceAccount(ctx, serviceAccountEmail)
	if err != nil {
		logger.Errorf("could not retrieve billing service. BA %s %s error [%s]", billingAccountID, serviceAccountEmail, err)
		return "", err
	}

	billingAccount, err := billingService.BillingAccounts.Get(fmt.Sprintf("billingAccounts/%s", billingAccountID)).Do()
	if err != nil {
		logger.Errorf("could not get billing account for ID [%s] error [%s]", billingAccountID, err)
		return "", err
	}

	return billingAccount.DisplayName, nil
}

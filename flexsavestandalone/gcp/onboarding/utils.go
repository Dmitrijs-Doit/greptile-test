package onboarding

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	billingDataStructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/retry"
)

type GCPStandaloneRequest struct {
	CustomerID       string `json:"customerId"`
	OrgID            string `json:"orgId"`
	BillingAccountID string `json:"billingAccountId,omitempty"`
	ProjectID        string `json:"projectId,omitempty"`
	TableID          string `json:"tableId,omitempty"`
	DatasetID        string `json:"datasetId,omitempty"`
	DryRun           bool   `json:"dryRun,omitempty"`
}

type GCPStandaloneResponse struct {
	Success bool `json:"success"`
}

const (
	billingAccountID = "billing account id"
	orgID            = "org id"
	projectID        = "project id"
	tableID          = "table id"
	datasetID        = "dataset id"

	// steps logs
	stepMessageTestEstimationsStarted   = "test-estimations-connection started"
	stepMessageTestEstimationsCompleted = "test-estimations-connection completed successfully"
	stepMessageActivateStarted          = "activate started"
	stepMessageActivateCompleted        = "activate completed successfully"

	// GCP permissions
	permissionRecommendationsList = "recommender.usageCommitmentRecommendations.list"
	billingAccountsGet            = "billing.accounts.get"
	permissionJobsCreate          = "bigquery.jobs.create"
	permissionResourceCreate      = "billing.resourceAssociations.create"
)

var (
	Success = &GCPStandaloneResponse{true}
	Failure = &GCPStandaloneResponse{false}

	// permissions
	permissionsEstimations = []string{permissionRecommendationsList}
	permissionsBilling     = []string{billingAccountsGet, permissionJobsCreate, permissionResourceCreate}

	// errors
	errorServiceAccount         = errors.New("empty service account")
	errorBillingAccount         = errors.New("empty billing account id")
	errorBillingAccountMismatch = errors.New("given billing account id does not correlates the one stored on firestore")
	errorOrgIDMismatch          = errors.New("given organization id does not correlates the one stored on firestore")
	errorEmptySavings           = errors.New("gcp respond with empty savings")
)

func (s *GcpStandaloneOnboardingService) getDocumentID(customerID string) string {
	return pkg.GetDocumentID(customerID, pkg.GCP)
}

// ComposeStandaloneID composes standaloneID given customer & org IDs. this is required in order to address a specific standalone org/account.
// EX: documentID: google-cloud-CUSTOMER_ID, orgID: ORG_ID
// will return: google-cloud-CUSTOMER_ID.accounts.ORG_ID
// leaving nestedID empty will compose documentID (which can address the entire firestore document only)
// ** note: GCP is yet to support multiple standalone accounts thus still using regular documentID convention (fill orgID empty when calling s.composeStandaloneID())
func (s *GcpStandaloneOnboardingService) composeStandaloneID(customerID, orgID string) string {
	// return pkg.ComposeStandaloneID(customerID, orgID, pkg.GCP)	//	TODO FSSA - uncomment to support multiple GCP accounts
	return s.getDocumentID(customerID)
}

// google-cloud-standalone...
func (s *GcpStandaloneOnboardingService) getAssetID(billingAccountID string) string {
	return flexsavestandalone.GetAssetID(pkg.GCP, billingAccountID)
}

func (s *GcpStandaloneOnboardingService) getEntity(ctx context.Context, customerID, standaloneID string) (*firestore.DocumentRef, error) {
	return flexsavestandalone.GetEntity(ctx, customerID, standaloneID, s.entitiesDAL, s.customersDAL.GetCustomer, s.flexsaveStandaloneDAL.GetStandalonePriorityID)
}

// EnableCustomer set enabledFlexsave.GCP on customer document
func (s *GcpStandaloneOnboardingService) enableCustomer(ctx context.Context, customerID string) error {
	return flexsavestandalone.EnableCustomer(ctx, customerID, pkg.GCP, s.customersDAL)
}

func (s *GcpStandaloneOnboardingService) getLogger(ctx context.Context) logger.ILogger {
	customerID, ok := ctx.Value(flexsavestandalone.CustomerIDKey).(string)
	if !ok {
		customerID = ""
	}

	logger := s.loggerProvider(ctx)
	flexsavestandalone.EnrichLogger(logger, customerID, pkg.GCP)

	return logger
}

// ParseRequest (fullPayload for activation endpoint)
func (s *GcpStandaloneOnboardingService) ParseRequest(ctx *gin.Context, fullPayload bool) (*GCPStandaloneRequest, *GCPStandaloneResponse) {
	step := pkg.OnboardingStepSavings
	if fullPayload {
		step = pkg.OnboardingStepActivation
	}

	var req GCPStandaloneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return nil, s.updateError(ctx, step, err)
	}

	if len(req.CustomerID) == 0 {
		return nil, s.updateError(ctx, step, flexsavestandalone.ErrorCustomerID)
	}

	ctxWithCustomerID := context.WithValue(ctx, flexsavestandalone.CustomerIDKey, req.CustomerID)

	if len(req.OrgID) == 0 && !req.DryRun {
		return nil, s.updateError(ctxWithCustomerID, step, flexsavestandalone.GetMissingError(orgID))
	}

	if !fullPayload {
		if len(req.BillingAccountID) == 0 {
			return nil, s.updateError(ctxWithCustomerID, step, flexsavestandalone.GetMissingError(billingAccountID))
		}
	}

	if fullPayload {
		if len(req.ProjectID) == 0 {
			return nil, s.updateError(ctxWithCustomerID, step, flexsavestandalone.GetMissingError(projectID))
		}

		if len(req.TableID) == 0 {
			return nil, s.updateError(ctxWithCustomerID, step, flexsavestandalone.GetMissingError(tableID))
		}

		if len(req.DatasetID) == 0 {
			return nil, s.updateError(ctxWithCustomerID, step, flexsavestandalone.GetMissingError(datasetID))
		}
	}

	return &req, Success
}

// verifyParams verifies correlation of params in the given request with those stored on firestore
func (s *GcpStandaloneOnboardingService) verifyParams(req *GCPStandaloneRequest, doc *pkg.GCPStandaloneOnboarding) error {
	if doc.OrgID != req.OrgID {
		return errorOrgIDMismatch
	}

	return nil
}

func (s *GcpStandaloneOnboardingService) ParseContractRequest(ctx *gin.Context) (*flexsavestandalone.StandaloneContractRequest, *GCPStandaloneResponse) {
	step := pkg.OnboardingStepContract

	req, err := flexsavestandalone.ParseContractRequest(ctx)
	ctxWithValue := context.WithValue(ctx, flexsavestandalone.CustomerIDKey, req.CustomerID)

	if err != nil {
		return nil, s.updateError(ctxWithValue, step, err)
	}

	if err := flexsavestandalone.ValidateContractRequest(req); err != nil {
		return nil, s.updateError(ctxWithValue, step, err)
	}

	return req, Success
}

// updateError prints & updates error on standalone fs document, and returns 200 with { success: false } payload
func (s *GcpStandaloneOnboardingService) updateError(ctx context.Context, step pkg.StandaloneOnboardingStep, originalError error) *GCPStandaloneResponse {
	logger := s.getLogger(ctx)

	customerID, ok := ctx.Value(flexsavestandalone.CustomerIDKey).(string)
	if !ok {
		customerID = ""
	}

	standaloneID := s.composeStandaloneID(customerID, "")

	flexsavestandalone.UpdateError(ctx, logger, pkg.GCP, standaloneID, step, originalError, s.flexsaveStandaloneDAL.UpdateStandaloneOnboardingError)

	return Failure
}

// getEstimations invokes gcpService several times if savings returned empty
func (s *GcpStandaloneOnboardingService) getEstimations(ctx context.Context, serviceAccountEmail, billingAccountID string) (float64, float64, error) {
	var monthlySavings, annualSavings float64

	err := retry.LinearDelay(
		func() error {
			var err error

			monthlySavings, annualSavings, err = s.gcpService.GetEstimatedSavings(ctx, serviceAccountEmail, []string{billingAccountID})
			if err != nil {
				return err
			}

			if monthlySavings == 0 && annualSavings == 0 {
				return flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypeSavings, errorEmptySavings)
			}

			return nil
		},
		6,
		time.Second*10,
	)

	return monthlySavings, annualSavings, err
}

// testIamPermissions wraps gcpService to test each permission individually
func (s *GcpStandaloneOnboardingService) testIamPermissions(ctx context.Context, orgID, serviceAccountEmail, permission string, dryRun bool) error {
	if err := s.gcpService.TestIamPermissions(ctx, orgID, serviceAccountEmail, []string{permission}, dryRun); err != nil {
		if strings.Contains(err.Error(), "IAM permissions") {
			errWithPermissionName := fmt.Errorf("[permission '%s']: %w", permission, err)
			return flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypePermissions, errWithPermissionName)
		}

		return err
	}

	return nil
}

func (s *GcpStandaloneOnboardingService) setServiceAccountConfig(ctx context.Context, serviceAccountEmail, billingAccountID string, req *GCPStandaloneRequest) error {
	saConfig := &pkg.ServiceAccount{
		BillingAccountID: billingAccountID,
		Billing: &pkg.BillingTableInfo{
			ProjectID: req.ProjectID,
			DatasetID: req.DatasetID,
			TableID:   req.TableID,
		},
		ServiceAccountEmail: serviceAccountEmail,
		Customer:            s.customersDAL.GetRef(ctx, req.CustomerID),
	}

	err := s.flexsaveStandaloneDAL.SetGCPServiceAccountConfig(ctx, saConfig)

	return err
}

func (s *GcpStandaloneOnboardingService) runAllTests(ctx context.Context, req *GCPStandaloneRequest, doc *pkg.GCPStandaloneOnboarding) error {
	if err := s.verifyParams(req, doc); err != nil {
		return err
	}

	permissionsGrp, _ := errgroup.WithContext(ctx)

	for _, permission := range permissionsBilling {
		permissionsGrp.Go(func() error {
			return s.testIamPermissions(ctx, req.OrgID, doc.ServiceAccountEmail, permission, req.DryRun)
		})
	}

	if err := permissionsGrp.Wait(); err != nil {
		return err
	}

	testingGrp, _ := errgroup.WithContext(ctx)

	testingGrp.Go(func() error {
		return s.testBillingConnection.TestBillingConnection(ctx, doc.BillingAccountID, doc.ServiceAccountEmail, &pkg.BillingTableInfo{
			TableID:   req.TableID,
			DatasetID: req.DatasetID,
			ProjectID: req.ProjectID,
		})
	})

	// -------  detailed billing pipeline call -------
	// if err := s.billingPipelineService.TestConnection(ctx, doc.BillingAccountID, doc.ServiceAccountEmail, &pkg.BillingTablesLocation{
	// 	DetailedProjectID: req.ProjectID,
	// 	DetailedDatasetID: req.DatasetID,
	// }); err != nil {
	// 	return err
	// }

	testingGrp.Go(func() error {
		return s.gcpService.TestAllocation(ctx, doc.ServiceAccountEmail, doc.BillingAccountID)
	})

	if err := testingGrp.Wait(); err != nil {
		return err
	}

	return nil
}

func (s *GcpStandaloneOnboardingService) runEnablement(ctx context.Context, req *GCPStandaloneRequest, doc *pkg.GCPStandaloneOnboarding) error {
	if err := s.serviceAccounts.MarkServiceAccountOnBoardSuccessful(ctx, doc.ServiceAccountEmail, doc.Customer.ID, doc.BillingAccountID); err != nil {
		return err
	}

	if err := s.flexsaveStandaloneDAL.CompleteStandaloneOnboarding(ctx, s.composeStandaloneID(req.CustomerID, "")); err != nil {
		return err
	}

	if err := s.enableCustomer(ctx, req.CustomerID); err != nil {
		return err
	}

	return nil
}

func (s *GcpStandaloneOnboardingService) runBillingImport(ctx context.Context, req *GCPStandaloneRequest, doc *pkg.GCPStandaloneOnboarding) error {
	if err := s.billingImportStatus.SetStatusPending(ctx, req.CustomerID, doc.BillingAccountID); err != nil {
		return err
	}

	if err := s.billingImportStatus.UpdateMaxTimesThresholds(ctx, req.CustomerID, doc.BillingAccountID); err != nil {
		return err
	}

	if err := s.billing.Onboard(ctx, &billingDataStructures.OnboardingRequestBody{
		CustomerID:          req.CustomerID,
		ServiceAccountEmail: doc.ServiceAccountEmail,
		BillingAccountID:    doc.BillingAccountID,
		ProjectID:           req.ProjectID,
		Dataset:             req.DatasetID,
		TableID:             req.TableID,
	}); err != nil {
		return err
	}

	// -------  detailed billing pipeline call -------
	// if err := s.billingPipelineService.Onboard(ctx, req.CustomerID, doc.BillingAccountID, doc.ServiceAccountEmail,
	// 	&pkg.BillingTablesLocation{
	// 		DetailedProjectID: req.ProjectID,
	// 		DetailedDatasetID: req.DatasetID,
	// 	}); err != nil {
	// 	return err
	// }

	return nil
}

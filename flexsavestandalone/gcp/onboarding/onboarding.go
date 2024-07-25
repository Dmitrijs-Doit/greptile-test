package onboarding

import (
	"context"
	"strings"

	"github.com/doitintl/firestore/pkg"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/savingsreportfile"
)

/**
Standalone onboarding documents are being store in a "StandaloneID" convention
`google-cloud-CUSTOMER_ID.accounts.ORG_ID`
which construct with:
* firestore document ID: `google-cloud-CUSTOMER_ID`
* accounts field: `accounts`
* gcp org ID: `ORG_ID


** In order to generate "Standalone ID" use server/shared/firestore/pkg.ComposeStandaloneID(). Call it easily by s.composeStandaloneID(customerID, "")
** In order to take crucial parts out of standaloneID use server/shared/firestore/pkg.ExtractStandaloneID()

**** note: GCP is yet to support multiple standalone accounts thus still using regular documentID convention (but works with the same compose & extract functions)
*/

// InitOnboarding attaches service account email with a given customer
func (s *GcpStandaloneOnboardingService) InitOnboarding(ctx context.Context, customerID string) *GCPStandaloneResponse {
	step := pkg.OnboardingStepInit
	ctx = context.WithValue(ctx, flexsavestandalone.CustomerIDKey, customerID)
	logger := s.getLogger(ctx)

	documentID := s.getDocumentID(customerID)
	standaloneID := s.composeStandaloneID(customerID, "")

	err := flexsavestandalone.InitOnboarding(ctx, customerID, documentID, s.customersDAL, s.flexsaveStandaloneDAL)
	if err == pkg.ErrorAlreadyExist {
		return Success
	}

	if err != nil {
		return s.updateError(ctx, step, err)
	}

	serviceAccountEmail, err := s.serviceAccounts.GetNextFreeServiceAccount(ctx, customerID)
	if err != nil {
		return s.updateError(ctx, step, err)
	}

	if serviceAccountEmail == "" {
		return s.updateError(ctx, step, errorServiceAccount)
	}

	document, err := s.flexsaveStandaloneDAL.GetGCPStandaloneOnboarding(ctx, standaloneID)
	if err != nil {
		return s.updateError(ctx, step, err)
	}

	document.ServiceAccountEmail = serviceAccountEmail

	if err := s.flexsaveStandaloneDAL.UpdateStandaloneOnboarding(ctx, standaloneID, document); err != nil {
		return s.updateError(ctx, step, err)
	}

	logger.Info(flexsavestandalone.StepMessageInitOnboardingCompleted)

	return Success
}

// TestEstimationsConnection tests if SA has the relevant permissions in order to get estimations & then gets estimations
func (s *GcpStandaloneOnboardingService) TestEstimationsConnection(ctx context.Context, req *GCPStandaloneRequest) *GCPStandaloneResponse {
	step := pkg.OnboardingStepSavings
	ctx = context.WithValue(ctx, flexsavestandalone.CustomerIDKey, req.CustomerID)
	logger := s.getLogger(ctx)
	logger.Info(stepMessageTestEstimationsStarted)

	standaloneID := s.composeStandaloneID(req.CustomerID, "")

	document, err := s.flexsaveStandaloneDAL.GetGCPStandaloneOnboarding(ctx, standaloneID)
	if err != nil {
		return s.updateError(ctx, step, err)
	}

	document.OrgID = req.OrgID
	document.BillingAccountID = req.BillingAccountID

	if err := s.flexsaveStandaloneDAL.UpdateStandaloneOnboarding(ctx, standaloneID, document); err != nil {
		return s.updateError(ctx, step, err)
	}

	if err := s.gcpService.TestIamPermissions(ctx, req.OrgID, document.ServiceAccountEmail, permissionsEstimations, req.DryRun); err != nil {
		permissionValues := []string{"IAM permissions", "IAM_PERMISSION_DENIED", "Error 403: Permission"}

		for _, permissionValue := range permissionValues {
			if strings.Contains(err.Error(), permissionValue) {
				return s.updateError(ctx, step, flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypePermissions, err))
			}
		}

		return s.updateError(ctx, step, err)
	}

	if err := s.RefreshEstimations(ctx, req.CustomerID); err != nil {
		return s.updateError(ctx, step, err)
	}

	logger.Info(stepMessageTestEstimationsCompleted)

	return Success
}

// RefreshEstimationsWrapper wraps and invokes RefreshEstimations when being triggered directly from client, only print warnings
func (s *GcpStandaloneOnboardingService) RefreshEstimationsWrapper(ctx context.Context, customerID string) *GCPStandaloneResponse {
	ctx = context.WithValue(ctx, flexsavestandalone.CustomerIDKey, customerID)
	logger := s.getLogger(ctx)
	logger.Info(flexsavestandalone.StepMessageRefreshEstimationsStarted)

	if err := s.RefreshEstimations(ctx, customerID); err != nil {
		logger.Warningf(flexsavestandalone.ErrorLogSyntax, pkg.OnboardingStepSavings, err)
		return Success
	}

	logger.Info(flexsavestandalone.StepMessageRefreshEstimationsCompleted)

	return Success
}

// RefreshEstimations gets & update savings estimations for a given customer's billing account
func (s *GcpStandaloneOnboardingService) RefreshEstimations(ctx context.Context, customerID string) error {
	if customerID == "" {
		return flexsavestandalone.ErrorCustomerID
	}

	standaloneID := s.composeStandaloneID(customerID, "")

	doc, err := s.flexsaveStandaloneDAL.GetGCPStandaloneOnboarding(ctx, standaloneID)
	if err != nil {
		return err
	}

	if doc.BillingAccountID == "" {
		return errorBillingAccount
	}

	monthlySavings, annualSavings, err := s.getEstimations(ctx, doc.ServiceAccountEmail, doc.BillingAccountID)
	if err != nil {
		return err
	}

	savings := map[string]float64{
		string(pkg.AnnualSavings):  annualSavings,
		string(pkg.MonthlySavings): monthlySavings,
	}

	if err := s.flexsaveStandaloneDAL.UpdateStandaloneOnboardingSavings(ctx, standaloneID, savings); err != nil {
		return err
	}

	return nil
}

// AddContract creates contract for a given customer
func (s *GcpStandaloneOnboardingService) AddContract(ctx context.Context, req *flexsavestandalone.StandaloneContractRequest) *GCPStandaloneResponse {
	ctx = context.WithValue(ctx, flexsavestandalone.CustomerIDKey, req.CustomerID)
	s.getLogger(ctx)

	if err := flexsavestandalone.AddContract(
		ctx,
		req,
		flexsavestandalone.StandaloneContractTypeGCP,
		0,
		s.contractsDAL.Add,
		s.entitiesDAL,
		s.customersDAL,
		s.accountManagersDAL,
		s.flexsaveStandaloneDAL,
	); err != nil {
		return s.updateError(ctx, pkg.OnboardingStepContract, err)
	}

	return Success
}

// Activate final stage of onboarding
func (s *GcpStandaloneOnboardingService) Activate(ctx context.Context, req *GCPStandaloneRequest) *GCPStandaloneResponse {
	step := pkg.OnboardingStepActivation
	ctx = context.WithValue(ctx, flexsavestandalone.CustomerIDKey, req.CustomerID)
	logger := s.getLogger(ctx)
	logger.Info(stepMessageActivateStarted)

	standaloneID := s.composeStandaloneID(req.CustomerID, "")

	doc, err := s.flexsaveStandaloneDAL.GetGCPStandaloneOnboarding(ctx, standaloneID)
	if err != nil {
		return s.updateError(ctx, step, err)
	}

	if err := s.runAllTests(ctx, req, doc); err != nil {
		return s.updateError(ctx, step, err)
	}

	if err := s.CreateAsset(ctx, req.CustomerID, doc.ServiceAccountEmail, doc.BillingAccountID); err != nil {
		return s.updateError(ctx, step, err)
	}

	if err := s.cloudConnectDAL.CreateGCPCloudConnect(ctx, doc.BillingAccountID, doc.ServiceAccountEmail, s.customersDAL.GetRef(ctx, req.CustomerID)); err != nil {
		return s.updateError(ctx, step, err)
	}

	if err := s.runEnablement(ctx, req, doc); err != nil {
		return s.updateError(ctx, step, err)
	}

	// skip the import for the automation test billing account
	if doc.BillingAccountID == "0138FC-C9BDCB-34BD58" {
		return Success
	}

	if err := s.runBillingImport(ctx, req, doc); err != nil {
		return s.updateError(ctx, step, err)
	}

	if err := s.setServiceAccountConfig(ctx, doc.ServiceAccountEmail, doc.BillingAccountID, req); err != nil {
		return s.updateError(ctx, step, err)
	}

	logger.Info(stepMessageActivateCompleted)

	return Success
}

// CreateAsset creates GCP Asset on fs given GCP billing account properties
func (s *GcpStandaloneOnboardingService) CreateAsset(ctx context.Context, customerID, serviceAccountEmail, billingAccountID string) error {
	assetRef := s.assetsDAL.GetRef(ctx, s.getAssetID(billingAccountID))
	customerRef := s.customersDAL.GetRef(ctx, customerID)

	contractRef, err := s.contractsDAL.GetCustomerContractRef(ctx, customerRef, string(flexsavestandalone.StandaloneContractTypeGCP))
	if err != nil {
		return err
	}

	displayName, err := s.gcpService.GetBillingAccountDisplayName(ctx, serviceAccountEmail, billingAccountID)
	if err != nil {
		return err
	}

	if displayName == "" {
		displayName = billingAccountID
	}

	asset := assets.GCPAsset{
		BaseAsset: assets.BaseAsset{
			AssetType: common.Assets.GoogleCloudStandalone,
			Bucket:    nil,
			Contract:  contractRef,
			Entity:    nil,
			Customer:  customerRef,
		},
		Properties: &assets.GCPProperties{
			NumProjects:      1,
			BillingAccountID: billingAccountID,
			DisplayName:      displayName,
		},
	}

	if _, err := assetRef.Set(ctx, asset); err != nil {
		return err
	}

	if err := flexsavestandalone.UpdateContractAssets(ctx, s.contractsDAL, contractRef, assetRef); err != nil {
		return err
	}

	return nil
}

func (s *GcpStandaloneOnboardingService) CreateServiceAccounts(ctx context.Context) error {
	return s.serviceAccounts.CreateServiceAccounts(ctx)
}

func (s *GcpStandaloneOnboardingService) InitEnvironment(ctx context.Context) error {
	return s.serviceAccounts.InitEnvironment(ctx)
}

func (s *GcpStandaloneOnboardingService) GetSavingsFileReport(ctx context.Context, customerID string, savingsReport savingsreportfile.StandaloneSavingsReport) ([]byte, error) {
	return s.savingsReportService.CreateSavingsReportFile(ctx, customerID, savingsReport)
}

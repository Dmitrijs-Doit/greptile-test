package onboarding

import (
	"context"

	"github.com/doitintl/firestore/pkg"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
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
func (s *GCPSaaSConsoleOnboardService) InitOnboarding(ctx context.Context, customerID string) *saasconsole.OnboardingResponse {
	step := pkg.OnboardingStepInit
	ctx = context.WithValue(ctx, saasconsole.CustomerIDKey, customerID)
	logger := s.getLogger(ctx)

	documentID := s.getDocumentID(customerID)

	err := saasconsole.InitOnboarding(ctx, customerID, documentID, s.customersDAL, s.saasConsoleDAL)
	if err == pkg.ErrorAlreadyExist {
		return saasconsole.Success
	}

	if err != nil {
		return s.updateAndSlackError(ctx, step, "", err)
	}

	serviceAccountEmail, err := s.serviceAccounts.GetNextFreeServiceAccount(ctx, customerID)
	if err != nil {
		return s.updateAndSlackError(ctx, step, "", err)
	}

	if serviceAccountEmail == "" {
		return s.updateAndSlackError(ctx, step, "", errorServiceAccount)
	}

	document, err := s.saasConsoleDAL.GetGCPAccounts(ctx, documentID)
	if err != nil {
		return s.updateAndSlackError(ctx, step, "", err)
	}

	document.ServiceAccountEmail = serviceAccountEmail

	if err := s.saasConsoleDAL.UpdateOnboarding(ctx, documentID, document); err != nil {
		return s.updateAndSlackError(ctx, step, "", err)
	}

	logger.Info(saasconsole.StepMessageInitOnboardingCompleted)

	return saasconsole.Success
}

// AddContract creates contract for a given customer
func (s *GCPSaaSConsoleOnboardService) AddContract(ctx context.Context, req *saasconsole.StandaloneContractRequest) *saasconsole.OnboardingResponse {
	ctx = context.WithValue(ctx, saasconsole.CustomerIDKey, req.CustomerID)
	s.getLogger(ctx)

	if err := saasconsole.AddContract(
		ctx,
		req,
		saasconsole.StandaloneContractTypeGCP,
		0,
		s.contractsDAL.Add,
		s.entitiesDAL,
		s.customersDAL,
		s.accountManagersDAL,
		s.saasConsoleDAL,
	); err != nil {
		return s.updateAndSlackError(ctx, pkg.OnboardingStepContract, req.AccountID, err)
	}

	return saasconsole.Success
}

// Activate final stage of onboarding
func (s *GCPSaaSConsoleOnboardService) Activate(ctx context.Context, req *GCPStandaloneRequest) *saasconsole.OnboardingResponse {
	step := pkg.OnboardingStepActivation
	ctx = context.WithValue(ctx, saasconsole.CustomerIDKey, req.CustomerID)
	logger := s.getLogger(ctx)
	logger.Info(stepMessageActivateStarted)

	documentID := s.getDocumentID(req.CustomerID)

	doc, err := s.saasConsoleDAL.GetGCPAccounts(ctx, documentID)
	if err != nil {
		return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
	}

	standaloneID := pkg.ComposeStandaloneID(req.CustomerID, req.BillingAccountID, pkg.GCP)
	if err := s.saasConsoleDAL.UpdateOnboarding(ctx, standaloneID, &pkg.GCPSaaSConsoleOnboarding{
		BillingAccountID: req.BillingAccountID,
		ProjectID:        req.ProjectID,
		DatasetID:        req.DatasetID,
	}); err != nil {
		return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
	}

	if err := s.runAllTests(ctx, req, doc); err != nil {
		return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
	}

	if !req.DryRun {
		if err := s.runBillingImport(ctx, req, doc); err != nil {
			return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
		}
	}

	if err := s.CreateAsset(ctx, req.CustomerID, doc.ServiceAccountEmail, req.BillingAccountID); err != nil {
		return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
	}

	if err := s.cloudConnectDAL.CreateGCPCloudConnect(ctx, req.BillingAccountID, doc.ServiceAccountEmail, s.customersDAL.GetRef(ctx, req.CustomerID)); err != nil {
		return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
	}

	if err := s.runEnablement(ctx, req, doc); err != nil {
		return s.updateAndSlackError(ctx, step, req.BillingAccountID, err)
	}

	logger.Info(stepMessageActivateCompleted)

	_ = saasconsole.PublishOnboardSuccessSlackNotification(ctx, pkg.GCP, s.customersDAL, req.CustomerID, req.BillingAccountID)

	return saasconsole.Success
}

// CreateAsset creates GCP Asset on fs given GCP billing account properties
func (s *GCPSaaSConsoleOnboardService) CreateAsset(ctx context.Context, customerID, serviceAccountEmail, billingAccountID string) error {
	assetRef := s.assetsDAL.GetRef(ctx, s.getAssetID(billingAccountID))
	customerRef := s.customersDAL.GetRef(ctx, customerID)

	// contractRef, err := s.contractsDAL.GetCustomerContractRef(ctx, customerRef, string(saasconsole.StandaloneContractTypeGCP))
	// if err != nil {
	// 	return err
	// }

	displayName, err := s.getBillingAccountDisplayName(ctx, serviceAccountEmail, billingAccountID)
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
			Contract:  nil, // contract should be added after trial ends
			Entity:    nil,
			Customer:  customerRef,
		},
		Properties: &assets.GCPProperties{
			NumProjects:      1,
			BillingAccountID: billingAccountID,
			DisplayName:      displayName,
		},
		StandaloneProperties: &assets.GCPStandaloneProperties{
			BillingReady: false,
		},
	}

	if _, err := assetRef.Set(ctx, asset); err != nil {
		return err
	}

	// if err := saasconsole.UpdateContractAssets(ctx, s.contractsDAL, contractRef, assetRef); err != nil {
	// 	return err
	// }

	return nil
}

func (s *GCPSaaSConsoleOnboardService) CreateServiceAccounts(ctx context.Context) error {
	return s.serviceAccounts.CreateServiceAccounts(ctx)
}

func (s *GCPSaaSConsoleOnboardService) InitEnvironment(ctx context.Context) error {
	return s.serviceAccounts.InitEnvironment(ctx)
}

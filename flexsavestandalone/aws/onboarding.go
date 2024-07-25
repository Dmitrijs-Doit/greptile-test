package aws

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws/savingsreportfile"

	"github.com/doitintl/firestore/pkg"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	"github.com/doitintl/retry"
)

/**
Standalone onboarding documents are being store in a "StandaloneID" convention
`amazon-web-services-CUSTOMER_ID.accounts.ACCOUNT_ID`
which construct with:
* firestore document ID: `amazon-web-services-CUSTOMER_ID`
* accounts field: `accounts`
* aws account ID: `ACCOUNT_ID

Single document on firestore (/integrations/flexsave-standalone/fs-onboarding)
may include single or many aws accounts

** In order to generate "Standalone ID" use server/shared/firestore/pkg.ComposeStandaloneID(). Call it easily by s.composeStandaloneID(customerID, accountID)
** In order to take crucial parts out of standaloneID use server/shared/firestore/pkg.ExtractStandaloneID()
*/

// InitOnboarding creates onboarding document for the 1st time
func (s *AwsStandaloneService) InitOnboarding(ctx context.Context, customerID string) {
	logger := s.getLogger(ctx, customerID)
	step := pkg.OnboardingStepInit

	documentID := s.getDocumentID(customerID)

	err := flexsavestandalone.InitOnboarding(ctx, customerID, documentID, s.customersDAL, s.flexsaveStandaloneDAL)
	if err == pkg.ErrorAlreadyExist {
		return
	}

	if err != nil {
		s.updateError(ctx, customerID, "", step, err)
		return
	}

	logger.Info(flexsavestandalone.StepMessageInitOnboardingCompleted)
}

// UpdateRecommendationsWrapper wraps and invokes UpdateRecommendations with timeout, after the server safely returns 200
func (s *AwsStandaloneService) UpdateRecommendationsWrapper(ctx context.Context, req *AWSStandaloneRequest) {
	logger := s.getLogger(ctx, req.CustomerID)
	step := pkg.OnboardingStepSavings

	logger.Info(stepMessageUpdateRecommendationsStarted)

	if err := s.validateFields(req, false); err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
		return
	}

	err := retry.BackOffDelay(
		func() error {
			return s.UpdateRecommendations(ctx, req)
		},
		5,
		time.Second*3,
	)
	if err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
		return
	}

	logger.Info(stepMessageUpdateRecommendationsCompleted)
}

// UpdateRecommendations initiate customer onboarding state on fs & update its savings estimations
func (s *AwsStandaloneService) UpdateRecommendations(ctx context.Context, req *AWSStandaloneRequest) error {
	standaloneID := s.composeStandaloneID(req.CustomerID, req.AccountID)

	document := pkg.AWSStandaloneOnboarding{
		BaseStandaloneOnboarding: pkg.BaseStandaloneOnboarding{
			Customer: s.customersDAL.GetRef(ctx, req.CustomerID), // todo fssa avoid customer & type in nested struct?
			Type:     pkg.AWS,
		},
		AccountID:        req.AccountID,
		IsMissingAWSRole: false,
	}

	if err := s.flexsaveStandaloneDAL.UpdateStandaloneOnboarding(ctx, standaloneID, &document); err != nil {
		return err
	}

	if err := s.RefreshEstimationsForAccount(ctx, &document); err != nil {
		return err
	}

	return nil
}

// RefreshEstimations re-calculates and updates savings for a given customer, only print warnings
func (s *AwsStandaloneService) RefreshEstimations(ctx context.Context, customerID string) {
	logger := s.getLogger(ctx, customerID)
	logger.Info(flexsavestandalone.StepMessageRefreshEstimationsStarted)

	onboardingDocuments, err := s.flexsaveStandaloneDAL.GetAWSStandaloneAccounts(ctx, s.getDocumentID(customerID))
	if err != nil {
		logger.Warningf(flexsavestandalone.ErrorLogSyntax, pkg.OnboardingStepSavings, err)
		return
	}

	for _, standaloneDocument := range onboardingDocuments.Accounts {
		if standaloneDocument.IsMissingAWSRole {
			continue
		}

		if err := s.RefreshEstimationsForAccount(ctx, standaloneDocument); err != nil {
			logger.Warningf(flexsavestandalone.ErrorLogSyntax, pkg.OnboardingStepSavings, err)
		}
	}

	logger.Info(flexsavestandalone.StepMessageRefreshEstimationsCompleted)
}

// RefreshEstimationsForAccount re-calculates and updates savings for a given aws account
func (s *AwsStandaloneService) RefreshEstimationsForAccount(ctx context.Context, standaloneDocument *pkg.AWSStandaloneOnboarding) error {
	customerID := standaloneDocument.Customer.ID
	if customerID == "" {
		return flexsavestandalone.ErrorCustomerID
	}

	accountID := standaloneDocument.AccountID
	if accountID == "" {
		return errorAccountID
	}

	savings, oneYearRecommendations, err := s.getSavings(ctx, accountID, true)
	if err != nil {
		return err
	}

	_, threeYearsRecommendations, err := s.getSavings(ctx, accountID, false)
	if err != nil {
		return err
	}

	recommendations := map[string]*pkg.AWSSavingsPlansRecommendation{
		strings.ToLower(oneYear):    oneYearRecommendations,
		strings.ToLower(threeYears): threeYearsRecommendations,
	}

	standaloneID := s.composeStandaloneID(customerID, accountID)
	if err := s.flexsaveStandaloneDAL.UpdateStandaloneOnboardingSavingsAndRecommendation(ctx, standaloneID, savings, recommendations); err != nil {
		return err
	}

	return nil
}

// AddContract creates contract for a given customer
func (s *AwsStandaloneService) AddContract(ctx context.Context, req *flexsavestandalone.StandaloneContractRequest) {
	step := pkg.OnboardingStepContract
	standaloneID := s.composeStandaloneID(req.CustomerID, req.AccountID)

	estimated, err := s.calculateEstimated(ctx, standaloneID)
	if err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
		return
	}

	if err := flexsavestandalone.AddContract(
		ctx,
		req,
		flexsavestandalone.StandaloneContractTypeAWS,
		estimated,
		s.contractsDAL.Add,
		s.entitiesDAL,
		s.customersDAL,
		s.accountManagersDAL,
		s.flexsaveStandaloneDAL,
	); err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
	}
}

// UpdateBillingWrapper wraps and invokes UpdateBilling with timeout, after the server safely returns 200
func (s *AwsStandaloneService) UpdateBillingWrapper(ctx context.Context, req *AWSStandaloneRequest) {
	step := pkg.OnboardingStepActivation
	logger := s.getLogger(ctx, req.CustomerID)

	logger.Info(stepMessageUpdateBillingStarted)

	if err := s.validateFields(req, true); err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
		return
	}

	err := retry.BackOffDelay(
		func() error {
			return s.UpdateBilling(ctx, req)
		},
		5,
		time.Second*3,
	)
	if err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
		return
	}

	logger.Info(stepMessageUpdateBillingCompleted)
}

// UpdateBilling updates billing for the customer, final onboarding step
func (s *AwsStandaloneService) UpdateBilling(ctx context.Context, req *AWSStandaloneRequest) error {
	if err := s.validateCustomerAccountID(ctx, req.CustomerID, req.AccountID); err != nil {
		return err
	}

	if err := s.validateCUR(ctx, req.CustomerID, req.AccountID, req.S3Bucket, req.CURPath); err != nil {
		return err
	}

	if err := s.CreateAsset(ctx, req.CustomerID, req.AccountID); err != nil {
		return err
	}

	if err := s.CreateCloudConnect(ctx, req.CustomerID, req.AccountID, req.Arn, req.S3Bucket, req.CURPath); err != nil {
		return err
	}

	customer, err := s.customersDAL.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		return err
	}

	err = s.payers.CreatePayerConfigForCustomer(ctx, types.PayerConfigCreatePayload{PayerConfigs: []types.PayerConfig{
		{
			CustomerID:      req.CustomerID,
			AccountID:       req.AccountID,
			PrimaryDomain:   customer.PrimaryDomain,
			FriendlyName:    "",
			Name:            customer.Name,
			Status:          string(pkg.StandalonePayerConfigStatusPending),
			Type:            string(pkg.StandalonePayerConfigTypeAWS),
			SageMakerStatus: string(pkg.StandalonePayerConfigStatusPending),
			RDSStatus:       string(pkg.StandalonePayerConfigStatusPending),
		},
	}})
	if err != nil {
		return err
	}

	if err := s.enableCustomer(ctx, req.CustomerID); err != nil {
		return err
	}

	if err := s.flexsaveStandaloneDAL.CompleteStandaloneOnboarding(ctx, s.composeStandaloneID(req.CustomerID, req.AccountID)); err != nil {
		return err
	}

	if err := s.UpdateStandaloneCustomerSpendSummary(ctx, req.CustomerID, 2); err != nil {
		s.getLogger(ctx, req.CustomerID).Error(err)
	}

	return nil
}

func (s *AwsStandaloneService) DeleteAWSEstimation(ctx context.Context, customerID string, accountID string) error {
	account, err := s.flexsaveStandaloneDAL.GetAWSAccount(ctx, customerID, accountID)
	if err != nil {
		return err
	}

	if account == nil {
		return errors.New("account not found")
	}

	if account.Completed {
		return errors.New("account has already been activated")
	}

	return s.flexsaveStandaloneDAL.DeleteAWSAccount(ctx, s.getDocumentID(customerID), accountID)
}

// CreateAsset creates AWS Asset on fs given AWS account properties
func (s *AwsStandaloneService) CreateAsset(ctx context.Context, customerID, accountID string) error {
	assetRef := s.assetsDAL.GetRef(ctx, s.getAssetID(accountID))
	customerRef := s.customersDAL.GetRef(ctx, customerID)
	standaloneID := s.composeStandaloneID(customerID, accountID)

	entityRef, err := s.getEntity(ctx, customerID, standaloneID)
	if err != nil {
		return err
	}

	contractRef, err := s.contractsDAL.GetCustomerContractRef(ctx, customerRef, string(flexsavestandalone.StandaloneContractTypeAWS))
	if err != nil {
		return err
	}

	properties, err := s.getAssetProperties(accountID)
	if err != nil {
		return err
	}

	asset := assets.AWSAsset{
		BaseAsset: assets.BaseAsset{
			AssetType: common.Assets.AmazonWebServicesStandalone,
			Bucket:    nil,
			Contract:  contractRef,
			Entity:    entityRef,
			Customer:  customerRef,
		},
		Properties: properties,
	}

	if _, err := assetRef.Set(ctx, asset); err != nil {
		return err
	}

	if err := flexsavestandalone.UpdateContractAssets(ctx, s.contractsDAL, contractRef, assetRef); err != nil {
		return err
	}

	return nil
}

// CreateCloudConnect creates AWS CloudConnect doc on fs if valid Cost and Usage Report exist on AWS account
func (s *AwsStandaloneService) CreateCloudConnect(ctx context.Context, customerID, accountID, role, s3Bucket, curPath string) error {
	billingEtl := &pkg.BillingEtl{
		Settings: &pkg.BillingEtlSettings{
			Active:      true,
			Bucket:      s3Bucket,
			CurBasePath: curPath,
			DoitArn:     pkg.StandaloneDoitArn,
		},
	}
	customerRef := s.customersDAL.GetRef(ctx, customerID)
	arn := getRoleArn(role, accountID)

	return s.cloudConnectDAL.CreateAWSCloudConnect(ctx, accountID, arn, customerRef, billingEtl)
}

func (s *AwsStandaloneService) GetSavingsFileReport(ctx context.Context, customerID string, savingsReport savingsreportfile.StandaloneSavingsReport) ([]byte, error) {
	return s.savingsReportService.CreateSavingsReportFile(ctx, customerID, savingsReport)
}

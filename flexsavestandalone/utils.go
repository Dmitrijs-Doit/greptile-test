package flexsavestandalone

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type StandaloneContractType string

const (
	// contract types
	StandaloneContractTypeAWS    StandaloneContractType = "amazon-web-services-standalone"
	StandaloneContractTypeGCP    StandaloneContractType = "google-cloud-standalone"
	StandaloneContractTypeSignUp StandaloneContractType = "flexsave-standalone-sign-up"

	// contract URLs
	StandaloneContractURLGCP     = "https://docs.google.com/document/d/1iar8Hrk3SssE0PNZiCBbGnbJx6fFuBYb3xoEIoYGg68"
	StandaloneContractURLAWS     = "https://docs.google.com/document/d/10R-44hceYCBXvr9EbPCMVjMSvp2_nNBvZqu3CLvIOKk"
	StandaloneContractStorageAWS = "flexsave-standalone/public/amazon-web-services-standalone-contract-v2.pdf"
	StandaloneContractStorageGCP = "flexsave-standalone/public/google-cloud-standalone-contract-v2.pdf"

	// steps logs
	StepMessageInitOnboardingCompleted     = "standalone document created successfully"
	StepMessageRefreshEstimationsStarted   = "refresh-estimations started"
	StepMessageRefreshEstimationsCompleted = "refresh-estimations completed successfully"

	CustomerIDKey = "customerId"

	missingSelectedPriorityMessage   = "no selectedPriorityID found for customer %s (proceeding with 1st active entity)"
	standaloneAccountManagersGroupID = "EwTS8g54q8TXi13eqkaC"
	errorMissing                     = "missing %s"
	ErrorLogSyntax                   = "%s: %s"
	IDSyntax                         = "%s-%s" //	todo remove

	customerFieldName = "enabledFlexsave"
)

var (
	// errors
	ErrorCustomerID      = errors.New("missing customer id")
	ErrorBillingProfile  = errors.New("no billing profile found")
	errorEmail           = errors.New("missing email")
	errorPlatform        = errors.New("unknown platform")
	errorContractVersion = errors.New("no contract version found")
)

type StandaloneContractRequest struct {
	Email           string  `json:"email"`
	CustomerID      string  `json:"customerId"`
	AccountID       string  `json:"accountId"`
	ContractVersion float64 `json:"contractVersion"`
}

func GetAssetID(platform pkg.StandalonePlatform, ID string) string { //	for asset only
	platformPrefix := common.Assets.GoogleCloudStandalone
	if platform == pkg.AWS {
		platformPrefix = common.Assets.AmazonWebServicesStandalone
	}

	return fmt.Sprintf(IDSyntax, platformPrefix, ID)
}

func GetMissingError(err string) error {
	return fmt.Errorf(errorMissing, err)
}

func ParseContractRequest(ctx *gin.Context) (*StandaloneContractRequest, error) {
	var req StandaloneContractRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return &req, err
	}

	return &req, nil
}

func ValidateContractRequest(req *StandaloneContractRequest) error {
	if len(req.Email) == 0 {
		return errorEmail
	}

	if len(req.CustomerID) == 0 {
		return ErrorCustomerID
	}

	if req.ContractVersion == 0 {
		return errorContractVersion
	}

	return nil
}

func BuildStandaloneError(errorType pkg.StandaloneOnboardingErrorType, err error) *pkg.StandaloneOnboardingError {
	return &pkg.StandaloneOnboardingError{
		Type:    errorType,
		Message: err.Error(),
	}
}

// HandleError prints & updates error state on standalone fs document
func UpdateError(
	ctx context.Context,
	logger logger.ILogger,
	platform pkg.StandalonePlatform,
	standaloneID string,
	step pkg.StandaloneOnboardingStep,
	err error,
	updateErrorFunc func(context.Context, string, *pkg.StandaloneOnboardingError, pkg.StandaloneOnboardingStep) error,
) {
	logger.Errorf(ErrorLogSyntax, step, err)

	if standaloneID != "" {
		var standaloneOnboardingError *pkg.StandaloneOnboardingError
		if !errors.As(err, &standaloneOnboardingError) { //	if possible, parse given error as type StandaloneOnboardingError
			standaloneOnboardingError = &pkg.StandaloneOnboardingError{ //	otherwise - generate new StandaloneOnboardingError with type general
				Message: err.Error(),
				Type:    pkg.OnboardingErrorTypeGeneral,
			}
		}

		if err := updateErrorFunc(ctx, standaloneID, standaloneOnboardingError, step); err != nil {
			logger.Error(err)
		}
	}
}

// EnrichLogger adds common labels to flexsave standalone logs
func EnrichLogger(l logger.ILogger, customerID string, platform pkg.StandalonePlatform) logger.ILogger {
	service := "flexsave-standalone-gcp"
	if platform == pkg.AWS {
		service = "flexsave-standalone-aws"
	}

	l.SetLabels(map[string]string{ // TODO FSSA support payloadJson
		logger.LabelCustomerID: customerID,
		"service":              service,
		"flow":                 "onboarding",
	})

	return l
}

// InitOnboarding creates onboarding document for the 1st time
func InitOnboarding(ctx context.Context, customerID, documentID string, customersDAL customerDal.Customers, flexsaveStandaloneDAL fsdal.FlexsaveStandalone) error {
	if customerID == "" {
		return ErrorCustomerID
	}

	customerRef := customersDAL.GetRef(ctx, customerID)

	err := flexsaveStandaloneDAL.InitStandaloneOnboarding(ctx, documentID, customerRef)

	return err
}

// EnableCustomer set enabledFlexsave on customer document
func EnableCustomer(ctx context.Context, customerID string, platform pkg.StandalonePlatform, customersDAL customerDal.Customers) error {
	customerRef := customersDAL.GetRef(ctx, customerID)

	customer, err := customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	enabledFlexsave := customer.EnabledFlexsave
	if enabledFlexsave == nil {
		enabledFlexsave = &common.CustomerEnabledFlexsave{}
	}

	switch pkg.StandalonePlatform(platform) {
	case pkg.AWS:
		enabledFlexsave.AWS = true
		break
	case pkg.GCP:
		enabledFlexsave.GCP = true
		break
	default:
		return errorPlatform
	}

	_, err = customerRef.Set(ctx, map[string]interface{}{
		"enabledFlexsave": enabledFlexsave,
	}, firestore.MergeAll)

	return nil
}

// DisableCustomer set enabledFlexsave on customer document
func DisableCustomer(ctx context.Context, customerID string, platform pkg.StandalonePlatform, customersDAL customerDal.Customers) error {
	customer, err := customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	enabledFlexsave := customer.EnabledFlexsave
	if enabledFlexsave == nil {
		enabledFlexsave = &common.CustomerEnabledFlexsave{}
	}

	switch pkg.StandalonePlatform(platform) {
	case pkg.AWS:
		enabledFlexsave.AWS = false
	case pkg.GCP:
		enabledFlexsave.GCP = false
	default:
		return errorPlatform
	}

	if err = customersDAL.UpdateCustomerFieldValue(ctx, customerID, customerFieldName, enabledFlexsave); err != nil {
		return err
	}

	return nil
}

// UpdateContractAssets adds asset ref to contract's assets (once)
func UpdateContractAssets(ctx context.Context, contractsDAL fsdal.Contracts, contractRef, assetRef *firestore.DocumentRef) error {
	ID := contractRef.ID

	contract, err := contractsDAL.Get(ctx, ID)
	if err != nil {
		return err
	}

	for _, asset := range contract.Assets {
		if asset.ID == assetRef.ID {
			return nil
		}
	}

	contract.Assets = append(contract.Assets, assetRef)

	return contractsDAL.Set(ctx, ID, contract)
}

// GetEntity returns selected billing profile OR the 1st active entity listed for a given customer
func GetEntity( // to be update with unique entity ID
	ctx context.Context,
	customerID,
	standaloneID string,
	entitiesDAL fsdal.Entities,
	getCustomerFunc func(context.Context, string) (*common.Customer, error),
	getPriorityFunc func(context.Context, string) (string, error),
) (*firestore.DocumentRef, error) {
	l := logger.FromContext(ctx)
	noSelectedPriorityID := false

	priorityID, err := getPriorityFunc(ctx, standaloneID)
	if err != nil {
		l.Errorf("failed to get priorityID for customer %s and standaloneID %s with error %s", customerID, standaloneID, err)
	}

	if priorityID == "" {
		noSelectedPriorityID = true

		l.Infof(missingSelectedPriorityMessage, customerID)
	}

	customer, err := getCustomerFunc(ctx, customerID)
	if err != nil {
		return nil, err
	}

	for _, entityRef := range customer.Entities {
		entity, err := entitiesDAL.GetEntity(ctx, entityRef.ID)
		if err != nil {
			return nil, err
		}

		if entity.PriorityID == priorityID {
			return entityRef, nil
		}

		if noSelectedPriorityID && entity.Active {
			return entityRef, nil
		}
	}

	return nil, ErrorBillingProfile
}

// AddContract creates contract for a given customer
func AddContract(
	ctx context.Context,
	req *StandaloneContractRequest,
	contractType StandaloneContractType,
	estimatedValue float64,
	addContractFunc func(context.Context, *pkg.Contract) error,
	entitiesDAL fsdal.Entities,
	customersDAL customerDal.Customers,
	accountManagersDAL fsdal.AccountManagers,
	flexsaveStandaloneDAL fsdal.FlexsaveStandalone,
) error {
	if err := ValidateContractRequest(req); err != nil {
		return err
	}

	contractTypeStr := string(contractType)
	platform := pkg.GCP
	url := StandaloneContractURLGCP
	storage := StandaloneContractStorageGCP

	var entity *firestore.DocumentRef

	var err error

	standaloneID := pkg.ComposeStandaloneID(req.CustomerID, req.AccountID, platform)

	if contractType == StandaloneContractTypeAWS {
		platform = pkg.AWS
		url = StandaloneContractURLAWS
		storage = StandaloneContractStorageAWS

		entity, err = GetEntity(ctx, req.CustomerID, standaloneID, entitiesDAL, customersDAL.GetCustomer, flexsaveStandaloneDAL.GetStandalonePriorityID)
		if err != nil {
			return err
		}
	}

	customer := customersDAL.GetRef(ctx, req.CustomerID)
	accountManager := accountManagersDAL.GetRef(standaloneAccountManagersGroupID)

	properties := map[string]interface{}{}
	properties["userEmail"] = req.Email
	now := time.Now()

	contract := pkg.Contract{
		Type:           contractTypeStr,
		Customer:       customer,
		Entity:         entity,
		Assets:         nil,
		Active:         true,
		StartDate:      &now,
		Properties:     properties,
		AccountManager: accountManager,
		IsRenewal:      false,
		ContractFile: &pkg.ContractFile{
			ID:       contractTypeStr,
			Name:     contractTypeStr,
			ParentID: contractTypeStr,
			URL:      url,
			Storage:  storage,
		},
	}

	if estimatedValue != 0 {
		contract.EstimatedValue = estimatedValue
	}

	if err := addContractFunc(ctx, &contract); err != nil {
		return err
	}

	if err := flexsaveStandaloneDAL.AgreedStandaloneContract(ctx, standaloneID, req.ContractVersion); err != nil {
		return err
	}

	return nil
}

package saasconsole

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

type StandaloneContractType string

type OnboardingResponse struct {
	Success bool `json:"success"`
}

const (
	// contract types
	StandaloneContractTypeAWS StandaloneContractType = "saas-console-aws"
	StandaloneContractTypeGCP StandaloneContractType = "saas-console-gcp"

	StandaloneContractStorageGCPFormat string = "saas-console/public/DoiT International SaaS Agreement GCP - v%d.pdf"
	StandaloneContractStorageAWSFormat string = "saas-console/public/DoiT International SaaS Agreement AWS - v%d.pdf"

	// steps logs
	StepMessageInitOnboardingCompleted = "SaaS Console onboarding document created successfully"

	CustomerIDKey = "customerId"

	missingSelectedPriorityMessage   = "no selectedPriorityID found for customer %s (proceeding with 1st active entity)"
	standaloneAccountManagersGroupID = "bNfRTZiE2a4eFRiyWbQA"
	errorMissing                     = "missing %s"
	ErrorLogSyntax                   = "%s: %s"
	IDSyntax                         = "%s-%s" //	todo remove

	enabledSaaSFieldName = "enabledSaaSConsole"
)

var (
	Success = &OnboardingResponse{true}
	Failure = &OnboardingResponse{false}

	// errors
	ErrorCustomerID      = errors.New("missing customer id")
	errorEmail           = errors.New("missing email")
	errorPlatform        = errors.New("unknown platform")
	errorContractVersion = errors.New("no contract version found")

	//contract URLs
	ContractsUrls = map[StandaloneContractType][]string{
		StandaloneContractTypeAWS: {
			"https://docs.google.com/document/d/1i52RlYVcCGJUm18teLj9vXnwvhtjk-hdEDNhlVDgDuQ",
			"https://docs.google.com/document/d/1LZ4buMgTYdkaR05Z07RMV-YoytviwhbGzCmZcbRQGZ0",
		},
		StandaloneContractTypeGCP: {
			"https://docs.google.com/document/d/16wlUw1ojVq_JCJe9WYoPw-d_SIsIEnbqiJTE8zd7r8w",
			"https://docs.google.com/document/d/10X211bm8gBJ3Oyyyy2XlTuxrNM0zBpyK35D0ct4z-Ck",
		},
	}
)

type AWSCustomCurInfo struct {
	AccountID  string
	Bucket     string
	PathPrefix string
	ReportPath string
}

var AWSCustomCustomerCurMap = map[string]map[string]AWSCustomCurInfo{
	"m3nFiMlthpifIZ4fJkTt": {
		"311740991759": {
			Bucket:     "finops-doit.management.prd.zi",
			PathPrefix: "cur",
			ReportPath: "311740991759",
		},
		"153118461199": {
			Bucket:     "finops-doit.management.prd.clickagy",
			PathPrefix: "cur",
			ReportPath: "153118461199",
		},
		"219188493823": {
			Bucket:     "finops-doit.management.prd.engage",
			PathPrefix: "cur",
			ReportPath: "219188493823",
		},
		"072848182243": {
			Bucket:     "finops-doit.management.prd.legacy-zi",
			PathPrefix: "cur",
			ReportPath: "072848182243",
		},
	},
}

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

	if req.ContractVersion < 1 {
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

// EnrichLogger adds common labels to billing standalone logs
func EnrichLogger(l logger.ILogger, customerID string, platform pkg.StandalonePlatform) logger.ILogger {
	service := "billing-standalone-gcp"
	if platform == pkg.AWS {
		service = "billing-standalone-aws"
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		"service":              service,
		"flow":                 "onboarding",
	})

	return l
}

// InitOnboarding creates onboarding document for the 1st time
func InitOnboarding(ctx context.Context, customerID, documentID string, customersDAL customerDal.Customers, saasConsoleDAL fsdal.SaaSConsoleOnboard) error {
	if customerID == "" {
		return ErrorCustomerID
	}

	customerRef := customersDAL.GetRef(ctx, customerID)

	err := saasConsoleDAL.InitOnboarding(ctx, documentID, customerRef)

	return err
}

// EnableCustomer set enabledSaaSConsole on customer document
func EnableCustomer(ctx context.Context, customerID string, platform pkg.StandalonePlatform, customersDAL customerDal.Customers) error {
	return updateCustomerState(ctx, customerID, platform, customersDAL, true)
}

// DisableCustomer set enabledSaaSConsole on customer document
func DisableCustomer(ctx context.Context, customerID string, platform pkg.StandalonePlatform, customersDAL customerDal.Customers) error {
	return updateCustomerState(ctx, customerID, platform, customersDAL, false)
}

func updateCustomerState(ctx context.Context, customerID string, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, enabled bool) error {
	customer, err := customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	enabledSaaSConsole := customer.EnabledSaaSConsole
	if enabledSaaSConsole == nil {
		enabledSaaSConsole = &common.CustomerEnabledSaaSConsole{}
	}

	switch pkg.StandalonePlatform(platform) {
	case pkg.AWS:
		enabledSaaSConsole.AWS = enabled
	case pkg.GCP:
		enabledSaaSConsole.GCP = enabled
	default:
		return errorPlatform
	}

	if err = customersDAL.UpdateCustomerFieldValue(ctx, customerID, enabledSaaSFieldName, enabledSaaSConsole); err != nil {
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
	saasConsoleDAL fsdal.SaaSConsoleOnboard,
) error {
	if err := ValidateContractRequest(req); err != nil {
		return err
	}

	contractTypeStr := string(contractType)

	if int(req.ContractVersion) > len(ContractsUrls[contractType]) {
		return errorContractVersion
	}

	url := ContractsUrls[contractType][int(req.ContractVersion)-1]
	storage := fmt.Sprintf(StandaloneContractStorageGCPFormat, int(req.ContractVersion))

	customer := customersDAL.GetRef(ctx, req.CustomerID)
	accountManager := accountManagersDAL.GetRef(standaloneAccountManagersGroupID)

	properties := map[string]interface{}{}
	properties["userEmail"] = req.Email
	now := time.Now()

	contract := pkg.Contract{
		Type:           contractTypeStr,
		Customer:       customer,
		Assets:         nil,
		Active:         true,
		StartDate:      &now,
		TimeCreated:    now,
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

	var standaloneID string
	if contractType == StandaloneContractTypeAWS {
		standaloneID = pkg.ComposeStandaloneID(req.CustomerID, req.AccountID, pkg.AWS)
	} else if contractType == StandaloneContractTypeGCP {
		standaloneID = pkg.GetDocumentID(req.CustomerID, pkg.GCP)
	} else {
		return errorPlatform
	}

	if err := saasConsoleDAL.AgreedContract(ctx, standaloneID, req.ContractVersion); err != nil {
		return err
	}

	return nil
}

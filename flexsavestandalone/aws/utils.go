package aws

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AWSStandaloneRequest struct {
	CustomerID string `json:"external_id"`
	Arn        string `json:"management_arn"`
	AccountID  string `json:"account_id"`
	StackID    string `json:"stack_id"`
	StackName  string `json:"stack_name"`
	CURPath    string `json:"cur_path"`
	S3Bucket   string `json:"s3_bucket"`
}

type EstimationSummaryCSVRequest struct {
	OfferingID                        string `json:"offering_id" validate:"required"`
	HourlyCommitmentToPurchase        string `json:"hourly_commitment_to_purchase" validate:"required"`
	EstimatedSavingsPlansCost         string `json:"estimated_savings_plans_cost" validate:"required"`
	EstimatedOnDemandCost             string `json:"estimated_on_demand_cost" validate:"required"`
	CurrentAverageHourlyOnDemandSpend string `json:"current_average_hourly_on_demand_spend" validate:"required"`
	CurrentMinimumHourlyOnDemandSpend string `json:"current_minimum_hourly_on_demand_spend" validate:"required"`
	CurrentMaximumHourlyOnDemandSpend string `json:"current_maximum_hourly_on_demand_spend" validate:"required"`
	EstimatedAverageUtilization       string `json:"estimated_average_utilization" validate:"required"`
	EstimatedMonthlySavingsAmount     string `json:"estimated_monthly_savings_amount" validate:"required"`
	EstimatedSavingsPercentage        string `json:"estimated_savings_percentage" validate:"required"`
	EstimatedROI                      string `json:"estimated_roi" validate:"required"`
}

const (
	stackDeletion  = "AWS initiated Stack deletion"
	standaloneRole = "doitintl_cmp"

	devSlackChannel  = "#test-aws-fs-ops-notifications"
	prodSlackChannel = "#fsaws-ops"

	// aws functions
	functionGetSavingsPlansPurchaseRecommendation = "GetSavingsPlansPurchaseRecommendation"
	functionDescribeReportDefinitions             = "DescribeReportDefinitions"
	functionDescribeAccount                       = "DescribeAccount"

	// steps log messages
	stepMessageUpdateBillingStarted           = "update-billing started"
	stepMessageUpdateBillingCompleted         = "update-billing completed successfully"
	stepMessageUpdateRecommendationsStarted   = "update-recommendations started"
	stepMessageUpdateRecommendationsCompleted = "update-recommendations completed successfully"

	accountID = "account id"
	s3Bucket  = "s3 bucket"
	curPath   = "CUR path"
)

var (
	// aws constant
	payer       = costexplorer.AccountScopePayer
	computeSP   = costexplorer.SupportedSavingsPlansTypeComputeSp
	noUpfront   = costexplorer.PaymentOptionNoUpfront
	sevenDays   = costexplorer.LookbackPeriodInDaysSevenDays
	thirtyDays  = costexplorer.LookbackPeriodInDaysThirtyDays
	threeYears  = costexplorer.TermInYearsThreeYears
	oneYear     = costexplorer.TermInYearsOneYear
	hourly      = costandusagereportservice.TimeUnitHourly
	reportOrCsv = costandusagereportservice.ReportFormatTextOrcsv
	resources   = costandusagereportservice.SchemaElementResources

	// errors
	errorAccountID            = errors.New("missing account id")
	errorBadAccountID         = errors.New("customer's account id does not correlates the one supplied from AWS")
	errorEmptyAssumeRole      = errors.New("empty assume role response returned from AWS")
	errorEmptySavingsPlans    = errors.New("empty savings plans recommendations returned from AWS")
	errorEmptyDescribeAccount = errors.New("empty account response returned from AWS")
	errorCUR                  = errors.New("no valid Cost And Usage Report found for AWS account")
	errorInvalidCUR           = errors.New("invalid Cost And Usage Report path")

	emptySavingsMap = map[string]float64{
		string(pkg.LastMonthComputeSpend): 0,
		string(pkg.EstimatedSavings):      0,
		string(pkg.MonthlySavings):        0,
	}
)

func (s *AwsStandaloneService) getDocumentID(customerID string) string {
	return pkg.GetDocumentID(customerID, pkg.AWS)
}

func (s *AwsStandaloneService) composeStandaloneID(customerID, accountID string) string {
	return pkg.ComposeStandaloneID(customerID, accountID, pkg.AWS)
}

// amazon-web-services-standalone...
func (s *AwsStandaloneService) getAssetID(accountID string) string {
	return flexsavestandalone.GetAssetID(pkg.AWS, accountID)
}

func getRoleArn(role, accountID string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, role)
}

func (s *AwsStandaloneService) getPayerAccountName(ID string) string {
	return fmt.Sprintf("standalone-payer-%s", ID)
}

func (s *AwsStandaloneService) extractCUR(curPath string) string {
	pathParts := strings.Split(curPath, "/")
	len := len(pathParts)

	return pathParts[len-1]
}

func (s *AwsStandaloneService) getLogger(ctx context.Context, customerID string) logger.ILogger {
	logger := s.loggerProvider(ctx)
	flexsavestandalone.EnrichLogger(logger, customerID, pkg.AWS)

	return logger
}

// enableCustomer set enabledFlexsave.AWS on customer document
func (s *AwsStandaloneService) enableCustomer(ctx context.Context, customerID string) error {
	return flexsavestandalone.EnableCustomer(ctx, customerID, pkg.AWS, s.customersDAL)
}

func (s *AwsStandaloneService) ParseRequest(ctx *gin.Context, fullPayload bool) *AWSStandaloneRequest {
	step := pkg.OnboardingStepSavings
	if fullPayload {
		step = pkg.OnboardingStepActivation
	}

	var req AWSStandaloneRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		s.updateError(ctx, req.CustomerID, req.AccountID, step, err)
	}

	req.CURPath = common.RemoveLeadingAndTrailingSlashes(req.CURPath)

	return &req
}

func (s *AwsStandaloneService) ParseContractRequest(ctx *gin.Context) *flexsavestandalone.StandaloneContractRequest {
	req, err := flexsavestandalone.ParseContractRequest(ctx)
	if err != nil {
		s.updateError(ctx, req.CustomerID, req.AccountID, pkg.OnboardingStepContract, err)
	}

	return req
}

// validateFields validates mandatory fields has received from the request (fullPayload true for activation endpoint)
func (s *AwsStandaloneService) validateFields(req *AWSStandaloneRequest, fullPayload bool) error {
	if len(req.CustomerID) == 0 {
		return flexsavestandalone.ErrorCustomerID
	}

	if len(req.AccountID) == 0 {
		return flexsavestandalone.GetMissingError(accountID)
	}

	if fullPayload {
		if len(req.S3Bucket) == 0 {
			return flexsavestandalone.GetMissingError(s3Bucket)
		}

		if len(req.CURPath) == 0 {
			return flexsavestandalone.GetMissingError(curPath)
		}
	}

	return nil
}

// StackDeletion add log pointing for stack deletion initiated by AWS console
func (s *AwsStandaloneService) StackDeletion(ctx context.Context, customerID string) {
	s.getLogger(ctx, customerID).Infof("%s for customer %s", stackDeletion, customerID)
}

// handleError prints & updates error on standalone fs document
func (s *AwsStandaloneService) updateError(ctx context.Context, customerID, accountID string, step pkg.StandaloneOnboardingStep, err error) {
	logger := s.getLogger(ctx, customerID)

	standaloneID := ""
	if accountID != "" { //	only pass standaloneID when accountID is known
		standaloneID = s.composeStandaloneID(customerID, accountID)
	}

	flexsavestandalone.UpdateError(ctx, logger, pkg.AWS, standaloneID, step, err, s.flexsaveStandaloneDAL.UpdateStandaloneOnboardingError)
}

// calculateEstimated returns {3yr - 1yr} savings calculation to be saved on a contract
func (s *AwsStandaloneService) calculateEstimated(ctx context.Context, standaloneID string) (float64, error) {
	onboardingDoc, err := s.flexsaveStandaloneDAL.GetAWSStandaloneOnboarding(ctx, standaloneID)
	if err != nil {
		return 0, err
	}

	if onboardingDoc.IsMissingAWSRole {
		return onboardingDoc.Savings["estimatedSavings"], nil
	}

	oneYearSavings := onboardingDoc.Recommendations[strings.ToLower(oneYear)].SavingsPlansPurchaseRecommendationSummary.EstimatedMonthlySavingsAmount

	oneYearSavingsFloat, err := strconv.ParseFloat(oneYearSavings, 64)
	if err != nil {
		return 0, err
	}

	threeYearsSavings := onboardingDoc.Recommendations[strings.ToLower(threeYears)].SavingsPlansPurchaseRecommendationSummary.EstimatedMonthlySavingsAmount

	threeYearsSavingsFloat, err := strconv.ParseFloat(threeYearsSavings, 64)
	if err != nil {
		return 0, err
	}

	return threeYearsSavingsFloat - oneYearSavingsFloat, nil
}

func (s *AwsStandaloneService) getEntity(ctx context.Context, customerID, standaloneID string) (*firestore.DocumentRef, error) {
	return flexsavestandalone.GetEntity(ctx, customerID, standaloneID, s.entitiesDAL, s.customersDAL.GetCustomer, s.flexsaveStandaloneDAL.GetStandalonePriorityID)
}

func (s *AwsStandaloneService) validateCustomerAccountID(ctx context.Context, customerID, accountID string) error {
	doc, err := s.flexsaveStandaloneDAL.GetAWSStandaloneOnboarding(ctx, s.composeStandaloneID(customerID, accountID))
	if err != nil {
		return err
	}

	if doc.AccountID != accountID {
		return errorBadAccountID
	}

	return nil
}

func GetAWSFlexsaveSlackChannel() string {
	if common.ProjectID == "me-doit-intl-com" {
		return prodSlackChannel
	}

	return devSlackChannel
}

func (s *AwsStandaloneService) publishConfigCreatedSlackNotification(ctx context.Context, accountID string, customerID string) error {
	payerAccount := s.getPayerAccountName(accountID)

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	channel := GetAWSFlexsaveSlackChannel()

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":tada: %v has just opted in to *Flexsave Standalone* with Payer Account: *%v*", customer.PrimaryDomain, payerAccount),
		},
		{
			"type":  slack.MarkdownType,
			"value": "New Payer Config Created <https://console.doit.com/flexsave-aws-operations|Here>",
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"color":  "#4CAF50",
				"fields": fields,
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
		return err
	}

	return nil
}

package aws

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/organizations"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
)

// getSavings get lastMonthComputeSpend & estimatedSavings calculations using AWS api
func (s *AwsStandaloneService) getSavings(ctx context.Context, accountID string, shortTerm bool) (map[string]float64, *pkg.AWSSavingsPlansRecommendation, error) {
	spRecommendations, err := s.getSavingsPlansPurchaseRecommendation(accountID, shortTerm)
	if err != nil {
		return nil, nil, err
	}

	currentOnDemandSpend := *spRecommendations.SavingsPlansPurchaseRecommendationSummary.CurrentOnDemandSpend
	estimatedMonthlySavingsAmount := *spRecommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedMonthlySavingsAmount

	lastMonthComputeSpend, err := strconv.ParseFloat(currentOnDemandSpend, 64)
	if err != nil {
		return nil, nil, err
	}

	estimatedSavings, err := strconv.ParseFloat(estimatedMonthlySavingsAmount, 64)
	if err != nil {
		return nil, nil, err
	}

	return map[string]float64{
			string(pkg.LastMonthComputeSpend): lastMonthComputeSpend,
			string(pkg.EstimatedSavings):      estimatedSavings,
			string(pkg.MonthlySavings):        estimatedSavings,
		},
		s.mapSavingsPlansRecommendations(spRecommendations),
		nil
}

// getSavingsPlansPurchaseRecommendation call the api with required params
func (s *AwsStandaloneService) getSavingsPlansPurchaseRecommendation(accountID string, shortTerm bool) (*costexplorer.SavingsPlansPurchaseRecommendation, error) {
	input := costexplorer.GetSavingsPlansPurchaseRecommendationInput{
		AccountScope:         &payer,
		TermInYears:          &oneYear,
		PaymentOption:        &noUpfront,
		LookbackPeriodInDays: &thirtyDays,
		SavingsPlansType:     &computeSP,
	}

	if !shortTerm {
		input.TermInYears = &threeYears
	}

	output, err := s.AWSAccess.GetSavingsPlansPurchaseRecommendation(input, accountID)
	if err != nil {
		return nil, s.handleAWSError(err)
	}

	if output == nil || output.SavingsPlansPurchaseRecommendation == nil || output.SavingsPlansPurchaseRecommendation.SavingsPlansPurchaseRecommendationSummary == nil {
		return nil, flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypeSavings, errorEmptySavingsPlans)
	}

	return output.SavingsPlansPurchaseRecommendation, nil
}

// validateCUR check that the given AWS account has a valid Cost and Usage Report
func (s *AwsStandaloneService) validateCUR(ctx context.Context, customerID, accountID, s3Bucket, curPath string) error {
	logger := s.getLogger(ctx, customerID)

	session, err := s.AWSAccess.GetAWSSession(accountID, functionDescribeReportDefinitions)
	if err != nil {
		return s.handleAWSError(err)
	}

	config := aws.Config{Region: aws.String(endpoints.UsEast1RegionID)}
	curService := costandusagereportservice.New(session, &config)

	output, err := curService.DescribeReportDefinitions(nil)
	if err != nil {
		return s.handleAWSError(err)
	}

	if output == nil || len(output.ReportDefinitions) < 1 {
		return flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypeCUR, errorCUR)
	}

	var curFailureDescriptions []string

	for index, report := range output.ReportDefinitions {
		valid, failureDescription := s.validateSingleCUR(ctx, report, s3Bucket, curPath)
		if valid {
			return nil
		}

		if failureDescription != "" {
			curFailureDescriptions = append(curFailureDescriptions, fmt.Sprintf("%d. %s", index+1, failureDescription))
		}
	}

	logger.Error(errors.New("no CUR found which matches validations. (see info log below)"))
	logger.Info(curFailureDescriptions)

	return flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypeCUR, errorInvalidCUR)
}

/*
	validateSingleCUR

conditions:
- bucket —> report's S3Bucket = ‘s3Bucket’ (given by the user)
- path —> report's S3Prefix = ‘curPath’ (given by the user) OR report's ReportName = last part of ‘curPath’ (given by the user)
- properties —> report's TimeUnit = hourly AND report's Format = CSV AND report's AdditionalSchemaElements contains RESOURCES

examples:
  - report prefix = "CUR", name = "doitintl-awsops102".
    Valid curPaths:
  - prefix "CUR"
  - name "doitintl-awsops102"
  - full path "CUR/doitintl-awsops102"
  - report prefix = "CUR/doitintl-awsops102", name = "doitintl-awsops102".
    valid curPaths:
  - prefix "CUR/doitintl-awsops102"
  - name "doitintl-awsops102"
  - full path "CUR/doitintl-awsops102/doitintl-awsops102"
*/
func (s *AwsStandaloneService) validateSingleCUR(ctx context.Context, report *costandusagereportservice.ReportDefinition, s3Bucket, curPath string) (bool, string) {
	reportName := s.extractCUR(curPath)
	validBucket := *report.S3Bucket == s3Bucket
	validPath := curPath == *report.S3Prefix || reportName == *report.ReportName

	validTimeUnit := *report.TimeUnit == hourly
	validFormat := *report.Format == reportOrCsv
	validProperties := validTimeUnit && validFormat

	validSchema := false

	for _, element := range report.AdditionalSchemaElements {
		if *element == resources {
			validSchema = true
		}
	}

	if validBucket && validPath && validProperties && validSchema {
		return true, ""
	}

	return false, fmt.Sprintf(
		"CUR '%s' invalid.\ngiven parameters - s3Bucket: '%s', curPath: '%s'\nvalidation analysis - bucket: %t, path: %t, time unit: %t, format: %t, schema: %t\nreport: %+v\n",
		*report.ReportName,
		s3Bucket,
		curPath,
		validBucket,
		validPath,
		validTimeUnit,
		validFormat,
		validSchema,
		report,
	)
}

// getAssetProperties populate asset properties given AWS account
func (s *AwsStandaloneService) getAssetProperties(accountID string) (*assets.AWSProperties, error) {
	session, err := s.AWSAccess.GetAWSSession(accountID, functionDescribeAccount)
	if err != nil {
		return nil, s.handleAWSError(err)
	}

	organizationsService := organizations.New(session)
	describeAccountInput := &organizations.DescribeAccountInput{
		AccountId: aws.String(accountID),
	}

	output, err := organizationsService.DescribeAccount(describeAccountInput)
	if err != nil {
		return nil, s.handleAWSError(err)
	}

	if output == nil || output.Account == nil {
		return nil, errorEmptyDescribeAccount
	}

	awsAccount := output.Account

	return &assets.AWSProperties{
		AccountID:    *awsAccount.Id,
		Name:         *awsAccount.Name,
		FriendlyName: *awsAccount.Id,
		OrganizationInfo: &assets.OrganizationInfo{
			PayerAccount: &domain.PayerAccount{
				AccountID:   *awsAccount.Id,
				DisplayName: s.getPayerAccountName(*awsAccount.Id),
			},
			Status: *awsAccount.Status,
			Email:  *awsAccount.Email,
		},
	}, nil
}

// mapSavingsPlansRecommendations from costexplorer struct to firestore/pkg struct
func (s *AwsStandaloneService) mapSavingsPlansRecommendations(recommendations *costexplorer.SavingsPlansPurchaseRecommendation) *pkg.AWSSavingsPlansRecommendation {
	parsedRecommendations := &pkg.AWSSavingsPlansRecommendation{
		AccountScope:         *recommendations.AccountScope,
		LookbackPeriodInDays: *recommendations.LookbackPeriodInDays,
		PaymentOption:        *recommendations.PaymentOption,
		SavingsPlansPurchaseRecommendationDetails: []*pkg.AWSSavingsPlansRecommendationDetail{},
		SavingsPlansPurchaseRecommendationSummary: &pkg.AWSSavingsPlansRecommendationSummary{
			CurrencyCode:                               *recommendations.SavingsPlansPurchaseRecommendationSummary.CurrencyCode,
			CurrentOnDemandSpend:                       *recommendations.SavingsPlansPurchaseRecommendationSummary.CurrentOnDemandSpend,
			DailyCommitmentToPurchase:                  *recommendations.SavingsPlansPurchaseRecommendationSummary.DailyCommitmentToPurchase,
			EstimatedMonthlySavingsAmount:              *recommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedMonthlySavingsAmount,
			EstimatedOnDemandCostWithCurrentCommitment: *recommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedOnDemandCostWithCurrentCommitment,
			EstimatedROI:                               *recommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedROI,
			EstimatedSavingsAmount:                     *recommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedSavingsAmount,
			EstimatedSavingsPercentage:                 *recommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedSavingsPercentage,
			EstimatedTotalCost:                         *recommendations.SavingsPlansPurchaseRecommendationSummary.EstimatedTotalCost,
			HourlyCommitmentToPurchase:                 *recommendations.SavingsPlansPurchaseRecommendationSummary.HourlyCommitmentToPurchase,
			TotalRecommendationCount:                   *recommendations.SavingsPlansPurchaseRecommendationSummary.TotalRecommendationCount,
		},
		SavingsPlansType: *recommendations.SavingsPlansType,
		TermInYears:      *recommendations.TermInYears,
	}

	for _, detail := range recommendations.SavingsPlansPurchaseRecommendationDetails {
		awsSavingsPlansRecommendationDetail := &pkg.AWSSavingsPlansRecommendationDetail{
			AccountID:                                  *detail.AccountId,
			CurrencyCode:                               *detail.CurrencyCode,
			CurrentAverageHourlyOnDemandSpend:          *detail.CurrentAverageHourlyOnDemandSpend,
			CurrentMaximumHourlyOnDemandSpend:          *detail.CurrentMaximumHourlyOnDemandSpend,
			CurrentMinimumHourlyOnDemandSpend:          *detail.CurrentMinimumHourlyOnDemandSpend,
			EstimatedAverageUtilization:                *detail.EstimatedAverageUtilization,
			EstimatedMonthlySavingsAmount:              *detail.EstimatedMonthlySavingsAmount,
			EstimatedOnDemandCost:                      *detail.EstimatedOnDemandCost,
			EstimatedOnDemandCostWithCurrentCommitment: *detail.EstimatedOnDemandCostWithCurrentCommitment,
			EstimatedROI:                               *detail.EstimatedROI,
			EstimatedSPCost:                            *detail.EstimatedSPCost,
			EstimatedSavingsAmount:                     *detail.EstimatedSavingsAmount,
			EstimatedSavingsPercentage:                 *detail.EstimatedSavingsPercentage,
			HourlyCommitmentToPurchase:                 *detail.HourlyCommitmentToPurchase,
			SavingsPlansDetails: &pkg.AWSSavingsPlansDetails{
				InstanceFamily: *detail.SavingsPlansDetails.InstanceFamily,
				OfferingID:     *detail.SavingsPlansDetails.OfferingId,
				Region:         *detail.SavingsPlansDetails.Region,
			},
			UpfrontCost: *detail.UpfrontCost,
		}

		parsedRecommendations.SavingsPlansPurchaseRecommendationDetails = append(parsedRecommendations.SavingsPlansPurchaseRecommendationDetails, awsSavingsPlansRecommendationDetail)
	}

	return parsedRecommendations
}

// handleAWSError detects pre-known aws error types
func (s *AwsStandaloneService) handleAWSError(err error) error {
	var awsErr awserr.Error
	if errors.As(err, &awsErr) {
		switch string(awsErr.Code()) {
		case "AccessDenied", "AccessDeniedException":
			return flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypePermissions, err)
		case "ValidationException":
			if strings.Contains(awsErr.Error(), "LINKED") {
				return flexsavestandalone.BuildStandaloneError(pkg.OnboardingErrorTypeLinkedAccount, err)
			}
		default:
			return err
		}
	}

	return err
}

package aws

import (
	"context"
	"fmt"
	"strconv"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
)

type OnboardingDocAttributes struct {
	Savings         map[string]float64
	Recommendations map[string]*pkg.AWSSavingsPlansRecommendation
	CustomerID      string
	AccountID       string
}

func (s *AwsStandaloneService) UpdateSavingsAndRecommendationsCSV(ctx context.Context, recommendationCSV EstimationSummaryCSVRequest, customerID, accountID string) error {
	savings, err := getThreeYearsSavings(recommendationCSV)
	if err != nil {
		return err
	}

	recommendations := map[string]*pkg.AWSSavingsPlansRecommendation{
		threeYears: &pkg.AWSSavingsPlansRecommendation{
			AccountScope:         payer,
			TermInYears:          threeYears,
			PaymentOption:        noUpfront,
			LookbackPeriodInDays: thirtyDays,
			SavingsPlansType:     computeSP,
			SavingsPlansPurchaseRecommendationDetails: []*pkg.AWSSavingsPlansRecommendationDetail{
				{
					AccountID:                         accountID,
					CurrentAverageHourlyOnDemandSpend: recommendationCSV.CurrentAverageHourlyOnDemandSpend,
					CurrentMaximumHourlyOnDemandSpend: recommendationCSV.CurrentMaximumHourlyOnDemandSpend,
					CurrentMinimumHourlyOnDemandSpend: recommendationCSV.CurrentMinimumHourlyOnDemandSpend,
					EstimatedAverageUtilization:       recommendationCSV.EstimatedAverageUtilization,
					EstimatedMonthlySavingsAmount:     recommendationCSV.EstimatedMonthlySavingsAmount,
					EstimatedOnDemandCost:             recommendationCSV.EstimatedOnDemandCost,
					EstimatedROI:                      recommendationCSV.EstimatedROI,
					EstimatedSPCost:                   recommendationCSV.EstimatedSavingsPlansCost,
					EstimatedSavingsPercentage:        recommendationCSV.EstimatedSavingsPercentage,
					HourlyCommitmentToPurchase:        recommendationCSV.HourlyCommitmentToPurchase,
				},
			},
			SavingsPlansPurchaseRecommendationSummary: &pkg.AWSSavingsPlansRecommendationSummary{
				CurrentOnDemandSpend:          fmt.Sprint(savings["lastMonthComputeSpend"]),
				EstimatedMonthlySavingsAmount: recommendationCSV.EstimatedMonthlySavingsAmount,
				EstimatedROI:                  recommendationCSV.EstimatedROI,
				EstimatedSavingsPercentage:    recommendationCSV.EstimatedSavingsPercentage,
				HourlyCommitmentToPurchase:    recommendationCSV.HourlyCommitmentToPurchase,
			},
		},
	}

	return s.CreateOrUpdateOnboardingDoc(ctx, OnboardingDocAttributes{
		Savings:         savings,
		Recommendations: recommendations,
		CustomerID:      customerID,
		AccountID:       accountID,
	})
}

func getThreeYearsSavings(recommendationCSV EstimationSummaryCSVRequest) (map[string]float64, error) {
	const (
		savingsCoverage = 0.85
		thirtyDays      = 30
		twentyFourHrs   = 24
	)

	estimatedSavings, err := strconv.ParseFloat(recommendationCSV.EstimatedMonthlySavingsAmount, 64)
	if err != nil {
		return nil, err
	}

	utilization, err := strconv.ParseFloat(recommendationCSV.EstimatedAverageUtilization, 64)
	if err != nil {
		return nil, err
	}

	threeYearSavings := estimatedSavings * (utilization / 100) * (savingsCoverage / 2)

	currentAverageODSpend, err := strconv.ParseFloat(recommendationCSV.CurrentAverageHourlyOnDemandSpend, 64)
	if err != nil {
		return nil, err
	}

	currentODSpend := thirtyDays * twentyFourHrs * currentAverageODSpend

	savings := map[string]float64{
		string(pkg.LastMonthComputeSpend): currentODSpend,
		string(pkg.EstimatedSavings):      threeYearSavings,
		string(pkg.MonthlySavings):        threeYearSavings,
	}

	return savings, nil
}

func (s *AwsStandaloneService) CreateOrUpdateOnboardingDoc(ctx context.Context, onboardingAttributes OnboardingDocAttributes) error {
	customerID := onboardingAttributes.CustomerID
	customerRef := s.customersDAL.GetRef(ctx, customerID)
	standaloneID := s.composeStandaloneID(customerID, onboardingAttributes.AccountID)

	err := s.flexsaveStandaloneDAL.InitStandaloneOnboarding(ctx, fmt.Sprintf("amazon-web-services-%s", customerID), customerRef)
	if err != nil {
		if err == pkg.ErrorAlreadyExist {
			return s.flexsaveStandaloneDAL.UpdateStandaloneOnboarding(ctx, standaloneID, onboardingUpdateFields(onboardingAttributes, customerRef))
		}

		return err
	}

	return s.flexsaveStandaloneDAL.UpdateStandaloneOnboarding(ctx, standaloneID, onboardingUpdateFields(onboardingAttributes, customerRef))
}

func onboardingUpdateFields(onboardingAttributes OnboardingDocAttributes, customerRef *firestore.DocumentRef) *pkg.AWSStandaloneOnboarding {
	accountID := onboardingAttributes.AccountID

	return &pkg.AWSStandaloneOnboarding{
		AccountID: accountID,
		BaseStandaloneOnboarding: pkg.BaseStandaloneOnboarding{
			Savings: onboardingAttributes.Savings,
		},
		Recommendations:  onboardingAttributes.Recommendations,
		IsMissingAWSRole: true,
	}
}

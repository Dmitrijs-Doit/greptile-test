package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/credits"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"

	fsdal "github.com/doitintl/firestore"
	fspkg "github.com/doitintl/firestore/pkg"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/recommendations"
	consts "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	newlyOnBoardedCutOffDayNumber = 3
	dailyCutOffDay                = 10
	dailyHours                    = 24
)

type DedicatedPayerService struct {
	loggerProvider logger.Provider
	*connection.Connection

	IntegrationsDAL        fsdal.Integrations
	bigQueryService        bq.BigQueryServiceInterface
	recommendationsService recommendations.Recommendations
	mpaDAL                 dal.MasterPayerAccounts
}

func NewDedicatedPayerService(log logger.Provider, conn *connection.Connection) *DedicatedPayerService {
	bigQueryService, err := bq.NewBigQueryService()
	if err != nil {
		panic(err)
	}

	integrationsDAL := fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background()))

	recommendationsService, err := recommendations.NewFlexAPIService()
	if err != nil {
		panic(err)
	}

	mpaDAL := dal.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background()))

	return &DedicatedPayerService{
		log,
		conn,
		integrationsDAL,
		bigQueryService,
		recommendationsService,
		mpaDAL,
	}
}

func (s *DedicatedPayerService) GetCache(ctx context.Context, configInfo pkg.CustomerInputAttributes, timeParams pkg.TimeParams) (*fspkg.FlexsaveSavings, error) {
	if len(configInfo.AssetIDs) == 0 {
		return nil, nil
	}

	log := s.loggerProvider(ctx)
	threeDaysAgo := timeParams.Now.AddDate(0, 0, -newlyOnBoardedCutOffDayNumber)

	savingsHistory, dailySavingsHistory, err := s.createSavingsHistory(ctx, configInfo, timeParams.Now)
	if err == bq.ErrNoActiveTable && len(configInfo.PayerIDs) == 0 {
		return nil, nil
	} else if err == bq.ErrNoActiveTable && configInfo.DedicatedPayerStartTime != nil && configInfo.DedicatedPayerStartTime.After(threeDaysAgo) {
		log.Infof("active billing table does not exist for customer %v", configInfo.CustomerID)
	} else if err != nil {
		return nil, err
	}

	reasonCantEnable := noError

	var savingsSummary fspkg.FlexsaveSavingsSummary

	if !configInfo.IsEnabled {
		var err error

		savingsSummary, err = s.createSavingsSummary(ctx, configInfo.PayerIDs, configInfo.CustomerID, log)
		if err != nil {
			reasonCantEnable = errFetchingRecommendations
		}

		if reasonCantEnable == noError {
			reasonCantEnable = getDedicatedPayerReasonCantEnable(savingsSummary)
		}

		if reasonCantEnable == noError {
			hasCredits, err := s.checkIfCustomerHasAwsActivateCredits(ctx, configInfo.CustomerID, time.Now())
			if err != nil {
				log.Errorf("could not fetch credit for customer %s, error occurred: %s", configInfo.CustomerID, err.Error())

				reasonCantEnable = errOther
			} else if hasCredits {
				reasonCantEnable = credits.ErrCustomerHasAwsActivateCredits
			}
		}
	}

	data := &fspkg.FlexsaveSavings{
		DailySavingsHistory: dailySavingsHistory,
		SavingsHistory:      savingsHistory,
		SavingsSummary:      &savingsSummary,
		ReasonCantEnable:    reasonCantEnable,
	}

	return data, nil
}

// required limit for 30d is -18 and for 7d is -4.2
func checkHasGoodSpend(totalWaste, timeframeHours, averageSavingsRate float64) bool {
	limit := -(timeframeHours * dailyHours * 0.5 * 0.05)
	return totalWaste > limit && averageSavingsRate > 0.1
}

func (s *DedicatedPayerService) createSavingsSummary(
	ctx context.Context, payerIDs []string, customerID string, log logger.ILogger,
) (fspkg.FlexsaveSavingsSummary, error) {
	var estimateSavings float64

	hourlyCommitment := 0.0

	savingsSummary := fspkg.FlexsaveSavingsSummary{
		CurrentMonth:                     &fspkg.FlexsaveCurrentMonthSummary{},
		NextMonth:                        &fspkg.FlexsaveMonthSummary{},
		CanBeEnabledBasedOnArtifactSpend: false,
	}

	for _, id := range payerIDs {
		payerRecommendations, err := s.recommendationsService.FetchComputeRecommendations(ctx, id, time.Now().UTC())
		if err != nil {
			if !strings.Contains(err.Error(), "Not found: Table") {
				log.Errorf(
					"error getting artifact recommendations for customer: %v with payer account: %v, with error: %v",
					customerID, id, err.Error())

				return savingsSummary, err
			}

			mpa, mpaErr := s.mpaDAL.GetMasterPayerAccount(ctx, id)
			if mpaErr != nil {
				return savingsSummary, mpaErr
			}

			noTableThreshold := mpa.OnboardingDate.AddDate(0, 0, 2)

			if mpa.OnboardingDate != nil && time.Now().After(noTableThreshold) {
				log.Errorf("table not found, customer %s, payer account: %s, onboarding date: %v, err: [%v]", customerID, id, mpa.OnboardingDate, err)
			} else {
				log.Warningf("table not found, customer %s, payer account: %s, onboarding date: %v, err: [%v]", customerID, id, mpa.OnboardingDate, err)
			}

			return savingsSummary, err
		}

		if payerRecommendations != nil {
			if *payerRecommendations.HourlyCommitmentToPurchase > hourlyCommitment {
				hourlyCommitment = *payerRecommendations.HourlyCommitmentToPurchase
			}

			hasGoodSpend := false

			if (payerRecommendations.TotalWaste != nil) && (payerRecommendations.AverageSavingsRate != nil) {
				hasGoodSpend = checkHasGoodSpend(
					*payerRecommendations.TotalWaste,
					float64(payerRecommendations.TimeframeHours),
					*payerRecommendations.AverageSavingsRate)
			}

			if hasGoodSpend {
				savingsSummary.CanBeEnabledBasedOnArtifactSpend = true
			}

			if hasGoodSpend || *payerRecommendations.HourlyCommitmentToPurchase > consts.MinimumHourlyCommitmentToPurchase {
				estimateSavings += *payerRecommendations.EstimatedSavingsAmount
			}
		}
	}

	savingsSummary.NextMonth.HourlyCommitment = &hourlyCommitment
	savingsSummary.NextMonth.Savings = estimateSavings

	return savingsSummary, nil
}

func getDedicatedPayerReasonCantEnable(savingsSummary fspkg.FlexsaveSavingsSummary) string {
	if savingsSummary.NextMonth.HourlyCommitment == nil {
		return errNoSpend
	}

	if *savingsSummary.NextMonth.HourlyCommitment > consts.MinimumHourlyCommitmentToPurchase {
		return noError
	}

	if *savingsSummary.NextMonth.HourlyCommitment < consts.MinimumHourlyCommitmentToPurchase {
		return errLowSpend
	}

	if savingsSummary.CanBeEnabledBasedOnArtifactSpend {
		return noError
	}

	return errLowSpend
}

func (s *DedicatedPayerService) createSavingsHistory(ctx context.Context, configInfo pkg.CustomerInputAttributes, now time.Time) (map[string]*fspkg.FlexsaveMonthSummary, map[string]*fspkg.FlexsaveMonthSummary, error) {
	if !configInfo.IsEnabled {
		return nil, nil, nil
	}

	err := s.bigQueryService.CheckActiveBillingTableExists(ctx, configInfo.CustomerID)
	if err != nil {
		return nil, nil, err
	}

	monthlySavings, err := s.createMonthlySavingsHistory(ctx, configInfo.CustomerID, now)
	if err != nil {
		return nil, nil, err
	}

	dailySavings, err := s.createDailySavingsHistory(ctx, configInfo.CustomerID, now)
	if err != nil {
		return monthlySavings, nil, err
	}

	return monthlySavings, dailySavings, err
}

func (s *DedicatedPayerService) createMonthlySavingsHistory(ctx context.Context, customerID string, now time.Time) (map[string]*fspkg.FlexsaveMonthSummary, error) {
	var spendSummary map[string]*fspkg.FlexsaveMonthSummary

	params := bq.BigQueryParams{
		Context:             ctx,
		CustomerID:          customerID,
		FirstOfCurrentMonth: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
		NumberOfMonths:      flexsaveHistoryMonthAmount,
	}

	spendSummary, err := s.bigQueryService.GetPayerSpendSummary(params)
	if err != nil {
		return spendSummary, err
	}

	return spendSummary, nil
}

func (s *DedicatedPayerService) createDailySavingsHistory(ctx context.Context, customerID string, now time.Time) (map[string]*fspkg.FlexsaveMonthSummary, error) {
	cache, err := s.IntegrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("shouldRunDailyCache() for customer: %s, err: %w", customerID, err)
	}

	if !shouldRunDailyCache(cache, time.Now()) {
		return nil, nil
	}

	start, end := getDaysWhenDailyIsApplicable(*cache.AWS.TimeEnabled, dailyCutOffDay)

	dailySavings, err := s.bigQueryService.GetPayerDailySpendSummary(bq.DailyBQParams{
		Context:    ctx,
		CustomerID: customerID,
		Start:      start,
		End:        end,
	})
	if err != nil {
		return nil, fmt.Errorf("GetPayerDailySpendSummary() for customer: %s, err: %w", customerID, err)
	}

	return dailySavings, nil
}

func shouldRunDailyCache(cache *fspkg.FlexsaveConfiguration, now time.Time) bool {
	if cache.AWS.TimeEnabled == nil {
		return false
	}

	dailyStart, dailyEnd := getDaysWhenDailyIsApplicable(*cache.AWS.TimeEnabled, dailyCutOffDay)

	return (now.After(dailyStart) || now.Equal(dailyStart)) && (now.Before(dailyEnd) || now.Equal(dailyEnd))
}

func getDaysWhenDailyIsApplicable(timeEnabled time.Time, dailyCutOffDay int) (start time.Time, end time.Time) {
	return timeEnabled, time.Date(timeEnabled.Year(), timeEnabled.Month()+1, dailyCutOffDay, 0, 0, 0, 0, timeEnabled.Location())
}

func (s *DedicatedPayerService) checkIfCustomerHasAwsActivateCredits(ctx context.Context, customerID string, now time.Time) (bool, error) {
	creditResults := s.bigQueryService.GetCustomerCredits(ctx, customerID, now)

	if creditResults.Err != nil {
		return false, creditResults.Err
	}

	for k, v := range creditResults.Credits {
		if k == customerID && v < utils.CreditLimitForFlexsaveEnablement {
			return true, nil
		}
	}

	return false, nil
}

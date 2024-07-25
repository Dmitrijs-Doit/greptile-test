package aws

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/iterator"
)

type BigQueryRequestParams struct {
	CustomerID       string
	AccountIDs       []string
	Start            time.Time
	NumberOfMonth    int
	ForecastFromTime time.Time
}

type FlexsaveStandaloneData struct {
	Spend          map[string]*pkg.FlexsaveMonthSummary `json:"spend"`
	SavingsSummary *pkg.FlexsaveSavingsSummary          `json:"savingsSummary"`
}

type DataResults struct {
	Daily map[string]map[string]float64 `json:"daily"`
	Cost  months                        `json:"cost"`
}

type spendResult struct {
	onDemandResults    *DataResults
	savingsResults     *DataResults
	savingsRateResults months
}

type forecast struct {
	savings  float64
	onDemand float64
}

type months map[string]float64

type itemType struct {
	PayerID string    `bigquery:"payer_id"`
	Cost    float64   `bigquery:"cost"`
	Date    time.Time `bigquery:"usage_date"`
}

type timeParams struct {
	Now              time.Time
	CurrentMonth     string
	ApplicableMonths []string
	PreviousMonth    string
}

var (
	minNumberOfDays     = 2
	numOfDaysToForecast = 30
	dateFormat          = "2006-01-02"
	monthToYear         = "1_2006"
)

func (s *AwsStandaloneService) GetCustomerSpend(ctx context.Context, queryParams BigQueryRequestParams, timeParams timeParams) (spendResult, error) {
	logger := s.loggerProvider(ctx)

	var sr spendResult

	err := s.checkBillingTableExists(ctx, queryParams.CustomerID)
	if err != nil {
		return sr, err
	}

	errChan := make(chan error)
	onDemandChan := make(chan *DataResults)
	savingsChan := make(chan *DataResults)
	recurringFeeChan := make(chan *DataResults)

	var onDemandResults *DataResults

	var savingsResults *DataResults

	var recurringFeeResults *DataResults

	onDemandQuery := s.buildOnDemandCostEquivalentQuery(queryParams)
	savingsQuery := s.buildStandaloneSavingsQuery(queryParams)
	recurringFeeQuery := s.buildStandaloneSavingsRecurringFeeQuery(queryParams)

	processNumber := 3

	go s.execSpendQuery(ctx, onDemandQuery, queryParams, onDemandChan, errChan)
	go s.execSpendQuery(ctx, savingsQuery, queryParams, savingsChan, errChan)
	go s.execSpendQuery(ctx, recurringFeeQuery, queryParams, recurringFeeChan, errChan)

	for i := 0; i < processNumber; i++ {
		select {
		case onDemand := <-onDemandChan:
			onDemandResults = onDemand
		case savings := <-savingsChan:
			savingsResults = savings
		case recurringFee := <-recurringFeeChan:
			recurringFeeResults = recurringFee
		case err := <-errChan:
			logger.Error(err)
		}
	}

	savingsResults = calculateSavings(savingsResults, recurringFeeResults, timeParams)

	savingsRateResults := make(map[string]float64)

	if onDemandResults != nil && savingsResults != nil {
		for month := range onDemandResults.Cost {
			onDemandResults.Cost[month] -= savingsResults.Cost[month]
			savingsRateResults[month] = getSavingsRate(savingsResults.Cost[month], onDemandResults.Cost[month]+savingsResults.Cost[month])
		}
	}

	return spendResult{
		onDemandResults:    onDemandResults,
		savingsResults:     savingsResults,
		savingsRateResults: savingsRateResults,
	}, nil
}

func (s *AwsStandaloneService) buildFlexsaveSpendSummary(ctx context.Context, spend *spendResult, forecastResults forecast, timeParams timeParams) (*FlexsaveStandaloneData, error) {
	logger := s.loggerProvider(ctx)

	nextMonthFlexsaveMonthSummary, err := s.getNextMonthFlexaveMonthSummary(ctx, spend, forecastResults)
	if err != nil {
		logger.Errorf("Error getting flexsave month summary: %v", err)
		return nil, err
	}

	projectedSavings, err := s.getProjectedSavings(spend, nextMonthFlexsaveMonthSummary.Savings, timeParams.CurrentMonth)
	if err != nil {
		logger.Errorf("Error getting projected savings: %v", err)
		return nil, err
	}

	savingsHistory := make(map[string]*pkg.FlexsaveMonthSummary)
	for _, month := range timeParams.ApplicableMonths {
		savingsHistory[month] = &pkg.FlexsaveMonthSummary{
			OnDemandSpend: spend.onDemandResults.Cost[month],
			Savings:       spend.savingsResults.Cost[month],
			SavingsRate:   spend.savingsRateResults[month],
		}
	}

	return &FlexsaveStandaloneData{
		Spend: savingsHistory,
		SavingsSummary: &pkg.FlexsaveSavingsSummary{
			CurrentMonth: &pkg.FlexsaveCurrentMonthSummary{
				Month:            timeParams.CurrentMonth,
				ProjectedSavings: projectedSavings,
			},
			NextMonth: nextMonthFlexsaveMonthSummary,
		},
	}, nil
}

func (s *AwsStandaloneService) getNextMonthFlexaveMonthSummary(ctx context.Context, spend *spendResult, forecastResults forecast) (*pkg.FlexsaveMonthSummary, error) {
	var (
		onDemandForecast float64
		savingsForecast  float64
	)

	onDemandForecast = forecastResults.onDemand
	savingsForecast = forecastResults.savings

	for _, dateSpend := range spend.onDemandResults.Daily {
		if len(dateSpend) >= minNumberOfDays {
			onDemandForecastVal, err := s.getForecast(ctx, dateSpend)
			if err != nil {
				return nil, err
			}

			onDemandForecast += onDemandForecastVal
		}
	}

	for _, dateSavings := range spend.savingsResults.Daily {
		if len(dateSavings) >= minNumberOfDays {
			savingsForecastVal, err := s.getForecast(ctx, dateSavings)
			if err != nil {
				return nil, err
			}

			savingsForecast += savingsForecastVal
		}
	}

	return &pkg.FlexsaveMonthSummary{
		OnDemandSpend: onDemandForecast,
		Savings:       savingsForecast,
		SavingsRate:   getSavingsRate(savingsForecast, onDemandForecast+savingsForecast),
	}, nil
}

func GetDaysInNextMonth() float64 {
	firstDayOfNextMonth := GetFirstDayOfNextMonth()
	firstDayOfMonthAfterNext := firstDayOfNextMonth.AddDate(0, 1, 0)

	return firstDayOfMonthAfterNext.Sub(firstDayOfNextMonth).Hours() / 24
}

func GetDaysInCurrentMonth() float64 {
	firstDayOfNextMonth := GetFirstDayOfNextMonth()
	firstDayOfThisMonth := firstDayOfNextMonth.AddDate(0, -1, 0)

	return firstDayOfNextMonth.Sub(firstDayOfThisMonth).Hours() / 24
}

func GetFirstDayOfNextMonth() time.Time {
	now := time.Now().UTC()
	nextMonth := time.Date(now.Year(), now.Month()+2, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)

	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func getCurrentMonthProjectedSavings(savingsSoFar float64, nextMonthSavings float64, isEnabled bool, lastMonthSavings float64, isEnabledDedicatedPayer bool) (float64, error) {
	if savingsSoFar == 0 && nextMonthSavings == 0 && !isEnabledDedicatedPayer || savingsSoFar == 0 && !isEnabled {
		return 0, nil
	}

	if savingsSoFar == 0 && nextMonthSavings == 0 {
		return lastMonthSavings, nil
	}

	numberOfFullDaysToDisregard := 3.0

	if isEnabledDedicatedPayer {
		numberOfFullDaysToDisregard = 0
	}

	now := time.Now().UTC()
	nextMonth := time.Date(now.Year(), now.Month()+2, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	firstDayOfNextMonth := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	daysInNextMonth := GetDaysInNextMonth()
	daysInCurrentMonth := GetDaysInCurrentMonth()

	// We calculate monthly savings up to 48 hours before today so need to create projection from this same point
	daysUntilEndOfMonthWithOffSet := (firstDayOfNextMonth.Sub(now).Hours() / 24) + numberOfFullDaysToDisregard
	// Maximum number of days used is the number of days in current month
	if daysUntilEndOfMonthWithOffSet > daysInCurrentMonth {
		daysUntilEndOfMonthWithOffSet = daysInCurrentMonth
	}

	daysWithValidData := float64(now.Day()) - numberOfFullDaysToDisregard

	// If we have no savings data for current month at a time when we would expect to return zero for projection.
	if savingsSoFar == 0 && daysWithValidData > 0 {
		return 0, nil
	}

	savings := savingsSoFar

	// Before we have actual savings data for current month use next month savings potential
	// adjusted to account for difference in month length.
	if savingsSoFar == 0 || daysWithValidData < 1 {
		savings = nextMonthSavings
		daysWithValidData = daysInNextMonth
	}

	savingsSoFar += (savings / float64(daysWithValidData)) * daysUntilEndOfMonthWithOffSet

	return savingsSoFar, nil
}

func (s *AwsStandaloneService) getProjectedSavings(spend *spendResult, nextMonthSavings float64, currentMonth string) (float64, error) {
	return getCurrentMonthProjectedSavings(
		spend.savingsResults.Cost[currentMonth],
		nextMonthSavings,
		true,
		0,
		true)
}

func (s *AwsStandaloneService) validateSpendResults(spend *spendResult, accounts []string, currentMonth string) (*spendResult, forecast, error) {
	var forecastResults forecast

	for _, account := range accounts {
		if spend.onDemandResults != nil && len(spend.onDemandResults.Daily[account]) >= minNumberOfDays {
			return spend, forecastResults, nil
		}

		res, err := s.getSavingsPlansPurchaseRecommendation(account, true)
		if err != nil {
			return nil, forecastResults, err
		}

		onDemandSpend, err := strconv.ParseFloat(*res.SavingsPlansPurchaseRecommendationSummary.CurrentOnDemandSpend, 64)
		if err != nil {
			return nil, forecastResults, err
		}

		onDemandForecast, err := strconv.ParseFloat(*res.SavingsPlansPurchaseRecommendationSummary.EstimatedOnDemandCostWithCurrentCommitment, 64)
		if err != nil {
			return nil, forecastResults, err
		}

		savingsForecast, err := strconv.ParseFloat(*res.SavingsPlansPurchaseRecommendationSummary.EstimatedSavingsAmount, 64)
		if err != nil {
			return nil, forecastResults, err
		}

		if spend.onDemandResults == nil {
			spend.onDemandResults = &DataResults{}
			spend.onDemandResults.Cost = make(map[string]float64)
		}

		if spend.savingsResults == nil {
			spend.savingsResults = &DataResults{}
			spend.savingsResults.Cost = make(map[string]float64)
		}

		if spend.savingsRateResults == nil {
			spend.savingsRateResults = make(map[string]float64)
		}

		spend.onDemandResults.Cost[currentMonth] += onDemandSpend
		spend.savingsResults.Cost[currentMonth] += 0
		spend.savingsRateResults[currentMonth] += 0
		forecastResults.onDemand += onDemandForecast - savingsForecast
		forecastResults.savings += savingsForecast
	}

	return spend, forecastResults, nil
}

func (s *AwsStandaloneService) getForecast(ctx context.Context, data map[string]float64) (float64, error) {
	if len(data) == 0 {
		return 0, nil
	}

	keys := make([]string, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	dailyForecast := make(map[string]float64)
	numberOfDays := numOfDaysToForecast
	exclude3days := time.Now().UTC().AddDate(0, 0, -3)

	if len(keys) < numOfDaysToForecast {
		numberOfDays = len(keys)
	}

	for _, k := range keys[len(keys)-numberOfDays:] {
		day, err := time.Parse(dateFormat, k)
		if err != nil {
			continue
		}

		if day.Before(exclude3days) {
			dailyForecast[k] = data[k]
		}
	}

	if len(dailyForecast) == 0 {
		return 0, nil
	}

	return GetForecast(ctx, dailyForecast)
}

func (s *AwsStandaloneService) execSpendQuery(ctx context.Context, queryString string, params BigQueryRequestParams, dataResults chan *DataResults, errChan chan error) {
	startOfTheMonth := time.Date(s.now().Year(), s.now().Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := getEndDate(startOfTheMonth, s.now())
	query := s.bigQueryClient.Query(queryString)

	query.Labels = map[string]string{
		common.LabelKeyHouse.String():    common.HouseAdoption.String(),
		common.LabelKeyEnv.String():      common.GetEnvironmentLabel(),
		common.LabelKeyFeature.String():  "flexsave",
		common.LabelKeyModule.String():   "standalone-spend-summary",
		common.LabelKeyCustomer.String(): strings.ToLower(params.CustomerID),
	}

	query.Parameters = []bigquery.QueryParameter{
		{Name: "start", Value: startOfTheMonth.AddDate(0, -params.NumberOfMonth, 0).Format(dateFormat)},
		{Name: "end", Value: endDate},
	}

	iter, err := s.queryHandler.Read(ctx, query)
	if err != nil {
		errChan <- err
		return
	}

	var row itemType

	dailySaved := make(map[string]map[string]float64)
	monthlySaved := make(map[string]float64)

	for {
		err = iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			errChan <- err
			break
		}

		month := row.Date.Format(monthToYear)
		day := row.Date.Format(dateFormat)
		payerID := row.PayerID

		if _, ok := dailySaved[payerID]; !ok {
			dailySaved[payerID] = make(map[string]float64)
		}

		monthlySaved[month] += row.Cost

		if row.Date.After(params.ForecastFromTime) {
			dailySaved[payerID][day] = row.Cost
		}
	}

	dataResults <- &DataResults{
		Cost:  monthlySaved,
		Daily: dailySaved,
	}
}

func calculateSavings(savings, recurringFee *DataResults, timeParams timeParams) *DataResults {
	if savings == nil || recurringFee == nil {
		return savings
	}

	currentMonthFee := 0.0
	previousMonthFees := make(map[string]float64)

	for payerID, dateSavings := range savings.Daily {
		for date, savingsDay := range dateSavings {
			savings.Daily[payerID][date] = math.Abs(savingsDay) - recurringFee.Daily[payerID][date]

			dateTime, err := time.Parse(dateFormat, date)
			if err != nil {
				continue
			}

			if dateTime.Month() == timeParams.Now.UTC().Month() && dateTime.Year() == timeParams.Now.UTC().Year() {
				currentMonthFee += recurringFee.Daily[payerID][date]
			}
		}
	}

	for month := range recurringFee.Cost {
		if month != timeParams.CurrentMonth {
			previousMonthFees[month] = recurringFee.Cost[month]
		}
	}

	savings.Cost[timeParams.CurrentMonth] = math.Abs(savings.Cost[timeParams.CurrentMonth]) - currentMonthFee

	for month, fee := range previousMonthFees {
		savings.Cost[month] = math.Abs(savings.Cost[month]) - fee
	}

	return savings
}

func getSavingsRate(savings float64, onDemandSpend float64) float64 {
	if savings > 0 && onDemandSpend > 0 {
		return savings / (onDemandSpend) * 100
	}

	return 0.0
}

func getEndDate(startDate time.Time, now time.Time) string {
	if now.Month() == startDate.Month() && now.Year() == startDate.Year() {
		return startDate.AddDate(0, 0, now.Day()).Format(dateFormat)
	}

	return startDate.AddDate(0, 1, 0).Format(dateFormat)
}

func GetForecast(ctx context.Context, dailyData map[string]float64) (float64, error) {
	if len(dailyData) == 0 {
		return 0.0, nil
	}

	daysInNextMonth := getDaysInNextMonth()

	totalDailyData := 0.0
	nonZeroOrValidValues := 0.0

	var isValidZero bool

	for _, val := range dailyData {
		if val == 0 && totalDailyData != 0 {
			isValidZero = true
		}

		if val != 0 || isValidZero {
			totalDailyData += val
			nonZeroOrValidValues++
		}
	}

	var averageDailyValue float64
	if nonZeroOrValidValues != 0 {
		averageDailyValue = totalDailyData / nonZeroOrValidValues
	}

	forecast := averageDailyValue * daysInNextMonth

	return forecast, nil
}

func getDaysInNextMonth() float64 {
	firstDayOfNextMonth := getFirstDayOfNextMonth()
	firstDayOfMonthAfterNext := firstDayOfNextMonth.AddDate(0, 1, 0)

	return firstDayOfMonthAfterNext.Sub(firstDayOfNextMonth).Hours() / 24
}

func getFirstDayOfNextMonth() time.Time {
	now := time.Now().UTC()
	nextMonth := time.Date(now.Year(), now.Month()+2, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)

	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
}

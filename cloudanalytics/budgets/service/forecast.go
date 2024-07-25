package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func getBudgetDuration(b *budget.Budget) *time.Duration {
	var duration time.Duration
	if b.Config.Type == budget.Fixed {
		duration = b.Config.EndPeriod.Sub(b.Config.StartPeriod)
	} else {
		now := time.Now()

		switch b.Config.TimeInterval {
		case report.TimeIntervalDay:
			duration = budget.DayDuration
		case report.TimeIntervalWeek:
			duration = budget.WeekDuration
		case report.TimeIntervalMonth:
			startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			endOfMonth := startOfMonth.AddDate(0, 1, 0)
			duration = endOfMonth.Sub(startOfMonth)
		case report.TimeIntervalQuarter:
			startOfQuarter := time.Date(now.Year(), time.Month(int(now.Month())/3)+1, 1, 0, 0, 0, 0, time.UTC)
			endOfQuarter := startOfQuarter.AddDate(0, 3, 0)
			duration = endOfQuarter.Sub(startOfQuarter)
		case report.TimeIntervalYear:
			startOfYear := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			endOfYear := startOfYear.AddDate(1, 0, 0)
			duration = endOfYear.Sub(startOfYear)
		}
	}

	return &duration
}

func (s *BudgetsService) getBudgetForecast(ctx context.Context, queryRequest *cloudanalytics.QueryRequest, queryResult *cloudanalytics.QueryResult, b *budget.Budget) ([]*budget.BudgetForecastPrediction, error) {
	l := s.loggerProvider(ctx)

	predictions := make([]*budget.BudgetForecastPrediction, 0)
	now := time.Now()
	metric := int(b.Config.Metric)
	maxFreshTime := now.UTC().Add(time.Hour * -36)

	forecastData, forecastRows, err := s.forecastService.GetForecastOriginAndResultRows(
		ctx,
		queryResult.Rows,
		0,
		queryRequest.Cols,
		string(report.TimeIntervalDayCumSum),
		metric,
		maxFreshTime,
		*queryRequest.TimeSettings.From,
		b.Config.EndPeriod,
	)
	if err != nil {
		return nil, err
	}

	var endPeriodForecastedValue float64

	zeroPeriod := false
	endPeriod := b.Config.EndPeriod

	if len(forecastData) == 0 {
		l.Infof("no forecasted data for budget name %s for customer %s", b.Name, b.Customer.ID)
	}

	if b.Config.Type == budget.Fixed && endPeriod.Before(now) && len(forecastData) > 0 &&
		forecastData[len(forecastData)-1].IsBeforePeriod(string(report.TimeIntervalDay), endPeriod) {
		endPeriod, err = time.Parse(times.YearMonthDayLayout, forecastData[len(forecastData)-1].DS)
		if err != nil {
			return nil, err
		}

		endPeriod = endPeriod.AddDate(0, 0, 1)
		if !endPeriod.After(b.Config.StartPeriod) {
			zeroPeriod = true
		}
	}

	if !zeroPeriod {
		endPeriodForecastedValue = s.getForecastedValueByDates(forecastRows, &b.Config.StartPeriod, &endPeriod)
	}

	prediction := budget.BudgetForecastPrediction{
		Date:  &b.Config.EndPeriod,
		Value: endPeriodForecastedValue,
	}
	b.Utilization.Forecasted = endPeriodForecastedValue

	predictions = append(predictions, &prediction)

	for i, alert := range b.Config.Alerts {
		if alert.Percentage > 0 {
			alertAmount := alert.Percentage * b.Config.Amount / 100
			thresholdDate, thresholdValue := s.getForecastedDateByValue(forecastRows, &b.Config.StartPeriod, alertAmount, &b.Config.EndPeriod)
			alertPrediction := budget.BudgetForecastPrediction{
				Date:  thresholdDate,
				Value: thresholdValue,
			}
			predictions = append(predictions, &alertPrediction)
			b.Config.Alerts[i].ForecastedDate = thresholdDate
		} else {
			b.Config.Alerts[i].ForecastedDate = nil
		}
	}

	forecastedTotalAmountDate, _ := s.getForecastedDateByValue(forecastRows, &b.Config.StartPeriod, b.Config.Amount, &b.Config.EndPeriod)
	if forecastedTotalAmountDate != nil && s.shouldSendForecastAlert(b, forecastedTotalAmountDate) {
		b.Utilization.ShouldSendForecastAlert = true
		b.Utilization.PreviousForecastedDate = b.Utilization.ForecastedTotalAmountDate
	} else {
		b.Utilization.ShouldSendForecastAlert = false
	}

	b.Utilization.ForecastedTotalAmountDate = forecastedTotalAmountDate

	return predictions, nil
}

func (s *BudgetsService) shouldSendForecastAlert(b *budget.Budget, forecastedTotalAmountDate *time.Time) bool {
	if b == nil ||
		b.Config.TimeInterval == report.TimeIntervalDay ||
		b.Config.TimeInterval == report.TimeIntervalWeek ||
		b.Utilization.ForecastedTotalAmountDate == nil ||
		b.Utilization.Current > b.Config.Amount ||
		time.Since(*b.Utilization.ForecastedTotalAmountDate) > 0 ||
		time.Since(*forecastedTotalAmountDate) > 0 ||
		b.Config.EndPeriod.Sub(*forecastedTotalAmountDate) < 0 {
		return false
	}

	forecastedDateChangeDuration := b.Utilization.ForecastedTotalAmountDate.Sub(*forecastedTotalAmountDate)
	budgetDurationInSeconds := getBudgetDuration(b).Seconds()

	return b.IsValid && forecastedDateChangeDuration.Seconds() > budgetDurationInSeconds/10
}

func (s *BudgetsService) getForecastRowDateAsInt(row []bigquery.Value) (int, int, int, error) {
	rowYear, err := strconv.Atoi(row[1].(string))
	if err != nil {
		return 0, 0, 0, err
	}

	rowMonth, err := strconv.Atoi(row[2].(string))
	if err != nil {
		return 0, 0, 0, err
	}

	dateValue := strings.Split(row[3].(string), " ")[0]

	rowDay, err := strconv.Atoi(dateValue)
	if err != nil {
		return 0, 0, 0, err
	}

	return rowYear, rowMonth, rowDay, nil
}

// getForecastedDateByValue - returns the forecasted date on which the givven value will be reached
func (s *BudgetsService) getForecastedDateByValue(forecastRows [][]bigquery.Value, startDate *time.Time, value float64, endDate *time.Time) (*time.Time, float64) {
	// forecastevValue is initially set as the currentMonth accumulated amount minus budget start date accumulated amount
	forecastedValue := s.getDateValue(forecastRows, s.getLastDayOfMonth(startDate), true) - s.getDateValue(forecastRows, startDate, false)

	// if the forecasted value is higher than the target value we search for the target amount within this month
	if forecastedValue > value {
		forecastedValue = 0.0
		lastForecastDate := s.getLastForecastDate(forecastRows)
		d := startDate.AddDate(0, 0, 1)

		for forecastedValue < value && lastForecastDate != nil && d.Before(*lastForecastDate) {
			forecastedValue = s.getDateValue(forecastRows, &d, s.shouldIncludeDate(d)) - s.getDateValue(forecastRows, startDate, false)
			// when forecastedValue is higher than target value we return the date
			if forecastedValue > value {
				return &d, forecastedValue
			}

			d = d.AddDate(0, 0, 1)
		}
	}

	// if forecastedValue is lower the the target value we skip to the next month
	currentDate := startDate.AddDate(0, 1, 0)

	// when forecastedValue is higher than target value we search for the value in the current iteration month
	for forecastedValue < value {
		dateValue := s.getDateValue(forecastRows, s.getLastDayOfMonth(&currentDate), true)
		// this condition is a seurity condition to make sure we stop the loop when forecast data is done
		if dateValue == 0 {
			return nil, 0.0
		}

		forecastedValue += dateValue
		currentDate = currentDate.AddDate(0, 1, 0)
	}

	currentDate = currentDate.AddDate(0, -1, 0)

	// we set the forecasted value to the beginning of the current iteration month
	forecastedValue -= s.getDateValue(forecastRows, s.getLastDayOfMonth(&currentDate), true)

	d := *s.getFirstDayOfMonth(&currentDate)
	for i := 0; i < 32; i++ {
		d = d.AddDate(0, 0, 1)
		v := s.getDateValue(forecastRows, &d, false)

		if (forecastedValue + v) > value {
			return &d, forecastedValue + v
		}
	}

	forecastedValue += s.getDateValue(forecastRows, s.getLastDayOfMonth(&currentDate), true)
	if forecastedValue > value {
		return &d, forecastedValue
	}

	return nil, 0.0
}

func (s *BudgetsService) getForecastedValueByDates(forecastRows [][]bigquery.Value, startDate, endDate *time.Time) float64 {
	startYear, startMonth, _ := startDate.Date()
	endYear, endMonth, _ := endDate.Date()

	if startMonth == endMonth && startYear == endYear {
		return s.getDateValue(forecastRows, endDate, false) - s.getDateValue(forecastRows, startDate, false)
	}

	value := 0.0
	firstMonthValue := s.getDateValue(forecastRows, s.getLastDayOfMonth(startDate), true) - s.getDateValue(forecastRows, startDate, false)
	value += firstMonthValue
	currentDate := startDate.AddDate(0, 1, 0)

	for currentDate.Before(*endDate) {
		value += s.getDateValue(forecastRows, s.getLastDayOfMonth(&currentDate), true)
		currentDate = currentDate.AddDate(0, 1, 0)
	}

	lastMonthValue := s.getDateValue(forecastRows, endDate, s.shouldIncludeDate(*endDate)) - s.getDateValue(forecastRows, s.getFirstDayOfMonth(endDate), false)
	value += lastMonthValue

	return value
}

func (s *BudgetsService) shouldIncludeDate(date time.Time) bool {
	return date == *s.getLastDayOfMonth(&date)
}

// set includeDay = true when calling getDateValue for last day of month with an intention to include
// this day data in total utilization amount
func (s *BudgetsService) getDateValue(forecastRows [][]bigquery.Value, date *time.Time, includeDay bool) float64 {
	fromYear := date.Year()
	fromMonth := int(date.Month())

	fromDay := date.Day()
	if !includeDay {
		fromDay--
	}

	if fromDay == 0 {
		return 0.0
	}

	for _, row := range forecastRows {
		rowYear, rowMonth, rowDay, err := s.getForecastRowDateAsInt(row)
		if err != nil {
			return 0.0
		}

		if rowYear == fromYear && rowMonth == fromMonth && rowDay == fromDay {
			value, ok := row[4].(float64)
			if !ok {
				return 0.0
			}

			return value
		}
	}

	return 0.0
}

func (s *BudgetsService) getLastDayOfMonth(t *time.Time) *time.Time {
	lastDate := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	lastDate = lastDate.AddDate(0, 0, -1)

	return &lastDate
}

func (s *BudgetsService) getFirstDayOfMonth(t *time.Time) *time.Time {
	firstDate := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	return &firstDate
}

func (s *BudgetsService) getLastForecastDate(forecastRows [][]bigquery.Value) *time.Time {
	rowYear, rowMonth, rowDay, err := s.getForecastRowDateAsInt(forecastRows[len(forecastRows)-1])
	if err != nil {
		return nil
	}

	lastForecastDate := time.Date(rowYear, time.Month(rowMonth), rowDay, 0, 0, 0, 0, time.UTC)

	return &lastForecastDate
}

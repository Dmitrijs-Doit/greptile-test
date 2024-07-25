package forecasts

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/times"
	"github.com/doitintl/http"
)

var intervalToStepFrequency = map[string]string{
	"hour":      "H",
	"day":       "D",
	"week":      "W",
	"month":     "MS",
	"quarter":   "Q",
	"year":      "Y",
	"dayCumSum": "MONTH_CUM_SUM",
}

func (s *Service) GetForecastOriginAndResultRows(
	ctx context.Context,
	queryResultRows [][]bigquery.Value,
	queryRequestRows int,
	queryRequestCols []*domainQuery.QueryRequestX,
	interval string,
	metric int,
	maxRefreshTime, from, to time.Time,
) ([]*domain.OriginSeries, [][]bigquery.Value, error) {
	return s.getForecastOriginAndResultRows(ctx, queryResultRows, queryRequestRows, queryRequestCols, interval, metric, maxRefreshTime, from, to)
}

func (s *Service) makeForecastRequest(ctx context.Context, rawReq *domain.ForecastRequest) (*domain.ForecastResponse, error) {
	l := s.loggerProvider(ctx)

	var forecast domain.ForecastResponse

	resp, err := s.httpClient.Post(ctx, &http.Request{
		URL:          "/predict",
		Payload:      rawReq,
		ResponseType: &forecast,
	})
	if err != nil {
		l.Errorf("%+v", resp)
		return nil, fmt.Errorf("Forecast request failed with error: %s", err)
	}

	return &forecast, nil
}

func (s *Service) getForecastOriginAndResultRows(ctx context.Context, queryResultRows [][]bigquery.Value, queryRequestRows int, queryRequestCols []*domainQuery.QueryRequestX, interval string, metric int, maxRefreshTime, from, to time.Time) ([]*domain.OriginSeries, [][]bigquery.Value, error) {
	// week_day is currently not supported for forecasts
	if weekDayUsed(queryRequestCols) {
		return nil, nil, errors.New("unsupported col week_day used in forecasts")
	}

	forecastData := make([]*domain.OriginSeries, 0)

	for _, row := range queryResultRows {
		forecastDataPoint, err := formatForecastRow(row, queryRequestCols, queryRequestRows, metric)
		if err != nil {
			return nil, nil, err
		}

		if forecastDataPoint.IsBeforePeriod(interval, maxRefreshTime) {
			forecastData = append(forecastData, forecastDataPoint)
		}
	}

	var lastDateWithData *time.Time

	lastDateWithData, err := utils.GetLatestDateWithData(&from, &to, queryRequestRows, report.TimeInterval(interval), &queryResultRows)
	if err != nil {
		return nil, nil, err
	}

	forecastCutOff := getForecastCutOff(*lastDateWithData, to, report.TimeInterval(interval))

	forecastRows, err := s.getForecasts(ctx, forecastData, interval, from, to, forecastCutOff)
	if err != nil {
		return nil, nil, err
	}

	if err := checkMatchingDateFields(forecastRows, queryResultRows, len(queryRequestCols)); err != nil {
		return nil, nil, err
	}

	return forecastData, forecastRows, nil
}

// getForecastCutOff computes the forecast cutoff that corresponds to the time interval.
// Note that right now lastDateWithData is always aligned to the interval, so for the month of May 2023 with
// an interval set to month, the lastDayWithData will be 2023/05/01. The code will work even if this changes
// in the future.
// We check if 'to' is at the end of the time interval and if so the forecast starts
// at the beginning of the next time interval. Otherwise we're still inside of the current
// time interval and emit the forecast including it.
func getForecastCutOff(lastDateWithData time.Time, to time.Time, interval report.TimeInterval) time.Time {
	var l, t time.Time

	lYear, lMonth, lDay := lastDateWithData.Date()
	lastDayWithData := time.Date(lYear, lMonth, lDay, 0, 0, 0, 0, lastDateWithData.Location())

	// Force crossing an interval boundary if we're in the last day
	to = to.AddDate(0, 0, 1)
	tYear, tMonth, tDay := to.Date()

	switch interval {
	case report.TimeIntervalDay:
		// We don't have data until the end of the day.
		return lastDayWithData.AddDate(0, 0, 1)
	case report.TimeIntervalWeek:
		lDaysSinceLastMonday := times.DaysSinceLastMonday(lastDayWithData)
		tDaysSinceLastMonday := times.DaysSinceLastMonday(to)
		l = time.Date(lYear, lMonth, lDay, 0, 0, 0, 0, lastDateWithData.Location()).AddDate(0, 0, -lDaysSinceLastMonday)
		t = time.Date(tYear, tMonth, tDay, 0, 0, 0, 0, to.Location()).AddDate(0, 0, -tDaysSinceLastMonday)

		if t.YearDay() != l.YearDay() {
			return l.AddDate(0, 0, 7)
		}
	case report.TimeIntervalMonth:
		l = time.Date(lYear, lMonth, 1, 0, 0, 0, 0, lastDateWithData.Location())
		t = time.Date(tYear, tMonth, 1, 0, 0, 0, 0, to.Location())

		if t.YearDay() != l.YearDay() {
			return l.AddDate(0, 1, 0)
		}
	case report.TimeIntervalQuarter:
		l = time.Date(lYear, lMonth-(lMonth-1)%3, 1, 0, 0, 0, 0, lastDayWithData.Location())
		t = time.Date(tYear, tMonth-(tMonth-1)%3, 1, 0, 0, 0, 0, to.Location())

		if t.YearDay() != l.YearDay() {
			return l.AddDate(0, 3, 0)
		}

	case report.TimeIntervalYear:
		l = time.Date(lYear, 1, 1, 0, 0, 0, 0, lastDayWithData.Location())

		if lYear != tYear {
			return l.AddDate(1, 0, 0)
		}
	}

	return l
}

func weekDayUsed(cols []*domainQuery.QueryRequestX) bool {
	for _, col := range cols {
		if col.Key == "week_day" {
			return true
		}
	}

	return false
}

func checkMatchingDateFields(forecastRows [][]bigquery.Value, queryResultRows [][]bigquery.Value, lenReqCols int) error {
	if len(forecastRows) == 0 {
		return nil
	}

	firstForecastRow := forecastRows[0]
	firstDateFieldIndex := len(firstForecastRow) - 1 - lenReqCols
	firstForecastRowDateStrings := firstForecastRow[firstDateFieldIndex : len(firstForecastRow)-1]

	// ["forecast", ..."dateItems", "forecastValue"]
	firstForecastRowDates := make([]interface{}, len(firstForecastRowDateStrings))
	for i, fieldItem := range firstForecastRowDateStrings {
		firstForecastRowDates[i] = fieldItem
	}

	for _, resultRow := range queryResultRows {
		resultRowItems := make([]interface{}, len(resultRow))
		for i, rowItem := range resultRow {
			resultRowItems[i] = rowItem
		}
		// we want to find at least one resultRow that contains date fields of first forecast row
		if slice.SubSlice(firstForecastRowDates, resultRowItems) {
			return nil
		}
	}

	return fmt.Errorf("first forecast row date does not match any of query result row date")
}

func (s *Service) getForecasts(ctx context.Context, rawData []*domain.OriginSeries, interval string, from, to time.Time, forecastCutOff time.Time) ([][]bigquery.Value, error) {
	series := aggregateForecastRows(rawData)
	if len(series) < 8 {
		return nil, nil
	}

	var steps int

	maxForecast := getMaxForecastRange(interval, from, to)
	if len(series)/2 >= maxForecast {
		steps = maxForecast
	} else {
		if len(series) > maxForecast {
			steps = maxForecast
		} else {
			steps = len(series)
		}
	}

	payload := &domain.ForecastRequest{
		Series:        series,
		Steps:         steps,
		StepFrequency: intervalToStepFrequency[interval],
	}

	res, err := s.makeForecastRequest(ctx, payload)
	if err != nil {
		return nil, err
	}

	cleanForecasts := cleanForecastResponse(res, interval, from, forecastCutOff)

	rows, err := makePredictionRows(cleanForecasts, interval)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

// aggregateForecastRows creates unique array by dates (sums up all services for given date)
func aggregateForecastRows(unAggregated []*domain.OriginSeries) []*domain.OriginSeries {
	aggregatedSeries := make([]*domain.OriginSeries, 0)
	dsMap := make(map[string]float64)

	for _, dp := range unAggregated {
		if _, ok := dsMap[dp.DS]; ok {
			dsMap[dp.DS] += dp.Value
		} else {
			dsMap[dp.DS] = dp.Value
		}
	}

	for interval, value := range dsMap {
		dp := &domain.OriginSeries{
			DS:    interval,
			Value: value,
		}
		aggregatedSeries = append(aggregatedSeries, dp)
	}

	sort.Slice(aggregatedSeries, func(i, j int) bool {
		return aggregatedSeries[i].DS < aggregatedSeries[j].DS
	})

	return aggregatedSeries
}

// formatForecastRow transforms BiqQuery row to forecast data point.
func formatForecastRow(row []bigquery.Value, requestCols []*domainQuery.QueryRequestX, rows, metric int) (*domain.OriginSeries, error) {
	firstDateIndex := rows
	cols := len(requestCols)
	valueIndex := rows + cols + metric
	lastDateIndex := firstDateIndex + cols - 1
	colIndexToBaseValueMappingFuncMap := make(map[int]func(string) string)
	dsParts := make([]string, cols)

	for i, col := range requestCols {
		rowIndex := i + firstDateIndex

		mapperFunc := domainQuery.KeyMap[col.Key].BaseValueMappingFunc
		if mapperFunc != nil {
			colIndexToBaseValueMappingFuncMap[rowIndex] = mapperFunc
		}
	}

	dateValues := row[firstDateIndex : lastDateIndex+1]
	for i, datePart := range dateValues {
		keyStr, err := query.BigqueryValueToString(datePart)
		if err != nil {
			return nil, err
		}

		if mapperFunc, ok := colIndexToBaseValueMappingFuncMap[i+firstDateIndex]; ok {
			keyStr = mapperFunc(keyStr)
		}

		dsParts[i] = keyStr
	}

	ds := strings.Join(dsParts, "-")
	y := row[valueIndex].(float64)

	return &domain.OriginSeries{
		DS:    ds,
		Value: y,
	}, nil
}

// TODO: Get rid of the "Forecast" value, don't really need it for the reports
func makePredictionRows(rawResponse *domain.ForecastResponse, interval string) ([][]bigquery.Value, error) {
	rows := make([][]bigquery.Value, 0)

	for _, forecastRow := range rawResponse.Prediction {
		row := make([]bigquery.Value, 0)
		row = append(row, "Forecast")

		dateFields, err := dateFieldsToRow(forecastRow.DS, interval)
		if err != nil {
			return nil, err
		}

		for _, datePart := range dateFields {
			row = append(row, datePart)
		}

		if forecastRow.Ignore {
			row = append(row, nil)
		} else {
			row = append(row, forecastRow.Value)
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func dateFieldsToRow(dateStr string, interval string) ([]string, error) {
	result := make([]string, 0)
	dateParts := strings.Split(dateStr, "-")

	switch report.TimeInterval(interval) {
	case report.TimeIntervalWeek:
		if len(dateParts) != 2 {
			return nil, fmt.Errorf("unexpected dateParts parts %v returned from forecating for report.TimeIntervalWeek", dateParts)
		}

		yearInt, err := strconv.Atoi(dateParts[0])
		if err != nil {
			return nil, err
		}

		// we expect week part to W44 , we only want the number
		weekInt, err := strconv.Atoi(dateParts[1][1:])
		if err != nil {
			return nil, err
		}

		monday, err := times.WeekStart(yearInt, weekInt)
		if err != nil {
			return nil, err
		}

		// add year to row as is
		result = append(result, dateParts[0])
		// add week in format: W44 (Nov 01)
		formattedWeek := dateParts[1] + monday.Format(" (Jan 02)")
		result = append(result, formattedWeek)
	default:
		result = append(result, dateParts...)
	}

	return result, nil
}

// cleanForecastResponse converts the forecast api ds format to the format of the reports
// and removes rows that are before the report requested "from" time
func cleanForecastResponse(res *domain.ForecastResponse, interval string, from time.Time, forecastCutOff time.Time) *domain.ForecastResponse {
	len := len(res.Prediction)
	cutoffIndex := 0

	for i := len - 1; i >= 0; i-- {
		point := res.Prediction[i]

		switch report.TimeInterval(interval) {
		case report.TimeIntervalHour:
			// Example: YYYY-MM-DD HH:MM => YYYY-MM-DD-HH:00
			point.DS = fmt.Sprintf("%s-%s:00", point.DS[:10], point.DS[11:13])
		case report.TimeIntervalDay, report.TimeIntervalDayCumSum:
			// Example: YYYY-MM-DD HH:MM => YYYY-MM-DD
			point.DS = point.DS[:10]
		case report.TimeIntervalWeek:
			// Example: YYYY-W## => YYYY-W##
		case report.TimeIntervalMonth:
			// Example: YYYY-MM-DD HH:MM => YYYY-MM
			point.DS = point.DS[:7]
		case report.TimeIntervalQuarter:
			// Example: YYYY-Q# => YYYY-Q#
		case report.TimeIntervalYear:
			// Example: YYYY-MM-DD HH:MM => YYYY
			point.DS = point.DS[:4]
		default:
		}

		if point.IsBeforePeriod(interval, forecastCutOff) {
			res.Prediction[i].Ignore = true
		}

		// any additional points before this point are not relevant for the report
		if point.IsBeforePeriod(interval, from) {
			cutoffIndex = i + 1
			break
		}
	}

	res.Prediction = res.Prediction[cutoffIndex:]

	return res
}

// TODO: should not be part of the forecast package
// forecast do not care about trends and should not really be aware of them
func FilterByTrend(nonForecastRows [][]bigquery.Value, forecastRows [][]bigquery.Value, trends []report.Feature, trendIndex int, brows int) ([][]bigquery.Value, error) {
	forecastFilteredRows := make([][]bigquery.Value, 0)
	rowKeys := make(map[string]struct{})

	for _, row := range nonForecastRows {
		key, err := query.GetRowKey(row, brows)
		if err != nil {
			return nil, err
		}

		if _, ok := rowKeys[key]; ok {
			continue
		}

		for _, trend := range trends {
			if string(trend) == row[trendIndex] {
				rowKeys[key] = struct{}{}
				break
			}
		}
	}

	for _, frow := range forecastRows {
		frowKey, err := query.GetRowKey(frow, brows)
		if err != nil {
			return nil, err
		}

		if _, ok := rowKeys[frowKey]; ok {
			forecastFilteredRows = append(forecastFilteredRows, frow)
		}
	}

	return forecastFilteredRows, nil
}

func GetForecastStart(from time.Time, to time.Time, interval string) time.Time {
	diff := to.Sub(from)
	t := from.Add(-diff)

	switch report.TimeInterval(interval) {
	case report.TimeIntervalHour:
		// Get from the last full hour
		return t.Truncate(time.Minute * 60)
	case report.TimeIntervalDay:
		// Get from the last full day
		return t.Truncate(time.Hour * 24)
	case report.TimeIntervalWeek:
		// Get from the last full ISO week (starts on Monday)
		// weekDay Mon=0, Tue=1...Sun=6
		weekDay := int(t.Weekday()+6) % 7
		return t.Truncate(time.Hour*24).AddDate(0, 0, -weekDay)
	case report.TimeIntervalMonth:
		// Get from the last full month
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	case report.TimeIntervalQuarter:
		// Get from the last full quarter
		return time.Date(t.Year(), t.Month()-(t.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC)
	case report.TimeIntervalYear:
		// Get from the last full year
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	default:
		return t.Truncate(time.Hour * 24)
	}
}

func getMaxForecastRange(interval string, from time.Time, to time.Time) int {
	diff := to.Sub(from)
	hoursDiff := int(diff.Hours())

	switch report.TimeInterval(interval) {
	case report.TimeIntervalHour:
		return hoursDiff
	case report.TimeIntervalDay, report.TimeIntervalDayCumSum:
		return hoursDiff / 24
	case report.TimeIntervalWeek:
		return (hoursDiff / 24) / 7
	case report.TimeIntervalMonth:
		return (hoursDiff / 24) / 30
	case report.TimeIntervalQuarter:
		return (hoursDiff / 24) / 90
	case report.TimeIntervalYear:
		return (hoursDiff / 24) / 365
	default:
		return hoursDiff
	}
}

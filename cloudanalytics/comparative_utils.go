package cloudanalytics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type ComparativeColumnValue struct {
	Pct bigquery.Value `json:"pct" firestore:"pct"`
	Val bigquery.Value `json:"val" firestore:"val"`
}

// getComparativeDataWithClause returns a with clause named comparative_data. This clause will select all exsiting cols
// excpet timeseries_key and will add comaprative data cols for each metric.
func (qr *QueryRequest) getComparativeDataWithClause() (string, error) {
	comparativeCols, namedWindow := "", ""
	cdClause := fmt.Sprintf(`comparative_data AS (
	SELECT
		T.* EXCEPT (%s),
		{comparative_cols}
	FROM
		query_data AS T
	{named_window}
)`, QueryTimeseriesKey)

	if qr.Comparative != nil {
		ccols, err := qr.getComparativeSelectCols()
		if err != nil {
			return "", err
		}

		comparativeCols = ccols
		namedWindow = qr.getComparativeNamedWindow()
	}

	withComparativeCols := strings.NewReplacer(
		"{comparative_cols}",
		comparativeCols,
		"{named_window}",
		namedWindow,
	).Replace(cdClause)

	return withComparativeCols, nil
}

// getComparativeNamedWindow return a name window clause to use in the comprartive_data clause
func (qr *QueryRequest) getComparativeNamedWindow() string {
	var w string

	if len(qr.Rows) > 0 {
		r := make([]string, 0)

		for i := 0; i < len(qr.Rows); i++ {
			rowName := fmt.Sprintf("row_%s", fmt.Sprint(i))
			r = append(r, rowName)
		}

		w = fmt.Sprintf("PARTITION BY %s ORDER BY %s", strings.Join(r, consts.Comma), QueryTimeseriesKey)
	} else {
		w = fmt.Sprintf("ORDER BY %s", QueryTimeseriesKey)
	}

	return fmt.Sprintf("WINDOW w AS (%s)", w)
}

const (
	diffPctFormat   = "IFNULL(IF(%s, SAFE_DIVIDE(%s, LAG(%s) OVER(w)) - 1, IF(LAG(%s) OVER(w) IS NULL, 0, 1)) * 100, 0)"
	diffValFormat   = "IFNULL(IF(%s, %s - LAG(%s) OVER(w), IF(LAG(%s) OVER(w) IS NULL, 0, %s)), 0)"
	diffArrayFormat = "ARRAY<FLOAT64>[%s %s %s] AS diff_%s"
)

// getComparativeSelectCols will return the actual comparative data cols that calculate
// the actual values diff and percentage diff
func (qr *QueryRequest) getComparativeSelectCols() (string, error) {
	interval := string(qr.TimeSettings.Interval)
	comparativeAliases := make([]string, 0)
	d := fmt.Sprintf("DATETIME_SUB(%s, INTERVAL 1 %s) = LAG(%s) OVER(w)", QueryTimeseriesKey, interval, QueryTimeseriesKey)
	buildDiffArray := func(m string) string {
		diffPct := fmt.Sprintf(diffPctFormat, d, m, m, m)
		diffVal := fmt.Sprintf(diffValFormat, d, m, m, m, m)

		return fmt.Sprintf(diffArrayFormat, diffPct, commaFormat, diffVal, m)
	}

	// Cost, Usage and Savings metrics diffs
	for i := 0; i < int(report.MetricEnumLength); i++ {
		m, err := domainQuery.GetMetricString(report.Metric(i))
		if err != nil {
			return "", err
		}

		comparativeAliases = append(comparativeAliases, buildDiffArray(m))
	}

	// CSP Margin metric diff
	if qr.IsCSP {
		m, err := domainQuery.GetMetricString(report.MetricMargin)
		if err != nil {
			return "", err
		}

		comparativeAliases = append(comparativeAliases, buildDiffArray(m))
	}

	// Custom metric diff
	if qr.CalculatedMetric != nil {
		m, err := domainQuery.GetMetricString(report.MetricCustom)
		if err != nil {
			return "", err
		}

		comparativeAliases = append(comparativeAliases, buildDiffArray(m))
	}

	// Extended metric diff
	if qr.Metric == report.MetricExtended {
		m, err := domainQuery.GetMetricString(report.MetricExtended)
		if err != nil {
			return "", err
		}

		comparativeAliases = append(comparativeAliases, buildDiffArray(m))
	}

	return strings.Join(comparativeAliases, commaFormat), nil
}

func (qr *QueryRequest) getComparativeDatetimeTruncateSelect() string {
	keyInterval := qr.TimeSettings.Interval
	if qr.TimeSettings.Interval == report.TimeIntervalWeek {
		keyInterval = report.TimeIntervalISOWeek
	}

	return fmt.Sprintf("DATETIME_TRUNC(T.usage_date_time, %s) AS %s", keyInterval, QueryTimeseriesKey)
}

func (qr *QueryRequest) addMissingComparativeRows(rows *[][]bigquery.Value) (*[][]bigquery.Value, error) {
	latestDataDate, err := getLatestDateWithData(qr, rows)
	if err != nil {
		return nil, err
	}

	var result [][]bigquery.Value

	for i, row := range *rows {
		if i == 0 {
			result = append(result, row)
			continue
		}

		if missingRow, err := getMissingRow(qr, (*rows)[i-1], row, latestDataDate); err != nil {
			return nil, err
		} else if missingRow != nil {
			result = append(result, missingRow)
		}

		result = append(result, row)
	}

	if missingRow, err := getLastMissingRow(qr, (*rows)[len(*rows)-1], latestDataDate); err != nil {
		return nil, err
	} else if missingRow != nil {
		result = append(result, missingRow)
	}

	return &result, nil
}

func getLatestDateWithData(qr *QueryRequest, rows *[][]bigquery.Value) (*time.Time, error) {
	firstDateIdx, _ := getDateFirstLastIndex(len(qr.Rows), len(qr.Cols))
	return utils.GetLatestDateWithData(qr.TimeSettings.From, qr.TimeSettings.To, firstDateIdx, qr.TimeSettings.Interval, rows)
}

func getMissingRow(qr *QueryRequest, prevRow []bigquery.Value, row []bigquery.Value, latestDataDate *time.Time) ([]bigquery.Value, error) {
	lenRows := len(qr.Rows)
	timeSettings := qr.TimeSettings
	firstDateIdx, lastDateIdx := getDateFirstLastIndex(lenRows, len(qr.Cols))

	prevDate, err := utils.GetRowDate(prevRow, firstDateIdx, timeSettings.Interval)
	if err != nil {
		return nil, err
	}

	date, err := utils.GetRowDate(row, firstDateIdx, timeSettings.Interval)
	if err != nil {
		return nil, err
	}

	isSameRow := isSameRowRecords(prevRow, row, lenRows)
	nextDate := getNextDate(prevDate, timeSettings.Interval)

	if (isSameRow && nextDate.Before(*date)) ||
		(!isSameRow && *prevDate != *latestDataDate) {
		missingRow := getMissingDiffRow(prevRow, timeSettings.Interval, qr.GetMetricCount(), firstDateIdx, lastDateIdx)
		if missingRow != nil {
			return missingRow, nil
		}
	}

	return nil, nil
}

func getLastMissingRow(qr *QueryRequest, lastRow []bigquery.Value, latestDataDate *time.Time) ([]bigquery.Value, error) {
	firstDateIdx, lastDateIdx := getDateFirstLastIndex(len(qr.Rows), len(qr.Cols))

	lastRowDate, err := utils.GetRowDate(lastRow, firstDateIdx, qr.TimeSettings.Interval)
	if err != nil {
		return nil, err
	}

	if *lastRowDate != *latestDataDate {
		if missingRow := getMissingDiffRow(lastRow, qr.TimeSettings.Interval, qr.GetMetricCount(),
			firstDateIdx, lastDateIdx); missingRow != nil {
			return missingRow, nil
		}
	}

	return nil, nil
}

func getNegativeComparative(value bigquery.Value) ComparativeColumnValue {
	if val, ok := value.(float64); ok {
		return ComparativeColumnValue{Pct: -100, Val: -1 * val}
	}

	return ComparativeColumnValue{}
}

/*
*
gets next missing record with diff vals
*/
func getMissingDiffRow(prevRow []bigquery.Value, interval report.TimeInterval, metricCount, firstDateIdx, lastDateIdx int) []bigquery.Value {
	var diffRow []bigquery.Value
	diffRow = append(diffRow, prevRow...)
	diffRow = getNextDateRow(diffRow, interval, firstDateIdx)

	if diffRow != nil {
		for i := 1; i <= metricCount; i++ {
			diffRow[lastDateIdx+i] = 0.0
			diffRow[lastDateIdx+metricCount+i] = getNegativeComparative(prevRow[lastDateIdx+i])
		}

		diffRow = append(diffRow, "no metric value")

		return diffRow
	}

	return nil
}

/*
*
check if 2 records are frlom the same row - have same row labels
*/
func isSameRowRecords(prevRow []bigquery.Value, row []bigquery.Value, lenRows int) bool {
	for i := 0; i < lenRows; i++ {
		if prevRow[i] != row[i] {
			return false
		}
	}

	return true
}

/*
*
get next date of next record by interval
*/
func getNextDate(prevDate *time.Time, interval report.TimeInterval) time.Time {
	var nextDate time.Time

	switch interval {
	case report.TimeIntervalHour:
		nextDate = prevDate.Add(time.Hour)
	case report.TimeIntervalDay:
		nextDate = prevDate.AddDate(0, 0, 1)
	case report.TimeIntervalWeek:
		nextDate = *prevDate
		for i := 1; i <= 7; i++ {
			nextDate = nextDate.AddDate(0, 0, 1)
			if nextDate.Weekday() == time.Monday {
				break
			}
		}
	case report.TimeIntervalMonth:
		nextDate = prevDate.AddDate(0, 1, 0)
	case report.TimeIntervalQuarter:
		nextDate = prevDate.AddDate(0, 3, 0)
	case report.TimeIntervalYear:
		nextDate = prevDate.AddDate(1, 0, 0)
	default:
	}

	return nextDate
}

func updateRowFromDate(row []bigquery.Value, nextDate time.Time, firstDateIdx int, numCols int, withTime bool) []bigquery.Value {
	dateStr := strings.Split(nextDate.Format("2006-01-02"), "-")
	for i := 0; i < numCols; i++ {
		row[firstDateIdx+i] = dateStr[i]
	}

	if withTime {
		timeParts := strings.Split(nextDate.Format("03:04:00"), ":")
		if len(timeParts) != 2 {
			return nil
		}

		row[firstDateIdx+numCols] = fmt.Sprintf("%s:%s", timeParts[0], timeParts[1])
	}

	return row
}

/*
*
finds the next date for record by interval and updates next new row with it
*/
func getNextDateRow(row []bigquery.Value, interval report.TimeInterval, firstDateIdx int) []bigquery.Value {
	prevRowdate, err := utils.GetRowDate(row, firstDateIdx, interval)
	if err != nil {
		return nil
	}

	nextDate := getNextDate(prevRowdate, interval)

	switch interval {
	case report.TimeIntervalYear, report.TimeIntervalMonth, report.TimeIntervalDay, report.TimeIntervalHour:
		numCols, withTime := utils.GetIntervalNumCols(interval)
		row = updateRowFromDate(row, nextDate, firstDateIdx, numCols, withTime)

	case report.TimeIntervalWeek:
		year, week := nextDate.ISOWeek()
		row[firstDateIdx] = strconv.Itoa(year)
		dayWeek := nextDate.Format(" (Jan 02)")
		row[firstDateIdx+1] = fmt.Sprintf("W%02d%s", week, dayWeek) // add week in format: W44 (Nov 01)

	case report.TimeIntervalQuarter:
		nextQuarterInt := (int(nextDate.Month()) / 3) + 1
		row[firstDateIdx] = nextDate.Year()
		row[firstDateIdx+1] = fmt.Sprintf("Q%d", nextQuarterInt)
	default:
	}

	return row
}

func getDateFirstLastIndex(lenRows int, lenCols int) (int, int) {
	return lenRows, lenCols + lenRows - 1
}

package service

import (
	"regexp"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func (s *ExternalAPIService) ProcessResult(qr *cloudanalytics.QueryRequest, r *domainReport.Report, result cloudanalytics.QueryResult) domainExternalAPI.RunReportResult {
	var runReportResult domainExternalAPI.RunReportResult

	numRowsCols := len(qr.Rows) + len(qr.Cols)
	metricIndex := qr.GetMetricIndex()
	metricCount := qr.GetMetricCount()

	// For metric cost,
	// ["row_0", "col_0", "col_1", cost, usage, savings, trend_cost, trend_usage, trend_savings]
	// numRowsCols = 3, metricIndex = 0, metricCount = 3 => trendIndex = 6
	trendIndex := numRowsCols + metricIndex + metricCount

	isIncreasing := false
	isDecreasing := false
	isForcast := false

	for _, trend := range r.Config.Features {
		switch trend {
		case "increasing":
			isIncreasing = true
		case "decreasing":
			isDecreasing = true
		case "forecast":
			isForcast = true
		}
	}

	// loop on all rows to filter values and filter rows by the ml features
	filteredRows := make([][]bigquery.Value, 0)

	for _, row := range result.Rows {
		if (isIncreasing && row[trendIndex] == "increasing") ||
			(isDecreasing && row[trendIndex] == "decreasing") ||
			(isForcast && row[trendIndex] == "none") ||
			(!isDecreasing && !isIncreasing) {
			newRow := row[0:numRowsCols]
			newRow = append(newRow, row[numRowsCols+metricIndex])
			filteredRows = append(filteredRows, newRow)
		}
	}

	// add rows to the schema
	for _, row := range qr.Rows {
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: row.Key, Type: "string"})
	}

	// add cols to the schema
	for _, col := range qr.Cols {
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: col.Key, Type: "string"})
	}

	// add metric to schema
	switch r.Config.Metric {
	case domainReport.MetricCost:
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: "cost", Type: "float"})
	case domainReport.MetricUsage:
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: "usage", Type: "float"})
	case domainReport.MetricSavings:
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: "saving", Type: "float"})
	case domainReport.MetricCustom:
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: qr.CalculatedMetric.Name, Type: "float"})
	case domainReport.MetricExtended:
		runReportResult.Schema = append(runReportResult.Schema, &domainExternalAPI.SchemaField{Name: r.Config.ExtendedMetric, Type: "float"})
	}

	runReportResult.Rows = filteredRows
	runReportResult.MlFeatures = r.Config.Features
	runReportResult.ForecastRows = result.ForecastRows

	addTimeStampToResult(&runReportResult)

	return runReportResult
}

func addTimeStampToResult(result *domainExternalAPI.RunReportResult) {
	// find the year, month, day, hour, week column index in schema, so we could calc the epoc time
	dateTimeIndexRows := domainExternalAPI.DateTimeIndex{
		Year:  -1,
		Month: -1,
		Day:   -1,
		Hour:  -1,
		Week:  -1,
	}

	for i, col := range result.Schema {
		if col.Name == "year" {
			dateTimeIndexRows.Year = i
		}

		if col.Name == "month" {
			dateTimeIndexRows.Month = i
		}

		if col.Name == "day" {
			dateTimeIndexRows.Day = i
		}

		if col.Name == "hour" {
			dateTimeIndexRows.Hour = i
		}

		if col.Name == "week" {
			dateTimeIndexRows.Week = i
		}
	}

	dateTimeIndexForecast := domainExternalAPI.DateTimeIndex{
		Year:  1,
		Month: -1,
		Day:   -1,
		Hour:  -1,
		Week:  -1,
	}

	if dateTimeIndexRows.Month != -1 {
		dateTimeIndexForecast.Month = 2
	} else if dateTimeIndexRows.Week != -1 {
		dateTimeIndexForecast.Week = 2
	}

	if dateTimeIndexRows.Day != -1 {
		dateTimeIndexForecast.Day = 3
	}

	if dateTimeIndexRows.Hour != -1 {
		dateTimeIndexForecast.Hour = 4
	}

	result.Schema = append(result.Schema, &domainExternalAPI.SchemaField{Name: "timestamp", Type: "timestamp"})
	addTimeStampToRows(&result.Rows, dateTimeIndexRows)
	addTimeStampToRows(&result.ForecastRows, dateTimeIndexForecast)
}

func addTimeStampToRows(rows *[][]bigquery.Value, dateTimeIndex domainExternalAPI.DateTimeIndex) {
	for i, row := range *rows {
		unixTime := getUnixTime(dateTimeIndex, row)
		newRow := append(row, unixTime)
		(*rows)[i] = newRow
	}
}

func getUnixTime(dateTimeIndex domainExternalAPI.DateTimeIndex, row []bigquery.Value) int64 {
	var date time.Time

	year := 1970
	if dateTimeIndex.Year != -1 {
		year = getIntFromBigQueryValue(row, dateTimeIndex.Year)
	}

	if dateTimeIndex.Week != -1 {
		week := getNumberOfWeekFromBigQueryValue(row, dateTimeIndex.Week)
		date = getDateOfWeek(year, week)
	} else {
		month := time.Month(getIntFromBigQueryValue(row, dateTimeIndex.Month))
		day := getIntFromBigQueryValue(row, dateTimeIndex.Day)
		hour := getHourFromBigQueryValue(row, dateTimeIndex.Hour)
		date = time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	}

	return date.Unix()
}

func getDateOfWeek(year int, week int) time.Time {
	startDayOfYear := int(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Weekday())

	var offsetDays = (8 - startDayOfYear) % 7

	var d = 1 + (week-1)*7 + offsetDays // 1st of January + 7 days for each week + offeset to Monday

	return time.Date(year, 1, d, 0, 0, 0, 0, time.UTC)
}

func getIntFromBigQueryValue(values []bigquery.Value, index int) int {
	if index == -1 {
		return 1
	}

	parsedStr, ok := values[index].(string)
	if ok {
		res, err := strconv.Atoi(parsedStr)
		if err != nil {
			return 1
		}

		return res
	} else {
		return 1
	}
}

var (
	numberOfWeekRE = regexp.MustCompile(`W(\d*) `)
	hourRE         = regexp.MustCompile(`(\d*):\d*`)
)

func getNumberOfWeekFromBigQueryValue(values []bigquery.Value, index int) int {
	parsedStr, ok := values[index].(string)
	if ok {
		result := numberOfWeekRE.FindStringSubmatch(parsedStr)

		res, err := strconv.Atoi(result[1])

		if err != nil {
			return 1
		}

		return res
	}

	return 1
}

func getHourFromBigQueryValue(values []bigquery.Value, index int) int {
	if index == -1 {
		return 0
	}

	parsedStr, ok := values[index].(string)
	if ok {
		result := hourRE.FindStringSubmatch(parsedStr)

		var resStr = parsedStr
		if result != nil {
			resStr = result[1]
		}

		res, err := strconv.Atoi(resStr)

		if err != nil {
			return 0
		}

		return res
	}

	return 0
}

//go:build integration
// +build integration

package trend

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/iterator"

	queryPkg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
)

func query(ctx context.Context, client *bigquery.Client, queryString string) (*bigquery.RowIterator, error) {

	query := client.Query(queryString)
	return query.Read(ctx)
}

func getRowsArr(queryString string) [][]bigquery.Value {

	projectID := "doitintl-cmp-dev"

	ctx := context.Background()

	client, err := bigquery.NewClient(ctx, projectID)
	defer client.Close()

	iter, err := query(ctx, client, queryString)
	if err != nil {
		fmt.Println(err)
	}

	rowsArr := make([][]bigquery.Value, 0)
	for {
		var row []bigquery.Value
		err := iter.Next(&row)
		if err != nil {
			if err == iterator.Done {
				break
			}
		}
		rowsArr = append(rowsArr, row)
	}

	return rowsArr
}

func printTimeSeries(values []float64) {
	// Get min and max values
	min, max := sliceMinMax(values)

	// Get floor and ceil of min and max
	Min := math.Floor(min)
	Max := int(math.Ceil(max))
	divider := 1
	if Max > 1000 {
		divider = 1000
	} else if Max > 100 {
		divider = 100
	}

	if min > 0.0 {

		for i := 0; i <= 2*Max/divider; i++ {
			j := float64((2*Max/divider - i)) / 2.0
			layer := make([]string, 0)
			for _, r := range values {
				if r/float64(divider) >= float64(j) {
					layer = append(layer, "|")
				} else {
					layer = append(layer, ".")
				}
			}
			if j > Min/float64(divider) {
				fmt.Println(layer, j*float64(divider))
			}
		}
	} else if min == max {
		layer := make([]string, 0)
		for _, r := range values {
			if r >= float64(min) {
				layer = append(layer, "|")
			} else {
				layer = append(layer, ".")
			}
		}
		fmt.Println(layer, min)
	} else {
		fmt.Println("Can't show chart. Values: ", values)
	}
}

func TestSlope(t *testing.T) {
	a := []float64{0.0, 10.0, 20.0, 30.0}
	slope := getSlopeDegree(a)
	assert.Equal(t, math.Round(slope*100)/100, 84.29)
}

func TestTrendsSpecificQuery(t *testing.T) {
	queryString := `WITH
		raw_data AS (
			SELECT * FROM doitintl-cmp-gcp-data.gcp_billing_01A29B_B56F30_AA7597.doitintl_billing_export_v1_01A29B_B56F30_AA7597
		),
		filtered_data AS (
			SELECT
				T.project_id AS row_0,
				FORMAT_DATETIME("%Y", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_0,
				FORMAT_DATETIME("%m", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_1,
				FORMAT_DATETIME("%d", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_2,
				report_value,
			FROM
				raw_data AS T
			LEFT JOIN
				UNNEST(report) AS report_value
			WHERE
				DATE(T.usage_start_time, "America/Los_Angeles") BETWEEN DATE("2020-08-01") AND DATE("2020-08-31")
				AND DATE(T.export_time) >= DATE("2020-08-01")
				AND DATE(T.export_time) <= DATE("2020-09-05")
		),
		query_data AS (
			SELECT
				row_0,
				col_0,
				col_1,
				col_2,
				SUM(report_value.cost) AS cost,
				SUM(report_value.usage) AS usage
			FROM
				filtered_data AS T
			GROUP BY 1, 2, 3, 4
		)
		SELECT
			S.*
		FROM
			query_data AS S
		WHERE row_0='doitintl-cmp-anomaly-dev' OR row_0='garrett-00001' OR row_0='doitintl-cmp-gcp-data'
		ORDER BY 1, 2, 3, 4;`

	// Get query result
	rowsArr := getRowsArr(queryString)

	rows, cols := 1, 3
	metricsCount := len(rowsArr[0]) - rows - cols
	costTrendIndex := len(rowsArr[0])

	filteredRows, err := Detection(rows, cols, rowsArr, "day")
	if err != nil {
		t.Errorf("Error")
	}

	for m := 0; m < metricsCount; m++ {
		timeSeries := make(map[bigquery.Value][]float64, 0)
		timeSeriesTrends := make(map[string]string, 0)
		for _, r := range filteredRows {
			key := queryPkg.GetRowKey(r, rows)
			timeSeries[key] = append(timeSeries[key], r[rows+cols+m].(float64))
			timeSeriesTrends[string(key)] = r[costTrendIndex+m].(string)
		}

		fmt.Println("")
		fmt.Println("Metric ID: ", m)
		fmt.Println(timeSeriesTrends)
		/*if m == 0 {
			assert.Equal(t, timeSeriesTrends, map[string]string{"doitintl-cmp-anomaly-dev": "decreasing", "doitintl-cmp-gcp-data": "increasing", "garrett-00001": "none"})
		} else if m == 1 {
			assert.Equal(t, timeSeriesTrends, map[string]string{"doitintl-cmp-anomaly-dev": "none", "doitintl-cmp-gcp-data": "increasing", "garrett-00001": "none"})
		}*/

		for m, n := range timeSeriesTrends {
			fmt.Println("")
			fmt.Println("Key: ", m, "Trend: ", n)
			printTimeSeries(timeSeries[m])
		}
	}

	assert.Equal(t, err, nil)
}

func TestTrendsAnyQuery1(t *testing.T) {
	queryString := `
			WITH conversion_rates AS (
				SELECT * FROM doitintl-cmp-gcp-data.gcp_billing.gcp_currencies_v1
				WHERE currency = "USD"
			),
		raw_data AS (
			SELECT * FROM doitintl-cmp-gcp-data.gcp_billing_01DFC3_847858_A76D77.doitintl_billing_export_v1_01DFC3_847858_A76D77
		),
		filtered_data AS (
			SELECT
				T.project_id AS row_0,
				T.service_description AS row_1,
				FORMAT_DATETIME("%Y", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_0,
				FORMAT_DATETIME("%m", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_1,
				FORMAT_DATETIME("%d", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_2,
				report_value,
				IFNULL(C.currency_conversion_rate, 1) AS currency_conversion_rate
			FROM
				raw_data AS T
			LEFT JOIN
				conversion_rates AS C
			ON
				C.invoice_month = DATE_TRUNC(DATE(T.usage_start_time, "America/Los_Angeles"), MONTH)
			LEFT JOIN
				UNNEST(report) AS report_value
			WHERE
				DATE(T.usage_start_time, "America/Los_Angeles") BETWEEN DATE("2020-09-01") AND DATE("2020-09-30")
				AND DATE(T.export_time) >= DATE("2020-09-01")
				AND DATE(T.export_time) <= DATE("2020-10-05")
		),
		query_data AS (
			SELECT
				row_0,
				row_1,
				col_0,
				col_1,
				col_2,
				SUM(report_value.cost * currency_conversion_rate) AS cost,
				SUM(report_value.usage) AS usage,
				SUM(report_value.savings * currency_conversion_rate) AS savings
			FROM
				filtered_data AS T
			GROUP BY 1, 2, 3, 4, 5
		)
		SELECT
			S.*
		FROM
			query_data AS S
		ORDER BY 1, 2, 3, 4, 5
	`

	// Get query result
	rowsArr := getRowsArr(queryString)

	printResults := false
	rows, cols := 2, 3
	metricsCount := len(rowsArr[0]) - rows - cols
	costTrendIndex := len(rowsArr[0])

	s := time.Now()
	filteredRows, err := Detection(rows, cols, rowsArr, "day")
	fmt.Println("Time needed for trend detection with 3 metrics and ", len(rowsArr), " rows: ", time.Since(s).Seconds(), " sec")
	if err != nil {
		t.Errorf("Error")
	}

	for m := 0; m < metricsCount; m++ {
		// timeSeries := make(map[bigquery.Value][]float64, 0)
		timeSeriesTrends := make(map[string]string, 0)
		for _, r := range filteredRows {
			key := queryPkg.GetRowKey(r, rows)
			// timeSeries[key] = append(timeSeries[key], r[rows+cols+m].(float64))
			timeSeriesTrends[string(key)] = r[costTrendIndex+m].(string)
		}

		fmt.Println("")
		fmt.Println("Metric ID: ", m)

		c := 0
		inc, dec, non := 0, 0, 0
		for m, n := range timeSeriesTrends {
			c += 1
			if n == "increasing" {
				inc += 1
			} else if n == "decreasing" {
				dec += 1
			} else {
				non += 1
			}

			if printResults {
				fmt.Println("Key: ", m, "Trend: ", n)
				//printTimeSeries(timeSeries[m])
			}
		}
		fmt.Println("Number of time series: ", c, " increasing: ", inc, " decreasing: ", dec, " none: ", non)
	}

	assert.Equal(t, err, nil)
}

func TestTrendsAnyQuery2(t *testing.T) {
	queryString := `WITH
		raw_data AS (
			SELECT * FROM doitintl-cmp-gcp-data.gcp_billing_01A29B_B56F30_AA7597.doitintl_billing_export_v1_01A29B_B56F30_AA7597
		),
		filtered_data AS (
			SELECT
				T.project_id AS row_0,
				FORMAT_DATETIME("%Y", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_0,
				FORMAT_DATETIME("%m", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_1,
				FORMAT_DATETIME("%d", DATETIME_TRUNC(DATETIME(T.usage_start_time, "America/Los_Angeles"), DAY)) AS col_2,
				report_value,
			FROM
				raw_data AS T
			LEFT JOIN
				UNNEST(report) AS report_value
			WHERE
				DATE(T.usage_start_time, "America/Los_Angeles") BETWEEN DATE("2020-08-01") AND DATE("2020-08-31")
				AND DATE(T.export_time) >= DATE("2020-08-01")
				AND DATE(T.export_time) <= DATE("2020-09-05")
		),
		query_data AS (
			SELECT
				row_0,
				col_0,
				col_1,
				col_2,
				SUM(report_value.cost) AS cost,
				SUM(report_value.usage) AS usage
			FROM
				filtered_data AS T
			GROUP BY 1, 2, 3, 4
		)
		SELECT
			S.*
		FROM
			query_data AS S
		ORDER BY 1, 2, 3, 4;`

	// Get query result
	rowsArr := getRowsArr(queryString)

	printResults := false
	rows, cols := 1, 3
	metricsCount := len(rowsArr[0]) - rows - cols
	costTrendIndex := len(rowsArr[0])

	s := time.Now()
	filteredRows, err := Detection(rows, cols, rowsArr, "day")
	fmt.Println("")
	fmt.Println("Time needed for trend detection with 3 metrics and ", len(rowsArr), " rows: ", time.Since(s).Seconds())
	if err != nil {
		t.Errorf("Error")
	}

	for m := 0; m < metricsCount; m++ {
		// timeSeries := make(map[bigquery.Value][]float64, 0)
		timeSeriesTrends := make(map[string]string, 0)
		for _, r := range filteredRows {
			key := queryPkg.GetRowKey(r, rows)
			// timeSeries[key] = append(timeSeries[key], r[rows+cols+m].(float64))
			timeSeriesTrends[string(key)] = r[costTrendIndex+m].(string)
		}

		fmt.Println("")
		fmt.Println("Metric ID: ", m)

		c := 0
		inc, dec, non := 0, 0, 0
		for m, n := range timeSeriesTrends {
			c += 1
			if n == "increasing" {
				inc += 1
			} else if n == "decreasing" {
				dec += 1
			} else {
				non += 1
			}

			if printResults {
				fmt.Println("Key: ", m, "Trend: ", n)
				//printTimeSeries(timeSeries[m])
			}
		}
		fmt.Println("Number of time series: ", c, " increasing: ", inc, " decreasing: ", dec, " none: ", non)
	}

	assert.Equal(t, err, nil)
}

/*
func TestTrendIncreasing(t *testing.T) {
	// Get query result
	rowsArr := getRowsArr()

	filteredRows, err := filterTrending(1,3,rowsArr)
	if err != nil {
		t.Errorf("Error")
	}

	correctRows := make([][]bigquery.Value, 0)
	correctRowsValues := make([]float64, 0)
	for _, r := range rowsArr {
		if r[0] == "doitintl-cmp-gcp-data" {
			correctRows = append(correctRows,r)
			correctRowsValues = append(correctRowsValues, r[4].(float64))
		}
	}

	fmt.Println("")
	fmt.Println("Trend: increasing, key: doitintl-cmp-gcp-data")
	printTimeSeries(correctRowsValues)
	assert.Equal(t, filteredRows, correctRows)
	assert.Equal(t, err, nil)
}

func TestTrendDecreasing(t *testing.T) {
	// Get query result
	rowsArr := getRowsArr()

	filteredRows, err := filterTrending(1, 3, rowsArr)
	if err != nil {
		t.Errorf("Error")
	}

	correctRows := make([][]bigquery.Value, 0)
	correctRowsValues := make([]float64, 0)
	for _, r := range rowsArr {
		if r[0] == "doitintl-cmp-anomaly-dev" {
			correctRows = append(correctRows,r)
			correctRowsValues = append(correctRowsValues, r[4].(float64))
		}
	}

	fmt.Println("")
	fmt.Println("Trend: decreasing, key: doitintl-cmp-anomaly-dev")
	printTimeSeries(correctRowsValues)
	assert.Equal(t, filteredRows, correctRows)
	assert.Equal(t, err, nil)
}

func TestNoTrend(t *testing.T) {
	// Get query result
	rowsArr := getRowsArr()

	mapping := getTrendIndicatorMapping(1, 3, 0,rowsArr)

	// ad-monitoring-287512: no trend
	// doitintl-cmp-gcp-data: increasing
	// doitintl-cmp-anomaly-dev: decreasing

	correctRowsValues := make([]float64, 0)
	for _, r := range rowsArr {
		if r[0] == "garrett-00001" {
			correctRowsValues = append(correctRowsValues, r[4].(float64))
		}
	}

	fmt.Println("")
	fmt.Println("Trend: not trending, key: garrett-00001")
	printTimeSeries(correctRowsValues)
	assert.Equal(t, mapping["garrett-00001"], "")
}

func TestTrendAll(t *testing.T) {
	// Get query result
	rowsArr := getRowsArr()

	filteredRows, err := filterTrending(1, 3, rowsArr)
	if err != nil {
		t.Errorf("Error")
	}

	correctRows := make([][]bigquery.Value, 0)
	for _, r := range rowsArr {
		if r[0] == "doitintl-cmp-anomaly-dev" ||  r[0] == "doitintl-cmp-gcp-data" {
			correctRows = append(correctRows,r)
		}
	}

	assert.Equal(t, filteredRows, correctRows)
	assert.Equal(t, err, nil)
  }
*/

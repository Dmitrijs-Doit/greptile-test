package trend

import (
	"context"
	"math"
	"sync"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/chewxy/stl"
	"gonum.org/v1/gonum/stat"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// parallelization job data
type job struct {
	key        string
	timeSeries []float64
}

const (
	alpha            = 0.05
	ppf              = 1.959963984540054 // norm.ppf(1-alpha/2)
	defaultSlope     = 22.5
	defaultThreshold = 0.1
)

var slopeThreshold float64 // slope threshold to classify a time series as 'not interesting' according to the slope coming from linear regression. Unit: degrees. Example: 22.5
var threshold float64      // values threshold to classify a time series as 'not interesting' showing the difference between max and min value. Example: 0.1 means 10%

func init() {
	ctx := context.Background()

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return
	}

	// Get trends parameters from Firestore
	go func() {
		type config struct {
			Trend struct {
				Slope     float64 `firestore:"slope"`
				Threshold float64 `firestore:"threshold"`
			} `firestore:"trend"`
		}

		// Listen to real time update on config values
		iter := fs.Collection("app").Doc("cloud-analytics").Snapshots(ctx)
		defer iter.Stop()

		for {
			docSnap, err := iter.Next()
			if err != nil {
				slopeThreshold = defaultSlope
				threshold = defaultThreshold

				return
			}

			var c config
			if err := docSnap.DataTo(&c); err != nil {
				slopeThreshold = defaultSlope
				threshold = defaultThreshold

				return
			}

			slopeThreshold = c.Trend.Slope
			threshold = c.Trend.Threshold
		}
	}()
}

func sign(x float64) float64 {
	// Get sign of a float (-1 if negative, 0 if 0 and 1 if positive)
	if x < 0 {
		return -1
	} else if x == 0 {
		return 0
	} else {
		return 1
	}
}

func getValuesFromMap(mapping map[float64]int) []int {
	// Get values from map
	v := make([]int, 0, len(mapping))
	for _, value := range mapping {
		v = append(v, value)
	}

	return v
}

func countUniqueValues(arr []float64) map[float64]int {
	//Create a counter for values in an array
	dict := make(map[float64]int)
	for _, num := range arr {
		dict[num] = dict[num] + 1
	}

	return dict
}

func MKtest(arr []float64, a float64) string {
	// Pseudocode see https://up-rs-esp.github.io/mkt/
	// Get array length
	n := len(arr)

	// Calculate S which is the number of positive differences minus the number of negative differences
	// If S is a positive number, observations obtained later in time tend to be larger than observations made earlier.
	// If S is a negative number, then observations made later in time tend to be smaller than observations made earlier.
	s := 0.0

	for k := 0; k < n-1; k++ {
		for j := k + 1; j < n; j++ {
			s += sign(arr[j] - arr[k])
		}
	}

	// Calculate unique observations
	uniq := countUniqueValues(arr)
	g := len(uniq)
	varS, z := 0.0, 0.0

	// Calculate var(s)
	if g == n {
		// No tie case
		varS = float64((n * (n - 1) * (2*n + 5))) / 18
	} else {
		//  Ties in data case
		tp := getValuesFromMap(uniq)
		sumExpression := 0

		for i := range tp {
			sumExpression += tp[i] * (tp[i] - 1) * (2*tp[i] + 5)
		}

		varS = float64((n*(n-1)*(2*n+5) + sumExpression)) / 18
	}

	// Compute the Mann-Kendall test statistic Z
	if s > 0 {
		z = float64(s-1) / math.Sqrt(float64(varS))
	} else if s == 0 {
		z = 0
	} else if s < 0 {
		z = float64(s+1) / math.Sqrt(float64(varS))
	}

	// Calculate the h value
	h := math.Abs(z) > ppf

	// Decide if we see trend in data (default: "none" which means no trend)
	trend := "none"
	if z < 0 && h == true {
		trend = "decreasing"
	} else if z > 0 && h == true {
		trend = "increasing"
	}

	return trend
}

func getSlopeDegree(timeSeries []float64) float64 {
	// Get X and Y arrays for linear regression as well as weights
	xs, ys, weights := make([]float64, 0), make([]float64, 0), make([]float64, 0)
	for i, v := range timeSeries {
		xs = append(xs, float64(i))
		ys = append(ys, float64(v))
		weights = append(weights, float64(1))
	}

	// Do not force the regression line to pass through the origin.
	origin := false

	// Compute and return the slope
	_, slope := stat.LinearRegression(xs, ys, weights, origin)

	deg := math.Atan(slope) * (180.0 / math.Pi)

	return deg
}

func sliceMinMax(array []float64) (float64, float64) {
	// Get min and max values of an array
	var max float64 = array[0.0]

	var min float64 = array[0.0]

	for _, value := range array {
		if max < value {
			max = value
		}

		if min > value {
			min = value
		}
	}

	return math.Floor(min*10) / 10, math.Floor(max*10) / 10
}

func getPeriodicity(interval string, arr []float64) int {
	arrLen := len(arr)
	periodicity := arrLen

	switch interval {
	case "hour":
		periodicity = 24
	case "day":
		periodicity = 7
	case "week":
		periodicity = 52
	case "month":
		periodicity = 12
	case "quarter":
		periodicity = 4
		// case "year":
		//	periodicity = arrLen
	}

	// Periodicity cannot be greater than length of arr
	if arrLen < periodicity {
		periodicity = arrLen
	}

	return periodicity
}

func hasNanValues(arr []float64) bool {
	hasNan := false

	for _, m := range arr {
		if math.IsNaN(m) {
			hasNan = true
		}
	}

	return hasNan
}

func getTrendIndicator(arr []float64, interval string) string {
	// Expect at least 3 data points to detect any trends
	if len(arr) < 3 {
		return "none"
	}

	// Check if the growth of time series values is at least bigger than the threshold
	min, max := sliceMinMax(arr)
	if sign(min) == 1 && sign(max) == 1 && ((max/min - 1) < threshold) {
		return "none"
	} else if sign(min) == -1 && sign(max) == -1 && (min/max-1) < threshold {
		return "none"
	} else if sign(min) == 0 && sign(max) == 0 {
		return "none"
	} else if math.IsNaN(min) || math.IsNaN(max) {
		return "none"
	}

	// Set the periodicity parameter.
	periodicity := getPeriodicity(interval, arr)

	// Make a copy of arr since it gets changes by Decompose in case of any issues
	arrCopy := make([]float64, len(arr))
	copy(arrCopy, arr)

	// Decompose time series and work on the trend part instead
	res := stl.Decompose(arr, periodicity, len(arr)-1, stl.Multiplicative(), stl.WithRobustIter(2), stl.WithIter(2))
	trendLine := res.Trend

	// Check if trendLine has NaN values
	hasNan := hasNanValues(trendLine)
	trendIndicator := "none"

	// Get trend indicator with MK test with trendLine if possible (if it does not include NaN values) and with original time series otherwise
	if hasNan {
		trendIndicator = MKtest(arrCopy, alpha)
	} else {
		trendIndicator = MKtest(trendLine, alpha)
	}

	// Verify that the trend is correct AND that the slope at least as big as the slopeThreshold
	if trendIndicator != "none" {
		slope := getSlopeDegree(arrCopy)
		if math.Abs(slope) < slopeThreshold {
			trendIndicator = "none"
		} else if slope < 0.0 && trendIndicator == "increasing" {
			trendIndicator = "none"
		} else if slope > 0.0 && trendIndicator == "decreasing" {
			trendIndicator = "none"
		}
	}

	return trendIndicator
}

/*
rows (int): number of fields that are making the "key" (example: 2 - PROJECT and SERVICE in case of [PROJECT, SERVICE, year, month, day, COST_AMOUNT, USAGE_AMOUNT])
cols (int): number of fields that are making the time series (example: 3 - year, month, and day in case of [PROJECT, SERVICE, year, month, day, COST_AMOUNT, USAGE_AMOUNT])
metricIndex (int): index of the metric we use to build time series (example: 0 - COST_AMOUNT in case of [PROJECT, SERVICE, year, month, day, COST_AMOUNT (metric_0), USAGE_AMOUNT (metric_1)])
rowsArr ([][]bigquery.Value): array containing BQ rows
*/
func getTrendIndicatorMapping(rows int, cols int, metricIndex int, rowsArr [][]bigquery.Value, interval string) (map[string]string, error) {
	timeSeriesValueIndex := rows + cols + metricIndex // index of metric column, e.g. COST_AMOUNT

	// Initialize trends mapping
	trendsMapping := make(map[string]string)

	// Create black list of keys containing null metric values
	blackList := make([]string, 0)

	// Group rows by first value (e.g. SKU description or projectId)
	groupedRows := make(map[string][]float64)

	for _, m := range rowsArr {
		if key, err := query.GetRowKey(m, rows); err != nil {
			return nil, err
		} else {
			if m[timeSeriesValueIndex] == nil {
				// Send key to black list
				blackList = append(blackList, key)
			} else {
				var val float64
				switch v := m[timeSeriesValueIndex].(type) {
				case int64:
					val = float64(v)
				default:
					val = v.(float64)
				}

				groupedRows[key] = append(groupedRows[key], val)
			}
		}
	}

	// Find the maximum time series length
	maxTimeSeriesLen := 0
	for _, timeSeries := range groupedRows {
		if len(timeSeries) > maxTimeSeriesLen {
			maxTimeSeriesLen = len(timeSeries)
		}
	}
	// Detect trends for series with at least 75% of the amount of points in the longest series
	minTimeSeriesLen := int(maxTimeSeriesLen/4) * 3

	// Set the time series to an empty one for all keys in the black list in order to avoid trend detection in such cases
	for _, key := range blackList {
		groupedRows[key] = nil
	}

	// Get trend mapping for all time series. Process all time series in parallel.
	var l = sync.Mutex{}

	jobs := make(chan *job)

	numWorkers := 8
	if len(groupedRows) < numWorkers {
		numWorkers = len(groupedRows)
	}

	var wg sync.WaitGroup

	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(jobs <-chan *job) {
			for j := range jobs {
				trendIndicator := getTrendIndicator(j.timeSeries, interval)

				l.Lock()
				trendsMapping[j.key] = trendIndicator
				l.Unlock()
			}

			wg.Done()
		}(jobs)
	}

	for key, timeSeries := range groupedRows {
		j := job{
			key:        key,
			timeSeries: nil,
		}
		if len(timeSeries) > 0 && len(timeSeries) >= minTimeSeriesLen {
			// Don't consider the last data point for trend analysis
			// (since we don't have complete data for e.g. this day/month)
			j.timeSeries = timeSeries[:len(timeSeries)-1]
		}
		jobs <- &j
	}

	close(jobs)
	wg.Wait()

	/* Example output:
	map[alexei-playground:increasing alexsoubbotin-scc:increasing alexsoubbotin-terraform-admin: ami-host-project-2: ami-host-test-01: ami-playground:decreasing (...)]
	*/
	return trendsMapping, nil
}

/*
Detection ...
rows (int): number of fields that are making the "key" (example: 2 - PROJECT and SERVICE in case of [PROJECT, SERVICE, year, month, day, COST_AMOUNT, USAGE_AMOUNT])
cols (int): number of fields that are making the time series (example: 3 - year, month, and day in case of [PROJECT, SERVICE, year, month, day, COST_AMOUNT, USAGE_AMOUNT])
rowsArr ([][]bigquery.Value): array containing BQ rows
interval (string): data interval, can be any of the following: "hour", "day", "week", "month", "quarter", "year", ""
*/
func Detection(rows int, cols int, rowsArr [][]bigquery.Value, interval string, metricsCount int) ([][]bigquery.Value, error) {
	metricsTrendMapping := make(map[int]map[string]string)

	for m := 0; m < metricsCount; m++ {
		// Get trendIndicatorMapping
		mapping, err := getTrendIndicatorMapping(rows, cols, m, rowsArr, interval)
		if err != nil {
			return nil, err
		}

		metricsTrendMapping[m] = mapping
	}

	// Filter rows according to the specified input trend
	filteredRows := make([][]bigquery.Value, 0)

	for _, r := range rowsArr {
		if key, err := query.GetRowKey(r, rows); err != nil {
			return nil, err
		} else {
			for t := 0; t < metricsCount; t++ {
				metricTrend := metricsTrendMapping[t][key]
				r = append(r, metricTrend)
			}

			filteredRows = append(filteredRows, r)
		}
	}

	return filteredRows, nil
}

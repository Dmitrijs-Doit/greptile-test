package cloudanalytics

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	splitDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

var varMap = map[string]string{
	"A": "foo",
	"B": "bar",
	"C": "baz",
}

func TestParseMetricsFormulaExpression(t *testing.T) {
	expressions := map[string]string{
		"A+B":       fmt.Sprintf("%s+%s", varMap["A"], varMap["B"]),
		"A+5":       fmt.Sprintf("%s+5", varMap["A"]),
		"2*A+B":     fmt.Sprintf("2*%s+%s", varMap["A"], varMap["B"]),
		"2.05*A+B":  fmt.Sprintf("2.05*%s+%s", varMap["A"], varMap["B"]),
		"A*B/3":     fmt.Sprintf("%s*SAFE_DIVIDE(%s,3)", varMap["A"], varMap["B"]),
		"A-B":       fmt.Sprintf("%s-%s", varMap["A"], varMap["B"]),
		"A*B":       fmt.Sprintf("%s*%s", varMap["A"], varMap["B"]),
		"A/B":       fmt.Sprintf("SAFE_DIVIDE(%s,%s)", varMap["A"], varMap["B"]),
		"A+(B/C)*A": fmt.Sprintf("%s+(SAFE_DIVIDE(%s,%s))*%s", varMap["A"], varMap["B"], varMap["C"], varMap["A"]),
		"A+(B/C)/A": fmt.Sprintf("%s+SAFE_DIVIDE((SAFE_DIVIDE(%s,%s)),%s)", varMap["A"], varMap["B"], varMap["C"], varMap["A"]),
	}
	for input, expextedOutput := range expressions {
		expressionArr, err := formulaSplitter(input, varMap)
		assert.Nil(t, err)
		result, err := parseMetricsFormulaExpression(expressionArr, varMap)
		assert.Nil(t, err)
		assert.Equal(t, result, expextedOutput)
	}
}

// TestFormulaSplitter - test custom metrics formula parsing function
func TestFormulaSplitter(t *testing.T) {
	expressions := map[string][]string{
		"A+B":             {"A", "+", "B"},
		"A+5":             {"A", "+", "5"},
		"A+55":            {"A", "+", "55"},
		"A+55.5":          {"A", "+", "55.5"},
		"2.05*A+B":        {"2.05", "*", "A", "+", "B"},
		"A*B/3":           {"A", "*", "B", "/", "3"},
		"A-B":             {"A", "-", "B"},
		"A*B":             {"A", "*", "B"},
		"A/B":             {"A", "/", "B"},
		"A+(B/C)*505.123": {"A", "+", "(", "B", "/", "C", ")", "*", "505.123"},
	}
	for input, expectedOutput := range expressions {
		result, err := formulaSplitter(input, varMap)
		assert.Nil(t, err)
		assert.Equal(t, result, expectedOutput)
	}
}

func TestGetDateBoundryWithoutPartial(t *testing.T) {
	now := time.Now().UTC()
	dayTimeSettings := QueryRequestTimeSettings{Interval: report.TimeIntervalDay}
	dayQueryRequest := QueryRequest{
		TimeSettings: &dayTimeSettings,
	}
	dayTestCases := map[time.Time]time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC): now.Add(-36 * time.Hour).Truncate(24 * time.Hour),
	}

	for input, expectedOutput := range dayTestCases {
		result := dayQueryRequest.getDateBoundryWithoutPartial(input)
		assert.Equal(t, result, expectedOutput)
	}

	weekTimeSettings := QueryRequestTimeSettings{Interval: report.TimeIntervalWeek}
	weekQueryRequest := QueryRequest{
		TimeSettings: &weekTimeSettings,
	}

	diffToSunday := int(now.Weekday())
	if diffToSunday == 0 {
		diffToSunday = 7
	}

	weekTestCases := map[time.Time]time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC): time.Date(now.Year(), now.Month(), now.Day()-diffToSunday, 0, 0, 0, 0, time.UTC),
	}
	for input, expectedOutput := range weekTestCases {
		result := weekQueryRequest.getDateBoundryWithoutPartial(input)
		assert.Equal(t, result, expectedOutput)
	}

	monthTimeSettings := QueryRequestTimeSettings{Interval: report.TimeIntervalMonth}
	monthQueryRequest := QueryRequest{
		TimeSettings: &monthTimeSettings,
	}
	monthTestCases := map[time.Time]time.Time{
		time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC):   time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.UTC),
		time.Date(now.Year(), now.Month()-2, now.Day(), now.Hour(), 0, 0, 0, time.UTC): time.Date(now.Year(), now.Month()-2, now.Day(), now.Hour(), 0, 0, 0, time.UTC),
	}

	for input, expectedOutput := range monthTestCases {
		result := monthQueryRequest.getDateBoundryWithoutPartial(input)
		assert.Equal(t, result, expectedOutput)
	}
}

func TestGetMetricIndex(t *testing.T) {
	tests := []struct {
		qr       QueryRequest
		expected int
	}{
		{
			qr: QueryRequest{
				Metric: 0,
				IsCSP:  false,
				Count:  nil,
			},
			expected: 0,
		},
		{
			qr: QueryRequest{
				Metric: 1,
				IsCSP:  false,
				Count:  nil,
			},
			expected: 1,
		},
		{
			qr: QueryRequest{
				Metric: 2,
				IsCSP:  false,
				Count:  nil,
			},
			expected: 2,
		},
		{
			qr: QueryRequest{
				Metric: 1,
				IsCSP:  true,
				Count:  nil,
			},
			expected: 1,
		},
		{
			qr: QueryRequest{
				Metric: 3,
				IsCSP:  true,
				Count:  nil,
			},
			expected: 3,
		},
		{
			qr: QueryRequest{
				Metric: 4,
				IsCSP:  true,
				Count:  nil,
			},
			expected: 4,
		},
		{
			qr: QueryRequest{
				Metric: 0,
				IsCSP:  false,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 3,
		},
		{
			qr: QueryRequest{
				Metric: 2,
				IsCSP:  false,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 3,
		},
		{
			qr: QueryRequest{
				Metric: 1,
				IsCSP:  true,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 4,
		},
		{
			qr: QueryRequest{
				Metric: 4,
				IsCSP:  true,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 5,
		},
		{
			qr: QueryRequest{
				Metric: 5,
				IsCSP:  true,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 5,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.qr.GetMetricIndex())
	}
}

func TestGetMetricCount(t *testing.T) {
	tests := []struct {
		qr       QueryRequest
		expected int
	}{
		{
			qr: QueryRequest{
				Metric: 0,
				IsCSP:  false,
				Count:  nil,
			},
			expected: 3,
		},
		{
			qr: QueryRequest{
				Metric: 1,
				IsCSP:  false,
				Count:  nil,
			},
			expected: 3,
		},
		{
			qr: QueryRequest{
				Metric: 2,
				IsCSP:  false,
				Count:  nil,
			},
			expected: 3,
		},
		{
			qr: QueryRequest{
				Metric: 1,
				IsCSP:  true,
				Count:  nil,
			},
			expected: 4,
		},
		{
			qr: QueryRequest{
				Metric: 3,
				IsCSP:  true,
				Count:  nil,
			},
			expected: 4,
		},
		{
			qr: QueryRequest{
				Metric: 4,
				IsCSP:  true,
				Count:  nil,
			},
			expected: 5,
		},
		{
			qr: QueryRequest{
				Metric: 0,
				IsCSP:  false,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 4,
		},
		{
			qr: QueryRequest{
				Metric: 2,
				IsCSP:  false,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 4,
		},
		{
			qr: QueryRequest{
				Metric: 4,
				IsCSP:  false,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 5,
		},
		{
			qr: QueryRequest{
				Metric: 5,
				IsCSP:  false,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 5,
		},
		{
			qr: QueryRequest{
				Metric: 1,
				IsCSP:  true,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 5,
		},
		{
			qr: QueryRequest{
				Metric: 4,
				IsCSP:  true,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 6,
		},
		{
			qr: QueryRequest{
				Metric: 5,
				IsCSP:  true,
				Count: &domainQuery.QueryRequestCount{
					Field: "T.cloud_provider",
				},
			},
			expected: 6,
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expected, test.qr.GetMetricCount())
	}
}

func TestProcessSplit(t *testing.T) {
	tests := []struct {
		name     string
		rows     []*domainQuery.QueryRequestX
		split    splitDomain.Split
		expected []*domainQuery.QueryRequestX
	}{
		{
			name: "Include origin",
			rows: []*domainQuery.QueryRequestX{
				{ID: "1", Label: "label1"},
				{ID: "2", Label: "label2"},
			},
			split: splitDomain.Split{
				ID:            "1",
				IncludeOrigin: true,
			},
			expected: []*domainQuery.QueryRequestX{
				{ID: "1", Label: "label1"},
				{ID: "1", Label: "label1 - origin"},
				{ID: "2", Label: "label2"},
			},
		},
		{
			name: "Exclude origin",
			rows: []*domainQuery.QueryRequestX{
				{ID: "1", Label: "label1"},
				{ID: "2", Label: "label2"},
			},
			split: splitDomain.Split{
				ID:            "1",
				IncludeOrigin: false,
			},
			expected: []*domainQuery.QueryRequestX{
				{ID: "1", Label: "label1"},
				{ID: "2", Label: "label2"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			returnedRows := processSplit(tc.rows, tc.split)
			assert.Equal(t, len(tc.expected), len(returnedRows), "Failed to add a row for origin %s", tc.name)
			assert.Equal(t, tc.expected[1].Label, returnedRows[1].Label, "Failed to add a row for origin %s", tc.name)
		})
	}
}

func TestGetMetricString(t *testing.T) {
	extendedMetricFlexsave := "flexsave"

	tests := []struct {
		name           string
		metric         report.Metric
		extendedMetric *string
		expectedRes    string
		err            error
	}{
		{
			name:        "ok cost",
			metric:      report.MetricCost,
			expectedRes: "report_value.cost * currency_conversion_rate",
		},
		{
			name:        "ok usage",
			metric:      report.MetricUsage,
			expectedRes: "report_value.usage",
		},
		{
			name:        "ok savings",
			metric:      report.MetricSavings,
			expectedRes: "report_value.savings * currency_conversion_rate",
		},
		{
			name:           "ok extended",
			metric:         report.MetricExtended,
			extendedMetric: &extendedMetricFlexsave,
			expectedRes:    `IF(report_value.ext_metric.key = "flexsave", report_value.ext_metric.value * IF(report_value.ext_metric.type = "cost", currency_conversion_rate, 1), 0)`,
		},
		{
			name:   "extended metric but metric not provided",
			metric: report.MetricExtended,
			err:    ExtendedMetricNotProvided,
		},
		{
			name:   "custom not supported",
			metric: report.MetricCustom,
			err:    MetricNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := getMetricQueryString(tt.metric, tt.extendedMetric)
			if !errors.Is(err, tt.err) {
				t.Errorf("getMetricQueryString() error = %v, wantErr %v", err, tt.err)
			}

			if err == nil {
				assert.Equal(t, tt.expectedRes, res)
			}
		})
	}
}

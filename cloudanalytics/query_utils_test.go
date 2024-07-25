package cloudanalytics

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestCreateDateFilters(t *testing.T) {
	to := time.Now().UTC()
	from := to.Add(-time.Hour * 24 * 7)
	expectedOutput := []string{
		// Should look like (dates vary): `DATE(T.usage_date_time) BETWEEN DATE("2021-10-21") AND DATE("2021-11-02")`
		fmt.Sprintf(`DATE(T.usage_date_time) BETWEEN DATE("%s") AND DATE("%s")`, from.Format(layout), to.Format(layout)),
		"DATE(T.export_time) >= DATE(@partition_start)",
		"DATE(T.export_time) <= DATE(@partition_end)",
	}
	dateFilters := createDateFilters(from, to)
	assert.Equal(t, dateFilters, expectedOutput)
}

func TestIsValidTimeSeriesReport(t *testing.T) {
	var testData = []struct {
		name     string
		interval report.TimeInterval
		cols     []*domainQuery.QueryRequestX
		result   bool
	}{
		{
			"Empty time interval",
			report.TimeInterval(""),
			[]*domainQuery.QueryRequestX{
				{Type: metadata.MetadataFieldTypeDatetime, Key: "year"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "month"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "day"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "week_day"},
			},
			false,
		},
		{
			"Not matching number of cols",
			report.TimeIntervalDay,
			[]*domainQuery.QueryRequestX{
				{Type: metadata.MetadataFieldTypeDatetime, Key: "year"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "month"},
			},
			false,
		},
		{
			"Not matching col type",
			report.TimeIntervalDay,
			[]*domainQuery.QueryRequestX{
				{Type: metadata.MetadataFieldTypeDatetime, Key: "year"},
				{Type: metadata.MetadataFieldTypeFixed, Key: "other"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "month"},
			},
			false,
		},
		{
			"Not matching col and key",
			report.TimeIntervalDay,
			[]*domainQuery.QueryRequestX{
				{Type: metadata.MetadataFieldTypeDatetime, Key: "year"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "other"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "month"},
			},
			false,
		},
		{
			"Not matching col and key with multiple",
			report.TimeIntervalDay,
			[]*domainQuery.QueryRequestX{
				{Type: metadata.MetadataFieldTypeDatetime, Key: "year"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "other"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "month"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "day"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "week_day"},
			},
			false,
		},
		{
			"Matching",
			report.TimeIntervalDay,
			[]*domainQuery.QueryRequestX{
				{Type: metadata.MetadataFieldTypeDatetime, Key: "year"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "month"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "day"},
				{Type: metadata.MetadataFieldTypeDatetime, Key: "week_day"},
			},
			true,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			res := isValidTimeSeriesReport(tt.interval, tt.cols)
			assert.Equal(t, tt.result, res)
		})
	}
}

func TestCanUseAggregatedTable(t *testing.T) {
	var testData = []struct {
		name   string
		qr     *QueryRequest
		attrs  []*domainQuery.QueryRequestX
		result bool
	}{
		{
			"noAggregate query can not use aggregated table",
			&QueryRequest{
				NoAggregate: true,
			},
			[]*domainQuery.QueryRequestX{},
			false,
		},
		{
			"count field of type MetadataFieldTypeLabel can not use aggregated table",
			&QueryRequest{
				Count: &domainQuery.QueryRequestCount{
					Type: metadata.MetadataFieldTypeLabel,
				},
			},
			[]*domainQuery.QueryRequestX{},
			false,
		},
		{
			"count field of type MetadataFieldTypeFixed can use aggregated table",
			&QueryRequest{
				Count: &domainQuery.QueryRequestCount{
					Type: metadata.MetadataFieldTypeFixed,
				},
			},
			[]*domainQuery.QueryRequestX{},
			true,
		},
		{
			"row field of type MetadataFieldTypeLabel can not use aggregated table",
			&QueryRequest{
				Rows: []*domainQuery.QueryRequestX{
					{
						Type: metadata.MetadataFieldTypeLabel,
					},
				},
			},
			[]*domainQuery.QueryRequestX{},
			false,
		},
		{
			"row field of type MetadataFieldTypeDatetime can use aggregated table",
			&QueryRequest{
				Rows: []*domainQuery.QueryRequestX{
					{
						Type: metadata.MetadataFieldTypeDatetime,
					},
				},
			},
			[]*domainQuery.QueryRequestX{},
			true,
		},
		{
			"row field of type Attribution with Composite element 'MetadataFieldTypeLabel' can not use aggregated table",
			&QueryRequest{
				Rows: []*domainQuery.QueryRequestX{
					{
						Type: metadata.MetadataFieldTypeAttribution,
						Composite: []*domainQuery.QueryRequestX{
							{
								Type: metadata.MetadataFieldTypeLabel,
							},
						},
					},
				},
			},
			[]*domainQuery.QueryRequestX{},
			false,
		},
		{
			"row field of type Attribution with Composite element 'MetadataFieldTypeFixed' can use aggregated table",
			&QueryRequest{
				Rows: []*domainQuery.QueryRequestX{
					{
						Type: metadata.MetadataFieldTypeAttribution,
						Composite: []*domainQuery.QueryRequestX{
							{
								Type: metadata.MetadataFieldTypeFixed,
							},
						},
					},
				},
			},
			[]*domainQuery.QueryRequestX{},
			true,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.qr.canUseAggregatedTable(tt.attrs)
			assert.Equal(t, tt.result, res)
		})
	}
}

func TestDedupeQueryParamNames(t *testing.T) {
	params := []bigquery.QueryParameter{
		{Name: "name1", Value: "value1"},
		{Name: "name2", Value: "value2"},
		{Name: "name3", Value: []string{"val1", "val2"}},
		{Name: "name4", Value: "value4"},
		{Name: "name5", Value: "value5"},
		{Name: "name6", Value: "value6"},
		{Name: "name2", Value: "value2"},                 // duplicate parameter
		{Name: "name3", Value: []string{"val1", "val2"}}, // duplicate parameter
	}

	expected := []bigquery.QueryParameter{
		{Name: "name1", Value: "value1"},
		{Name: "name2", Value: "value2"},
		{Name: "name3", Value: []string{"val1", "val2"}},
		{Name: "name4", Value: "value4"},
		{Name: "name5", Value: "value5"},
		{Name: "name6", Value: "value6"},
	}

	result := dedupeQueryParamNames(params)

	assert.Equal(t, len(expected), len(result), "Length of result slice does not match expected length")

	// Create a map to hold the expected values for easy comparison
	expectedMap := make(map[string]interface{})
	for _, param := range expected {
		expectedMap[param.Name] = param.Value
	}

	for _, param := range result {
		expectedValue, ok := expectedMap[param.Name]
		assert.True(t, ok, "Parameter with name %q not found in expected slice", param.Name)
		assert.Equal(t, expectedValue, param.Value, "Parameter with name %q has unexpected value", param.Name)
		delete(expectedMap, param.Name)
	}

	assert.Equal(t, 0, len(expectedMap), "Expected parameters not found in result slice: %v", expectedMap)
}

func TestResultSizeValidation(t *testing.T) {
	testData := []struct {
		name       string
		result     [][]bigquery.Value
		numGroupBy int
		rThreshold int
		sThreshold int
		wantErr    bool
	}{
		{
			name: "Too many rows",
			result: [][]bigquery.Value{
				{"a", "b", "1"},
				{"a", "b", "2"},
				{"a", "b", "3"},
			},
			rThreshold: 2,
			wantErr:    true,
		},
		{
			name: "Too many series",
			result: [][]bigquery.Value{
				{"a", "c", "1"},
				{"b", "d", "2"},
				{"a", "e", "3"},
				{"b", "c", "4"},
				{"b", "d", "2"},
				{"a", "e", "6"},
			},
			rThreshold: 10,
			sThreshold: 4,
			numGroupBy: 3,
			wantErr:    true,
		},
		{
			name: "Happy path",
			result: [][]bigquery.Value{
				{"a", "c", "1"},
				{"b", "d", "2"},
				{"a", "e", "3"},
				{"b", "c", "4"},
				{"b", "d", "2"},
				{"a", "e", "6"},
			},
			rThreshold: 10,
			sThreshold: 5,
			numGroupBy: 3,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			err := resultSizeValidation(tt.result, tt.numGroupBy, tt.rThreshold, tt.sThreshold)
			if (err != nil) != tt.wantErr {
				t.Errorf("resultSizeValidation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

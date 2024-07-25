package service

import (
	"strconv"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
)

func TestNewLimitService_ApplyLimits(t *testing.T) {
	l := NewLimitService()

	var resultRows [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "full_results_two.json", &resultRows); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	var expectedRows [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "two_levels.json", &expectedRows); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	rows, filters := getRowsFilters()

	res, err := l.ApplyLimits(resultRows, filters, rows, report.LimitAggregationNone, 4)

	assert.Nil(t, err)

	assert.Equal(t, len(expectedRows), len(res))

	assert.ElementsMatch(t, res, expectedRows)
}

func TestNewLimitService_applyLimit(t *testing.T) {
	var rows [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "full_results.json", &rows); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	var limitRows [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "one_level.json", &limitRows); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	oneLevel := []LevelLimit{
		{Limit: 5, LimitMetricIndex: 3, LimitOrder: "desc", LimitIndex: 0, LimitReverse: false, Level: 0},
	}

	result1, _ := applyLimit(rows, oneLevel, report.LimitAggregationNone)

	assert.Equal(t, len(limitRows), len(result1))

	assert.ElementsMatchf(t, limitRows, result1, "the result should be the same as the limit rows")

	twoLevels := []LevelLimit{
		{Limit: 5, LimitMetricIndex: 4, LimitOrder: "desc", LimitIndex: 0, LimitReverse: false, Level: 0},
		{Limit: 3, LimitMetricIndex: 4, LimitOrder: "desc", LimitIndex: 1, LimitReverse: false, Level: 1},
	}

	var rows2 [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "full_results_two.json", &rows2); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	var limitRows2 [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "two_levels.json", &limitRows2); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	result2, _ := applyLimit(rows2, twoLevels, report.LimitAggregationNone)

	assert.Equal(t, len(limitRows2), len(result2))

	assert.ElementsMatchf(t, limitRows2, result2, "the result should be the same as the limit rows")

	threeLevels := []LevelLimit{
		{Limit: 5, LimitMetricIndex: 4, LimitOrder: "desc", LimitIndex: 0, LimitReverse: false, Level: 0},
		{Limit: 3, LimitMetricIndex: 4, LimitOrder: "desc", LimitIndex: 1, LimitReverse: false, Level: 1},
		{Limit: 4, LimitMetricIndex: 4, LimitOrder: "desc", LimitIndex: 2, LimitReverse: false, Level: 2},
	}

	var rows3 [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "full_results_two.json", &rows3); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	var limitRows3 [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "two_levels.json", &limitRows3); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	result3, _ := applyLimit(rows3, threeLevels, report.LimitAggregationNone)

	assert.Equal(t, len(limitRows3), len(result3))

	assert.ElementsMatchf(t, limitRows3, result3, "the result should be the same as the limit rows")
}

func TestNewLimitService_sortAggregateData(t *testing.T) {
	// create some test data
	data := []LimitIndexAggregateData{
		{Key: "service-1", MetricSum: 6.0, Rows: [][]bigquery.Value{{1}, {2}, {3}}},
		{Key: "service-2", MetricSum: 15.0, Rows: [][]bigquery.Value{{4}, {5}, {6}}},
		{Key: "service-3", MetricSum: 24.0, Rows: [][]bigquery.Value{{7}, {8}, {9}}},
	}

	level := LevelLimit{
		Limit: 2, LimitMetricIndex: 3, LimitOrder: "desc", LimitIndex: 0, LimitReverse: false, Level: 0,
	}

	resultMap := make(map[string]LimitIndexAggregateData)
	for i, d := range data {
		resultMap[strconv.Itoa(i)] = d
	}

	// test sorting in ascending order
	sorted := sortAggregateData(resultMap, level, report.LimitAggregationNone)

	assert.Equal(t, 2, len(sorted))
	assert.Equal(t, 24.0, sorted[0].MetricSum)
	assert.Equal(t, 15.0, sorted[1].MetricSum)

	// test sorting in descending order
	level.LimitReverse = true
	sorted = sortAggregateData(resultMap, level, report.LimitAggregationNone)

	assert.Equal(t, 2, len(sorted))
	assert.Equal(t, 6.0, sorted[0].MetricSum)
	assert.Equal(t, 15.0, sorted[1].MetricSum)

	// test level with limit 0 and aggregation top (takes all items)
	level.Limit = 0
	level.LimitAggregation = report.LimitAggregationTop
	sorted = sortAggregateData(resultMap, level, report.LimitAggregationTop)

	assert.Equal(t, 3, len(sorted))
	assert.Equal(t, 6.0, sorted[0].MetricSum)
	assert.Equal(t, 15.0, sorted[1].MetricSum)
	assert.Equal(t, 24.0, sorted[2].MetricSum)
}

func TestNewLimitService_getLimitGroupsMap(t *testing.T) {
	// Test case with no rows
	var (
		rows       [][]bigquery.Value
		limitIndex int
	)

	metricIndex := 1

	var expected = make(map[string]LimitIndexAggregateData)

	got, err := getLimitGroupsMap(rows, limitIndex, metricIndex)

	if len(got) != len(expected) {
		t.Errorf("Expected %d groups, but got %d", len(expected), len(got))
	}

	assert.Nil(t, err)

	// Test case with one row
	rows = [][]bigquery.Value{{"A", 10.0}}
	metricIndex = 1
	expected = map[string]LimitIndexAggregateData{
		"A": {
			Key:       "A",
			MetricSum: 10.0,
			Rows:      rows,
		},
	}
	got, err = getLimitGroupsMap(rows, limitIndex, metricIndex)

	assert.Nil(t, err)

	if len(got) != len(expected) {
		t.Errorf("Expected %d groups, but got %d", len(expected), len(got))
	}

	assert.Equal(t, expected, got)

	// Test case with multiple rows
	rows = [][]bigquery.Value{
		{"A", 10.0},
		{"B", 20.0},
		{"A", 30.0},
	}
	limitIndex = 0
	metricIndex = 1
	expected = map[string]LimitIndexAggregateData{
		"A": {
			Key:       "A",
			MetricSum: 40.0,
			Rows: [][]bigquery.Value{
				{"A", 10.0},
				{"A", 30.0},
			},
		},
		"B": {
			Key:       "B",
			MetricSum: 20.0,
			Rows: [][]bigquery.Value{
				{"B", 20.0},
			},
		},
	}

	got, err = getLimitGroupsMap(rows, limitIndex, metricIndex)
	if len(got) != len(expected) {
		t.Errorf("Expected %d groups, but got %d", len(expected), len(got))
	}

	assert.Nil(t, err)

	assert.Equal(t, expected, got)

	// Test case with multiple rows
	expectedMultipleRows := map[string]LimitIndexAggregateData{
		"A": {
			Key:       "A",
			MetricSum: 30.0,
			Rows: [][]bigquery.Value{
				{"A", 30.0},
			},
		},
		"B": {
			Key:       "B",
			MetricSum: 20.0,
			Rows: [][]bigquery.Value{
				{"B", 20.0},
			},
		},
		"false": {
			Key:       "false",
			MetricSum: 10.0,
			Rows: [][]bigquery.Value{
				{false, 10.0},
			},
		},
	}

	rows = [][]bigquery.Value{
		{false, 10.0},
		{"B", 20.0},
		{"A", 30.0},
	}

	got, err = getLimitGroupsMap(rows, limitIndex, metricIndex)
	assert.Nil(t, err)

	assert.Equal(t, expectedMultipleRows, got)
}

func TestNewLimitService_getTopGroups(t *testing.T) {
	sortedData := []LimitIndexAggregateData{
		{Key: "B", MetricSum: 6.0, Rows: [][]bigquery.Value{{"B", 2.0}, {"B", 4.0}}},
		{Key: "A", MetricSum: 6.0, Rows: [][]bigquery.Value{{"A", 1.0}, {"A", 5.0}}},
	}

	topGroups := getTopGroups(sortedData)

	expectedMap := map[string]LimitIndexAggregateData{
		"B": {Key: "B", MetricSum: 6.0, Rows: [][]bigquery.Value{{"B", 2.0}, {"B", 4.0}}},
		"A": {Key: "A", MetricSum: 6.0, Rows: [][]bigquery.Value{{"A", 1.0}, {"A", 5.0}}},
	}

	assert.Equal(t, expectedMap, topGroups)
}

func TestNewLimitService_getAggregatedGroup(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name           string
		data           LimitIndexAggregateDataSlice
		aggregate      report.LimitAggregation
		level          LevelLimit
		expectedResult *LimitIndexAggregateData
	}{
		{
			name: "Test aggregation with LimitAggregationAll",
			data: LimitIndexAggregateDataSlice{
				{
					Key:       "key1",
					MetricSum: 10.0,
					Rows: [][]bigquery.Value{
						{"value1", 6},
						{"value2", 4},
					},
				},
				{
					Key:       "key2",
					MetricSum: 20.0,
					Rows: [][]bigquery.Value{
						{"value3", 11},
						{"value4", 9},
					},
				},
			},
			aggregate: report.LimitAggregationAll,
			level: LevelLimit{
				LimitPlural: "services",
				LimitIndex:  0,
				Limit:       0,
			},
			expectedResult: &LimitIndexAggregateData{
				Key:       aggregatedLabelPrefix + "services",
				MetricSum: 30.0,
				Rows: [][]bigquery.Value{
					{aggregatedLabelPrefix + "services", 6},
					{aggregatedLabelPrefix + "services", 4},
					{aggregatedLabelPrefix + "services", 11},
					{aggregatedLabelPrefix + "services", 9},
				},
			},
		},
		{
			name: "Test aggregation with LimitAggregationTop",
			data: LimitIndexAggregateDataSlice{
				{
					Key:       "key1",
					MetricSum: 10.0,
					Rows: [][]bigquery.Value{
						{"value1", "value12", 4},
						{"value2", "value23", 6},
					},
				},
				{
					Key:       "key2",
					MetricSum: 20.0,
					Rows: [][]bigquery.Value{
						{"value3", "value34", 3},
						{"value4", "value45", 4},
					},
				},
			},
			aggregate: report.LimitAggregationTop,
			level: LevelLimit{
				LimitPlural: "SKUs",
				LimitIndex:  0,
				Limit:       0,
			},
			expectedResult: &LimitIndexAggregateData{
				Key:       aggregatedLabelPrefix + "SKUs",
				MetricSum: 30.0,
				Rows: [][]bigquery.Value{
					{aggregatedLabelPrefix + "SKUs", "value12", 4},
					{aggregatedLabelPrefix + "SKUs", "value23", 6},
					{aggregatedLabelPrefix + "SKUs", "value34", 3},
					{aggregatedLabelPrefix + "SKUs", "value45", 4},
				},
			},
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getAggregatedGroup(tc.data, tc.aggregate, tc.level)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestNewLimitService_getPlurals(t *testing.T) {
	rows, filters := getRowsFilters()
	agRow := &domainQuery.QueryRequestX{Key: "kdcpGccsYlGz8hTu4pZS", ID: "attribution_group:kdcpGccsYlGz8hTu4pZS", Type: "attribution_group", Label: "Engineering Teams"}
	rows = append(rows, agRow)

	agFilter := &domainQuery.QueryRequestX{Key: "attribution_group", ID: "attribution_group:kdcpGccsYlGz8hTu4pZS", Type: "attribution_group"}
	filters = append(filters, agFilter)

	expecetedPlurals := []string{"countries", "services", "Engineering Teams"}

	for i, filter := range filters {
		limitIndex := domainQuery.FindIndexInQueryRequestX(rows, filter.ID)
		plural := getPlural(rows, filter.Key, filter.Type, limitIndex)
		assert.Equal(t, plural, expecetedPlurals[i])
	}
}

func TestNewLimitService_getPreviousLimitPlurals(t *testing.T) {
	var emptySlice []string

	level1 := &LevelLimit{
		Level:           1,
		LimitPlural:     "apples",
		LimitPluralPrev: "",
		LevelPrev:       nil,
	}

	level2 := &LevelLimit{
		Level:           2,
		LimitPlural:     "oranges",
		LimitPluralPrev: "",
		LevelPrev:       level1,
	}

	level3 := &LevelLimit{
		Level:           3,
		LimitPlural:     "bananas",
		LimitPluralPrev: "fruits",
		LevelPrev:       level2,
	}

	level4 := &LevelLimit{
		Level:           4,
		LimitPlural:     "pears",
		LimitPluralPrev: "fruits",
		LevelPrev:       level3,
	}

	tests := []struct {
		name     string
		level    *LevelLimit
		expected []string
	}{
		{
			name:     "No previous LimitPlurals",
			level:    level1,
			expected: emptySlice,
		},
		{
			name:     "Single previous LimitPlural",
			level:    level2,
			expected: []string{"apples"},
		},
		{
			name:     "Multiple previous LimitPlurals",
			level:    level4,
			expected: []string{"bananas", "oranges", "apples"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plurals := getPreviousLimitPlurals(test.level)
			assert.Equal(t, test.expected, plurals)
		})
	}
}

func TestNewLimitService_getLevelRows(t *testing.T) {
	rows, filters := getRowsFilters()
	desc := "desc"
	rowWithoutFilter := &domainQuery.QueryRequestX{LimitConfig: domainQuery.LimitConfig{Limit: 0, LimitOrder: &desc}, Key: "sku", ID: "fixed:sku", Type: "fixed"}
	appendedIndex := len(rows) - 1
	rows = append(rows[:appendedIndex], append([]*domainQuery.QueryRequestX{rowWithoutFilter}, rows[appendedIndex:]...)...)

	result := getLevelRows(rows, filters)

	assert.Equal(t, len(rows), len(result))
	assert.Equal(t, result[1].LimitConfig.Limit, 0)

	for i, levelRow := range result {
		assert.Equal(t, rows[i].Key, levelRow.Key)
		assert.Equal(t, rows[i].ID, levelRow.ID)
		assert.Equal(t, rows[i].Type, levelRow.Type)
	}
}

func getRowsFilters() ([]*domainQuery.QueryRequestX, []*domainQuery.QueryRequestX) {
	desc := "desc"
	rows := []*domainQuery.QueryRequestX{
		{Key: "country", ID: "fixed:country", Type: "fixed"},
		{Key: "service_description", ID: "fixed:service_description", Type: "fixed"},
	}

	filters := []*domainQuery.QueryRequestX{
		{LimitConfig: domainQuery.LimitConfig{Limit: 5, LimitOrder: &desc}, Key: "country", ID: "fixed:country", Type: "fixed"},
		{LimitConfig: domainQuery.LimitConfig{Limit: 3, LimitOrder: &desc}, Key: "service_description", ID: "fixed:service_description", Type: "fixed"},
	}

	return rows, filters
}

package service

import (
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain"
	domainSplit "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
)

const convertJSONErrMsgFmt = "could not convert json test file into struct. error %s"

func TestService_Split(t *testing.T) {
	tests := []struct {
		name            string
		splitParamsFile string
		wantRowsFile    string
	}{
		{
			name:            "Even split",
			splitParamsFile: "build_split_params.json",
			wantRowsFile:    "new_rows_post_split.json",
		},
		{
			name:            "Custom split",
			splitParamsFile: "build_split_params2.json",
			wantRowsFile:    "new_rows_post_split2.json",
		},
		{
			name:            "Proportional split",
			splitParamsFile: "build_split_params3.json",
			wantRowsFile:    "new_rows_post_split3.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSplittingService()

			var splitParams domain.BuildSplit
			if err := testTools.ConvertJSONFileIntoStruct("testData", tt.splitParamsFile, &splitParams); err != nil {
				t.Fatalf(convertJSONErrMsgFmt, err)
			}

			var newResRows [][]bigquery.Value
			if err := testTools.ConvertJSONFileIntoStruct("testData", tt.wantRowsFile, &newResRows); err != nil {
				t.Fatalf(convertJSONErrMsgFmt, err)
			}

			if err := s.Split(splitParams); err != nil {
				t.Fatalf("Split() returned error: %s", err)
			}

			originRows := *splitParams.ResRows
			assert.Equal(t, len(originRows), len(newResRows), "Split() returned unexpected result")
			// check that the slices are have the same elements but not in order
			assert.ElementsMatch(t, originRows, newResRows, "Split() returned unexpected result")
		})
	}
}

func TestService_ValidateSplitsReq(t *testing.T) {
	tests := []struct {
		name     string
		req      *[]domainSplit.Split
		wantErrs []error
	}{
		{
			name: "ID used as origin and target in same split",
			req: &[]domainSplit.Split{
				{
					ID:     "attribution_group:split1",
					Origin: "attribution:def",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:def", Value: 123},
						{ID: "attribution:ghi", Value: 123},
					},
				},
			},
			wantErrs: []error{
				NewValidationError(
					ValidationErrorTypeIDCannotBeOriginAndTargetInSameSplit,
					"attribution_group:split1",
					"attribution:def",
				),
			},
		},
		{
			name: "ID used in circular dependency as both target and origin",
			req: &[]domainSplit.Split{
				{
					ID:     "attribution_group:split2",
					Origin: "attribution:abc",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:def", Value: 123},
						{ID: "attribution:qrp", Value: 123},
					},
				},
				{
					ID:     "attribution_group:xyz",
					Origin: "attribution:def",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:abc", Value: 123}, // This target has the same ID as the origin in the first split
						{ID: "attribution:ghi", Value: 123},
					},
				},
			},
			wantErrs: []error{
				NewValidationError(
					ValidationErrorTypeCircularDependency,
					"attribution_group:split2",
					"attribution:def",
				),
				NewValidationError(
					ValidationErrorTypeCircularDependency,
					"attribution_group:xyz",
					"attribution:abc",
				),
			},
		},
		{
			name: "success no duplicate IDs",
			req: &[]domainSplit.Split{
				{
					ID:     "attribution-group:split1",
					Origin: "def",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:ghi", Value: 123},
					},
				},
				{
					ID:     "attribution-group:split2",
					Origin: "abc",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:def", Value: 123},
					},
				},
			},
		},
		{
			name: "error on unsupported split type",
			req: &[]domainSplit.Split{
				{
					ID:     "attribution-group:split1",
					Origin: "def",
					Type:   "some-invalid-type",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:ghi", Value: 123},
					},
				},
				{
					ID:     "attribution-group:split2",
					Origin: "abc",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:def", Value: 123},
					},
				},
			},
			wantErrs: []error{ErrInvalidSplitType},
		},
		{
			name: "error on the duplicate origin",
			req: &[]domainSplit.Split{
				{
					ID:     "attribution_group:split1",
					Origin: "attribution:def",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:ghi", Value: 123},
					},
				},
				{
					ID:     "attribution_group:split2",
					Origin: "attribution:def",
					Type:   "attribution_group",
					Targets: []domainSplit.SplitTarget{
						{ID: "attribution:abc", Value: 456},
					},
				},
			},
			wantErrs: []error{
				NewValidationError(
					ValidationErrorTypeOriginDuplicated,
					"attribution_group:split2",
					"attribution:def",
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSplittingService()

			if errs := s.ValidateSplitsReq(tt.req); !assert.ElementsMatch(t, errs, tt.wantErrs) {
				t.Errorf("ValidateSplitsReq() errs = %v, wantErrs = %v", errs, tt.wantErrs)
			}
		})
	}
}

func TestService_CreateNewRow(t *testing.T) {
	originRow1 := []bigquery.Value{"project1", nil, nil, 1.0, 2.0}
	expectedRow1 := []bigquery.Value{"project1", "bruteforce", "Unallocated", 0.0, 0.0}
	originRow2 := []bigquery.Value{"project1", "kraken", 18.0, 3.6}
	expectedRow2 := []bigquery.Value{"project1", "kraken", 0.0, 0.0}

	// Test case 1: originID is not empty
	newRow1 := createNewRow(5, 3, 2, 1, originRow1, "bruteforce", "Unallocated")
	assert.Equal(t, expectedRow1, newRow1, "createNewRow() returned unexpected result")

	// Test case 2: originID is empty
	newRow2 := createNewRow(4, 2, 2, 1, originRow2, "kraken", "")
	assert.Equal(t, expectedRow2, newRow2, "createNewRow() returned unexpected result")
}

func TestService_AddSplitValuesToRow(t *testing.T) {
	tests := []struct {
		name                 string
		originRow            []bigquery.Value
		targetRow            []bigquery.Value
		initialMetricsValues []float64
		metricLength         int
		metricOffset         int
		splitValue           []float64
		expectedRow          []bigquery.Value
	}{
		{
			name:                 "success with adding from 'unallocated' origin",
			originRow:            []bigquery.Value{"project1", nil, nil, 1.0, 2.0},
			targetRow:            []bigquery.Value{"project1", "bruteforce", "Unallocated", 0.0, 0.0},
			initialMetricsValues: []float64{1.0, 2.0},
			metricLength:         2,
			metricOffset:         3,
			splitValue:           []float64{0.5},
			expectedRow:          []bigquery.Value{"project1", "bruteforce", "Unallocated", 0.5, 1.0},
		},
		{
			name:                 "split from a non 'Unallocated' origin",
			originRow:            []bigquery.Value{"project1", "bruteforce", 18.0, 3.6},
			targetRow:            []bigquery.Value{"project1", "kraken", 0.0, 0.0},
			initialMetricsValues: []float64{18.0, 3.6},
			metricLength:         2,
			metricOffset:         2,
			splitValue:           []float64{1.0 / 3.0},
			expectedRow:          []bigquery.Value{"project1", "kraken", 6.0, 1.2},
		},
		{
			name:                 "split from a non 'Unallocated' origin, values are int",
			originRow:            []bigquery.Value{"project1", "bruteforce", int32(18), 3.6},
			targetRow:            []bigquery.Value{"project1", "kraken", int8(0), 0},
			initialMetricsValues: []float64{18.0, 3.6},
			metricLength:         2,
			metricOffset:         2,
			splitValue:           []float64{1.0 / 3.0},
			expectedRow:          []bigquery.Value{"project1", "kraken", 6.0, 1.2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addSplitValuesToRow(tt.originRow, tt.targetRow, tt.initialMetricsValues, tt.metricLength, tt.metricOffset, tt.splitValue)
			assert.Equal(t, tt.expectedRow, tt.targetRow, "addSplitValuesToRow() returned unexpected result")
		})
	}
}

func TestService_RemoveSplitRows(t *testing.T) {
	// Define input slice with some rows.
	rows := [][]bigquery.Value{
		{"project1", "bruteforce", nil, 0.5, 1.0},
		{"project2", nil, 10, 0.2, 0.5},
		{"project3", nil, 5, 0.8, 0.9},
	}

	// Define indices of rows to be removed.
	indices := []int{1, 2}

	// Remove the specified rows.
	removeSplitRows(&rows, indices)

	// Define expected output slice after removal.
	expectedRows := [][]bigquery.Value{
		{"project1", "bruteforce", nil, 0.5, 1.0},
	}

	// Use assert package to check that the actual output matches the expected output.
	assert.Equal(t, expectedRows, rows, "removeSplitRows(%v, %v) = %v; expected %v", rows, indices, rows, expectedRows)
}

func TestService_RemoveSplitKey(t *testing.T) {
	slice := []bigquery.Value{"a", "b", "c", "d"}

	// remove element in the middle
	expected1 := []bigquery.Value{"a", "c", "d"}
	newSlice1 := removeSplitKey(slice, 1)
	assert.Equal(t, expected1, newSlice1, "removeSplitKey() returned unexpected result")

	// remove last element
	expected2 := []bigquery.Value{"a", "b", "c"}
	newSlice2 := removeSplitKey(slice, len(slice)-1)
	assert.Equal(t, expected2, newSlice2, "removeSplitKey() returned unexpected result")
}

func TestService_InitializeSplitMap(t *testing.T) {
	// split with two targets
	split2 := domainSplit.Split{
		Origin: "origin",
		Targets: []domainSplit.SplitTarget{
			{
				ID: "target1",
			},
			{
				ID: "target2",
			},
		},
	}
	expectedMap2 := map[string]map[string]int{
		"origin":  {},
		"target1": {},
		"target2": {},
	}
	resultMap2 := initializeSplitMap(split2)
	assert.Equal(t, expectedMap2, resultMap2, "initializeSplitMap() returned unexpected result")
}

func TestService_CalculatePercentageSplitValues(t *testing.T) {
	testCases := []struct {
		name     string
		split    *domainSplit.Split
		expected []domainSplit.SplitTarget
	}{
		{
			name: "all non-zero values",
			split: &domainSplit.Split{
				Targets: []domainSplit.SplitTarget{
					{ID: "target1", Value: 0.25},
					{ID: "target2", Value: 0.5},
					{ID: "target3", Value: 0.25},
				},
			},
			expected: []domainSplit.SplitTarget{
				{ID: "target1", Value: 0.25},
				{ID: "target2", Value: 0.5},
				{ID: "target3", Value: 0.25},
			},
		},
		{
			name: "some zero values",
			split: &domainSplit.Split{
				Targets: []domainSplit.SplitTarget{
					{ID: "target1", Value: 0.25},
					{ID: "target2", Value: 0},
					{ID: "target3", Value: 0.25},
					{ID: "target4", Value: 0},
					{ID: "target5", Value: 0.5},
				},
			},
			expected: []domainSplit.SplitTarget{
				{ID: "target1", Value: 0.25},
				{ID: "target2", Value: 0},
				{ID: "target3", Value: 0.25},
				{ID: "target4", Value: 0},
				{ID: "target5", Value: 0.5},
			},
		},
		{
			name: "all zero values",
			split: &domainSplit.Split{
				Targets: []domainSplit.SplitTarget{
					{ID: "target1", Value: 0},
					{ID: "target2", Value: 0},
					{ID: "target3", Value: 0},
				},
			},
			expected: []domainSplit.SplitTarget{
				{ID: "target1", Value: 1 / 3.0},
				{ID: "target2", Value: 1 / 3.0},
				{ID: "target3", Value: 1 / 3.0},
			},
		},
		{
			name: "total value greater than 1",
			split: &domainSplit.Split{
				Targets: []domainSplit.SplitTarget{
					{ID: "target1", Value: 0.2},
					{ID: "target2", Value: 0.5},
					{ID: "target3", Value: 0.4},
					{ID: "target4", Value: 0.1},
				},
			},
			expected: []domainSplit.SplitTarget{
				{ID: "target1", Value: 0.25},
				{ID: "target2", Value: 0.25},
				{ID: "target3", Value: 0.25},
				{ID: "target4", Value: 0.25},
			},
		},
		{
			name: "total value less than 1",
			split: &domainSplit.Split{
				Targets: []domainSplit.SplitTarget{
					{ID: "target1", Value: 0.2},
					{ID: "target2", Value: 0},
					{ID: "target3", Value: 0.5},
				},
			},
			expected: []domainSplit.SplitTarget{
				{ID: "target1", Value: 0.2},
				{ID: "target2", Value: 0},
				{ID: "target3", Value: 0.5},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			calculatePercentageSplitValues(tc.split)
			assert.Equal(t, tc.expected, tc.split.Targets)
		})
	}
}

func TestService_ReplaceAttrIDsWithName(t *testing.T) {
	// Set up test data
	attributions := []*domainQuery.QueryRequestX{
		{ID: "attribution:abc", Key: "bruteforce"},
		{ID: "attribution:def", Key: "kraken"},
		{ID: "attribution:ghi", Key: "gigabright"},
	}

	split := &domainSplit.Split{
		Origin: "attribution:ghi",
		Targets: []domainSplit.SplitTarget{
			{ID: "attribution:abc", Value: 0.2},
			{ID: "attribution:def", Value: 0.3},
		},
	}

	// Call the function being tested
	replaceAttrIDsWithName(attributions, split)

	// Check the results
	expectedSplit := &domainSplit.Split{
		Origin: "gigabright",
		Targets: []domainSplit.SplitTarget{
			{ID: "bruteforce", Value: 0.2},
			{ID: "kraken", Value: 0.3},
		},
	}

	assert.Equal(t, expectedSplit, split, "replaceAttrIDsWithName() returned unexpected result")
}

func TestService_KeyValMapForSplitting(t *testing.T) {
	var split domainSplit.Split
	if err := testTools.ConvertJSONFileIntoStruct("testData", "split.json", &split); err != nil {
		t.Fatalf(convertJSONErrMsgFmt, err)
	}

	var resRows [][]bigquery.Value
	if err := testTools.ConvertJSONFileIntoStruct("testData", "result_rows.json", &resRows); err != nil {
		t.Fatalf(convertJSONErrMsgFmt, err)
	}

	var expectedMap map[string]map[string]int
	if err := testTools.ConvertJSONFileIntoStruct("testData", "key_index_map.json", &expectedMap); err != nil {
		t.Fatalf(convertJSONErrMsgFmt, err)
	}

	resultMap := keyValMapForSplitting(5, 0, &split, resRows, true)
	// Check that the result map has the expected keys and values
	assert.Equal(t, expectedMap, resultMap, "keyValMapForSplitting() returned unexpected result")
	// Check that the result map does not have the unallocated key since it is not used in the splitting
	_, ok := resultMap[consts.Unallocated]
	assert.Equal(t, false, ok, "keyValMapForSplitting() returned unexpected result")
	_, ok = resultMap["incorrect_key"]
	assert.Equal(t, false, ok, "keyValMapForSplitting() returned unexpected result")
}

func TestService_CalculateProportionalPercentageSplitValues(t *testing.T) {
	tests := []struct {
		name       string
		paramsFile string
		numRows    int
		numCols    int
		want       domainSplit.SplitTargetPerMetric
	}{
		{
			name:       "Proportional split between teams",
			paramsFile: "build_split_params3.json",
			numRows:    3,
			numCols:    2,
			want: domainSplit.SplitTargetPerMetric{
				"gigabright": []float64{
					0.9681420667499554,
					0.9884857790171446,
					0,
				},
				"turing": []float64{
					0.03185793325004462,
					0.011514220982855374,
					0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var splitParams domain.BuildSplit
			if err := testTools.ConvertJSONFileIntoStruct("testData", "build_split_params3.json", &splitParams); err != nil {
				t.Fatalf(convertJSONErrMsgFmt, err)
			}

			splits := *splitParams.SplitsReq
			split := splits[0]

			replaceAttrIDsWithName(splitParams.Attributions, &split)

			splitIndex := domainQuery.FindIndexInQueryRequestX(splitParams.RowsCols, split.ID)

			splitTargetsPerMetric := calculateProportionalPercentageSplitValues(
				split.Targets, splitParams, splitIndex)

			assert.Equal(t, test.want, splitTargetsPerMetric, "calculateProportionalPercentageSplitValues() returned unexpected result")
		})
	}
}

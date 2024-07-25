package query

import (
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

func TestFormatFilters(t *testing.T) {
	tests := []struct {
		attrFilters []string
		inverse     bool
		expected    string
	}{
		{[]string{"a = 1", "b = 2"}, false, "a = 1\n\t\tb = 2"},
		{[]string{"a = 1", "b = 2"}, true, "NOT a = 1\n\t\tb = 2"},
		{[]string{}, false, ""},
	}
	for _, test := range tests {
		result := formatFilters(test.attrFilters, test.inverse)
		if result != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, result)
		}
	}
}

func TestHandleFilter(t *testing.T) {
	values := []string{"/doit.com/Data Engineering/", "/doit.com/IT Systems/", "/2522914815/4576301977/"}
	expectedQueryParams := []bigquery.QueryParameter{{Name: "YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA", Value: values}}
	regexpString := "^[a-z]+$"
	invalidRegexString := "[a-z"
	tests := []struct {
		name               string
		params             buildFilterParams
		currentFilters     buildFilterResult
		expectedPredicates []string
		expectedParams     *[]bigquery.QueryParameter
		wantErr            bool
	}{
		{
			name: "valid filter with folder names",
			params: buildFilterParams{
				attrRow: &domainQuery.QueryRequestX{
					Type:   metadata.MetadataFieldTypeFixed,
					Field:  "T.project.ancestry_names",
					Values: &values,
				},
				id:    "attribution:uiKfGc78lZgY8mJnIAfK:project_ancestry_names:0",
				index: "0",
			},
			currentFilters: buildFilterResult{
				predicates:  &[]string{},
				queryParams: &[]bigquery.QueryParameter{},
			},
			expectedPredicates: []string{"T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA)"},
			expectedParams:     &expectedQueryParams,
		},
		{
			name: "valid filter with regexp value",
			params: buildFilterParams{
				attrRow: &domainQuery.QueryRequestX{
					Type:   metadata.MetadataFieldTypeFixed,
					Field:  "a",
					Regexp: &regexpString,
				},
				id:    "filter2",
				index: "0",
			},
			currentFilters: buildFilterResult{
				predicates:  &[]string{},
				queryParams: &[]bigquery.QueryParameter{},
			},
			expectedPredicates: []string{"IFNULL(REGEXP_CONTAINS(a, @ZmlsdGVyMjA), FALSE)"},
			expectedParams: &[]bigquery.QueryParameter{
				{
					Name:  "ZmlsdGVyMjA",
					Value: "^[a-z]+$",
				},
			},
		},
		{
			name: "invalid regexp value",
			params: buildFilterParams{
				attrRow: &domainQuery.QueryRequestX{
					Type:   metadata.MetadataFieldTypeFixed,
					Field:  "a",
					Regexp: &invalidRegexString,
				},
				id:    "filter3",
				index: "0",
			},
			currentFilters: buildFilterResult{
				predicates:  &[]string{},
				queryParams: &[]bigquery.QueryParameter{},
			},
			expectedPredicates: []string{},
			expectedParams:     &[]bigquery.QueryParameter{},
			wantErr:            true,
		},
	}

	for _, test := range tests {
		err := handleFilter(test.params, test.currentFilters)
		if test.wantErr {
			assert.Error(t, err)
		}

		assert.Equal(t, test.expectedPredicates, *test.currentFilters.predicates)
		assert.Equal(t, test.expectedParams, test.currentFilters.queryParams)
		assert.Equal(t, test.expectedPredicates, *test.currentFilters.predicates)
		assert.Equal(t, *test.expectedParams, *test.currentFilters.queryParams)
	}
}

func TestGetRowKeyReturnsCorrectValues(t *testing.T) {
	rowsArray := [][]bigquery.Value{
		{nil, "2020", "08", -0.000013414004, nil, 0},
		{nil, nil, false, -0.000013414004, nil, 0},
		{"", "5757", nil, 45, nil, 0},
		{"©", "¥", "¶", "Ë", nil, "Æ"},
		{`Ƣ`, "»", "Þ", "Ë", "Ħ", 0.44},
	}
	rowsNum := []int{3, 3, 3, 6, 5}

	keys := []string{"<nil>202008", "<nil><nil>false", "5757<nil>", "©¥¶Ë<nil>Æ", "Ƣ»ÞËĦ"}

	for i, row := range rowsArray {
		key, _ := GetRowKey(row, rowsNum[i])
		assert.NotEmpty(t, key)
		assert.NotNil(t, key)
		assert.Equal(t, keys[i], key)
	}
}

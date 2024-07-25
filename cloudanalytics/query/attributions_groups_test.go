package query

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/mocks"
	testTools "github.com/doitintl/hello/scheduled-tasks/common/test_tools"
)

const jsonToStructErrTmpl = "could not convert json test file into struct. error %s"

func TestQuery_HandleAttributionGroups(t *testing.T) {
	ctx := context.Background()
	attributionQuery := &mocks.IAttributionQuery{}

	q := &Query{
		attributionQuery: attributionQuery,
	}

	attributionQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(nil)
	attributionQuery.On("LogicalOperatorsAlphaToSymbol", "A AND B OR (C AND D) OR NOT E").Return("A && B || (C && D) || ! E")
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "A && B || (C && D) || ! E").Return("A AND B OR (C AND D) OR NOT E")
	attributionQuery.On("LogicalOperatorsAlphaToSymbol", "A AND B").Return("A && B").Times(6)
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "A = 1  && B").Return("A = 1  AND B").Times(6)
	attributionQuery.On("LogicalOperatorsAlphaToSymbol", "A = 1  AND B").Return("A = 1  && B").Times(6)
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "A = 1  && B = 2 ").Return("A = 1  AND B = 2 ").Times(6)
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA) && T.cloud_provider IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6Y2xvdWRfcHJvdmlkZXI6MTA)").Return("T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA) AND T.cloud_provider IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6Y2xvdWRfcHJvdmlkZXI6MTA)")
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.location.region IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6cmVnaW9uOjAx) && T.location.zone IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6em9uZToxMQ)").Return("T.location.region IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6cmVnaW9uOjAx) AND T.location.zone IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6em9uZToxMQ)")
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.service_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6c2VydmljZV9pZDowMg) && T.billing_account_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6YmlsbGluZ19hY2NvdW50X2lkOjEy)").Return("T.service_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6c2VydmljZV9pZDowMg) AND T.billing_account_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6YmlsbGluZ19hY2NvdW50X2lkOjEy)")
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA) && T.cloud_provider IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6Y2xvdWRfcHJvdmlkZXI6MTA)").Return("T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA) AND T.cloud_provider IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6Y2xvdWRfcHJvdmlkZXI6MTA)")
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.location.region IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6cmVnaW9uOjAx) && T.location.zone IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6em9uZToxMQ)").Return("T.location.region IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6cmVnaW9uOjAx) AND T.location.zone IN UNNEST(@YXR0cmlidXRpb246OEluVk45Vjh1SkhGN2lxZDdaMlk6em9uZToxMQ)")
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.service_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6c2VydmljZV9pZDowMg) && T.billing_account_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6YmlsbGluZ19hY2NvdW50X2lkOjEy)").Return("T.service_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6c2VydmljZV9pZDowMg) AND T.billing_account_id IN UNNEST(@YXR0cmlidXRpb246WVR5aVBQU3dXU1Y5T3liQTNYUkk6YmlsbGluZ19hY2NvdW50X2lkOjEy)")

	var attributionGroupQueryRequest1 domainQuery.AttributionGroupQueryRequest
	if err := testTools.ConvertJSONFileIntoStruct("testData", "attribution_group1.json", &attributionGroupQueryRequest1); err != nil {
		t.Fatalf(jsonToStructErrTmpl, err)
	}

	var attributionGroupQueryRequest2 domainQuery.AttributionGroupQueryRequest
	if err := testTools.ConvertJSONFileIntoStruct("testData", "attribution_group1.json", &attributionGroupQueryRequest2); err != nil {
		t.Fatalf(jsonToStructErrTmpl, err)
	}

	var expectedResult *domain.AttrFiltersParams
	if err := testTools.ConvertJSONFileIntoStruct("testData", "expected_res_handle_attribution_groups.json", &expectedResult); err != nil {
		t.Fatalf(jsonToStructErrTmpl, err)
	}

	var filtersParams domain.AttrFiltersParams

	if err := q.HandleAttributionGroups(ctx, &filtersParams, []*domainQuery.AttributionGroupQueryRequest{&attributionGroupQueryRequest1, &attributionGroupQueryRequest2}); err != nil {
		t.Fatalf("HandleAttributionGroups failed. error %s", err)
	}

	assert.Equal(t, expectedResult.AttrGroupsConditions, filtersParams.AttrGroupsConditions)
	assert.Equal(t, expectedResult.AttrRows, filtersParams.AttrRows)
	assert.Equal(t, expectedResult.CompositeFilters, filtersParams.CompositeFilters)
	assert.Equal(t, expectedResult.MetricFilters, filtersParams.MetricFilters)

	for i, queryParam := range filtersParams.QueryParams {
		assert.Equal(t, queryParam.Name, expectedResult.QueryParams[i].Name)
	}
}

func TestQuery_HandleAttributionGroups_AbsoluteKey(t *testing.T) {
	ctx := context.Background()
	attributionQuery := &mocks.IAttributionQuery{}

	q := &Query{
		attributionQuery: attributionQuery,
	}

	attributionQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(nil)
	attributionQuery.On("LogicalOperatorsAlphaToSymbol", "A AND B").Return("A && B").Times(1)
	attributionQuery.On("LogicalOperatorsSymbolToAlpha", "T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA) && T.cloud_provider IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6Y2xvdWRfcHJvdmlkZXI6MTA)").Return("T.project.ancestry_names IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6cHJvamVjdF9hbmNlc3RyeV9uYW1lczowMA) AND T.cloud_provider IN UNNEST(@YXR0cmlidXRpb246dWlLZkdjNzhsWmdZOG1KbklBZks6Y2xvdWRfcHJvdmlkZXI6MTA)")

	var attributionGroupQueryRequest1 domainQuery.AttributionGroupQueryRequest
	if err := testTools.ConvertJSONFileIntoStruct("testData", "attribution_group3.json", &attributionGroupQueryRequest1); err != nil {
		t.Fatalf(jsonToStructErrTmpl, err)
	}

	var expectedResult *domain.AttrFiltersParams
	if err := testTools.ConvertJSONFileIntoStruct("testData", "expected_res_handle_attribution_groups3.json", &expectedResult); err != nil {
		t.Fatalf(jsonToStructErrTmpl, err)
	}

	var filtersParams domain.AttrFiltersParams

	if err := q.HandleAttributionGroups(ctx, &filtersParams, []*domainQuery.AttributionGroupQueryRequest{&attributionGroupQueryRequest1}); err != nil {
		t.Fatalf("HandleAttributionGroups failed. error %s", err)
	}

	assert.Equal(t, expectedResult.AttrGroupsConditions, filtersParams.AttrGroupsConditions)
	assert.Equal(t, expectedResult.AttrRows, filtersParams.AttrRows)
	assert.Equal(t, expectedResult.CompositeFilters, filtersParams.CompositeFilters)
	assert.Equal(t, expectedResult.MetricFilters, filtersParams.MetricFilters)

	for i, queryParam := range filtersParams.QueryParams {
		assert.Equal(t, queryParam.Name, expectedResult.QueryParams[i].Name)
	}
}

func TestQuery_PrepareAttrGroupFilters(t *testing.T) {
	type PrepareAttrGroupFiltersParams struct {
		AttributionGroups []*domainQuery.AttributionGroupQueryRequest `json:"attributionGroups"`
		Filters           []*domainQuery.QueryRequestX                `json:"filters"`
		Rows              []*domainQuery.QueryRequestX                `json:"rows"`
		Cols              []*domainQuery.QueryRequestX                `json:"cols"`
	}

	var params PrepareAttrGroupFiltersParams
	if err := testTools.ConvertJSONFileIntoStruct("testData", "prep_data.json", &params); err != nil {
		t.Fatalf("could not convert json test file into struct. error %s", err)
	}

	q := &Query{}

	// Call the function for the first filter
	attrGroupFilters := q.PrepareAttrGroupFilters(params.AttributionGroups, params.Filters, params.Rows, params.Cols)
	// Check the output for the filter
	assert.Equal(t, attrGroupFilters, params.Filters)
	assert.Equal(t, "row_0", params.Filters[0].Field, "Field should be updated to 'row_0' for the first filter")
	assert.Equal(t, []string{"bruteforce", "gigabright"}, *params.Filters[0].Values, "Values should be updated to ['some_key'] for the first filter")
	// Call the function for a filter that is not for an attribution group
	params.Filters[0].Type = "non_attr_group_type"
	attrGroupFilters2 := q.PrepareAttrGroupFilters(params.AttributionGroups, params.Filters, params.Rows, params.Cols)
	// Check the output for an irrelevant filter, should not attempt to modify the filter
	// the values will be modified from the previous call
	assert.Nil(t, attrGroupFilters2)
}

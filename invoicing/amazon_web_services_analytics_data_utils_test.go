package invoicing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	testUtils "github.com/doitintl/tests"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
)

func Test_queryResultTransformer_Transform(t *testing.T) {
	type args struct {
		rows [][]bigquery.Value
	}

	tests := []struct {
		name                       string
		args                       args
		wantDaysToAccountToCostMap map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem
		wantAccounts               []string
		wantErr                    assert.ErrorAssertionFunc
	}{
		{
			name: "single row",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "usage", "0011111", 20.0, 999999, 3.0},
				},
			},
			wantDaysToAccountToCostMap: map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
				dateAsTime("2022-01-01"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 20.0, Savings: 3.0},
				},
			},
			wantAccounts: []string{"001404415847"},
			wantErr:      testUtils.AssertError(""),
		},
		{
			name: "single day two accounts",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "usage", "0011111", 20.0, 999999, 3.0},
					{"2022", "01", "01", "", "001356782165", "usage", "0011111", 10.0, 999999, 1.5},
				},
			},
			wantDaysToAccountToCostMap: map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
				dateAsTime("2022-01-01"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 20.0, Savings: 3.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 10.0, Savings: 1.5},
				},
			},
			wantAccounts: []string{"001404415847", "001356782165"},
			wantErr:      testUtils.AssertError(""),
		},
		{
			name: "two days single account",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "usage", "0011111", 10.0, 999999, 1.5},
					{"2022", "01", "02", "", "001404415847", "usage", "0011111", 20.0, 999999, 3.0},
				},
			},
			wantDaysToAccountToCostMap: map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
				dateAsTime("2022-01-01"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 10.0, Savings: 1.5},
				},
				dateAsTime("2022-01-02"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 20.0, Savings: 3.0},
				},
			},
			wantAccounts: []string{"001404415847"},
			wantErr:      testUtils.AssertError(""),
		},
		{
			name: "two days two accounts",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "usage", "0011111", 10.0, 999999, 1.5},
					{"2022", "01", "01", "", "001356782165", "usage", "0011111", 20.0, 999999, 3.0},

					{"2022", "01", "02", "", "001404415847", "FlexsaveComputeNegation", "0011111", -30.0, 999999, 0.0},
					{"2022", "01", "02", "", "001356782165", "usage", "0011111", 40.0, 999999, 6.0},
				},
			},
			wantDaysToAccountToCostMap: map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
				dateAsTime("2022-01-01"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 10.0, Savings: 1.5},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 20.0, Savings: 3.0},
				},
				dateAsTime("2022-01-02"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexsaveComputeNegation", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: -30.0, FlexsaveComputeNegations: -30.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "usage", Label: ""}:                   &pkg.CostAndSavingsAwsLineItem{Costs: 40.0, Savings: 6.0},
				},
			},
			wantAccounts: []string{"001404415847", "001356782165"},
			wantErr:      testUtils.AssertError(""),
		},
		{
			name: "two accounts aggregated costs and flexsave savings",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "FlexSaveCoveredUsage", "", "0011111", false, 12.0, 999999, 5.5},
					{"2022", "01", "01", "", "001404415847", "FlexsaveComputeNegation", "", "0011111", false, -5.5, 999999, 0.0},
					{"2022", "01", "01", "", "001404415847", "Usage", "", "0011111", false, 100.1, 999999, 0.0},
					{"2022", "01", "01", "", "001356782165", "Usage", "", "0011111", false, 123.45, 999999, 6.7},

					{"2022", "01", "02", "", "001404415847", "FlexsaveSagemakerNegation", "", "0011111", false, -98.7, 999999, 6.0},
					{"2022", "01", "02", "", "001356782165", "FlexsaveRDSManagementFee", "", "0011111", false, 4.0, 999999, 2.0},
					{"2022", "01", "02", "", "001356782165", "FlexSaveCoveredUsage", "", "0011111", false, 13.5, 999999, 2.0},
					{"2022", "01", "02", "", "001356782165", "XXXDiscount", "", "0011111", false, -1.2, 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
				dateAsTime("2022-01-01"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexSaveCoveredUsage", Label: ""}:    &pkg.CostAndSavingsAwsLineItem{Costs: 12.0, Savings: 5.5, FlexsaveComputeNegations: 0.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexsaveComputeNegation", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: -5.5, Savings: 0.0, FlexsaveComputeNegations: -5.5},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "Usage", Label: ""}:                   &pkg.CostAndSavingsAwsLineItem{Costs: 100.1, Savings: 0.0, FlexsaveComputeNegations: 0.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "Usage", Label: ""}:                   &pkg.CostAndSavingsAwsLineItem{Costs: 123.45, Savings: 6.7},
				},
				dateAsTime("2022-01-02"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexsaveSagemakerNegation", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: -98.7, Savings: 6.0, FlexsaveSagemakerNegations: -98.7},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "FlexsaveRDSManagementFee", Label: ""}:  &pkg.CostAndSavingsAwsLineItem{Costs: 4.0, Savings: 2.0, FlexsaveRDSNegations: -2.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "FlexSaveCoveredUsage", Label: ""}:      &pkg.CostAndSavingsAwsLineItem{Costs: 13.5, Savings: 2.0, FlexsaveComputeNegations: 0.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "XXXDiscount", Label: ""}:               &pkg.CostAndSavingsAwsLineItem{Costs: -1.2, Savings: 0.0, FlexsaveComputeNegations: 0.0},
				},
			},
			wantAccounts: []string{"001404415847", "001356782165"},
			wantErr:      nil,
		},
		{
			name: "two accounts marketplace costs",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "FlexSaveCoveredUsage", "", "0011111", false, 12.0, 999999, 5.5},
					{"2022", "01", "01", "", "001404415847", "FlexsaveComputeNegation", "", "0011111", false, -5.5, 999999, 0.0},
					{"2022", "01", "01", "", "001404415847", "Usage", "", "0011111", false, 100.1, 999999, 0.0},
					{"2022", "01", "01", "", "001356782165", "Usage", "", "0011111", false, 123.45, 999999, 6.7},
					{"2022", "01", "01", "", "001356782165", "Usage", "mangoDB", "0011111", true, 50.45, 999999, 6.7},

					{"2022", "01", "02", "", "001404415847", "FlexsaveSagemakerNegation", "", "0011111", false, -98.7, 999999, 6.0},
					{"2022", "01", "02", "", "001356782165", "FlexsaveRDSManagementFee", "", "0011111", false, 4.0, 999999, 2.0},
					{"2022", "01", "02", "", "001356782165", "FlexSaveCoveredUsage", "", "0011111", false, 13.5, 999999, 2.0},
					{"2022", "01", "02", "", "001356782165", "XXXDiscount", "", "0011111", false, -1.2, 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
				dateAsTime("2022-01-01"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexSaveCoveredUsage", Label: ""}:                                 &pkg.CostAndSavingsAwsLineItem{Costs: 12.0, Savings: 5.5, FlexsaveComputeNegations: 0.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexsaveComputeNegation", Label: ""}:                              &pkg.CostAndSavingsAwsLineItem{Costs: -5.5, Savings: 0.0, FlexsaveComputeNegations: -5.5},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "Usage", Label: ""}:                                                &pkg.CostAndSavingsAwsLineItem{Costs: 100.1, Savings: 0.0, FlexsaveComputeNegations: 0.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "Usage", Label: "", IsMarketplace: false}:                          &pkg.CostAndSavingsAwsLineItem{Costs: 123.45, Savings: 6.7},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "Usage", Label: "", MarketplaceSD: "mangoDB", IsMarketplace: true}: &pkg.CostAndSavingsAwsLineItem{Costs: 50.45, Savings: 6.7},
				},
				dateAsTime("2022-01-02"): {
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001404415847", PayerAccountID: "0011111", CostType: "FlexsaveSagemakerNegation", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: -98.7, Savings: 6.0, FlexsaveSagemakerNegations: -98.7},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "FlexsaveRDSManagementFee", Label: ""}:  &pkg.CostAndSavingsAwsLineItem{Costs: 4.0, Savings: 2.0, FlexsaveRDSNegations: -2.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "FlexSaveCoveredUsage", Label: ""}:      &pkg.CostAndSavingsAwsLineItem{Costs: 13.5, Savings: 2.0, FlexsaveComputeNegations: 0.0},
					pkg.CostAndSavingsAwsLineItemKey{AccountID: "001356782165", PayerAccountID: "0011111", CostType: "XXXDiscount", Label: ""}:               &pkg.CostAndSavingsAwsLineItem{Costs: -1.2, Savings: 0.0, FlexsaveComputeNegations: 0.0},
				},
			},
			wantAccounts: []string{"001404415847", "001356782165"},
			wantErr:      nil,
		},
		{
			name: "invalid year",
			args: args{
				rows: [][]bigquery.Value{
					{1, "01", "01", "", "001404415847", 10.0, 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: nil,
			wantAccounts:               nil,
			wantErr:                    testUtils.AssertError("unsupported field type for query result, value: 1"),
		},
		{
			name: "invalid month",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", 1, "01", "", "001404415847", 10.0, 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: nil,
			wantAccounts:               nil,
			wantErr:                    testUtils.AssertError("unsupported field type for query result, value: 1"),
		},
		{
			name: "invalid day",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", 1, "", "001404415847", "usage", "0011111", 10.0, 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: nil,
			wantAccounts:               nil,
			wantErr:                    testUtils.AssertError("unsupported field type for query result, value: 1"),
		},
		{
			name: "invalid accountID",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", 1, "usage", "0011111", 10.0, 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: nil,
			wantAccounts:               nil,
			wantErr:                    testUtils.AssertError("unsupported field type for query result, value: 1"),
		},
		{
			name: "invalid cost",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "usage", "", "0011111", false, "abc", 999999, 0.0},
				},
			},
			wantDaysToAccountToCostMap: nil,
			wantAccounts:               nil,
			wantErr:                    testUtils.AssertError("unexpected type cost in row[9], expected float64, actual abc"),
		},
		{
			name: "invalid flexsaveSavings",
			args: args{
				rows: [][]bigquery.Value{
					{"2022", "01", "01", "", "001404415847", "usage", "", "0011111", false, 10.0, 999999, "abc"},
				},
			},
			wantDaysToAccountToCostMap: nil,
			wantAccounts:               nil,
			wantErr:                    testUtils.AssertError("unexpected type savings in row[11], expected float64, actual abc"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &queryResultTransformer{}

			gotMap, gotAccounts, err := data.TransformToDaysToAccountsToCostAndAccountIDs(tt.args.rows)
			if tt.wantErr != nil && !tt.wantErr(t, err, fmt.Sprintf("TransformToDaysToAccountsToCostAndAccountIDs(%v)", tt.args.rows)) {
				return
			}

			if !reflect.DeepEqual(tt.wantDaysToAccountToCostMap, gotMap) {
				assert.Fail(t, "failed")
			}

			assert.True(t, reflect.DeepEqual(tt.wantDaysToAccountToCostMap, gotMap))
			assert.Equal(t, tt.wantAccounts, gotAccounts)
		})
	}
}

func Test_billingDataQueryBuilder_GetBillingDataQuery(t *testing.T) {
	expectedRequest := cloudanalytics.QueryRequest{}

	jsonPayload, err := os.ReadFile("testdata/getBillingDataQuery_expected_request.json")
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(jsonPayload, &expectedRequest)
	if err != nil {
		t.Fatal(err)
	}

	// Needed because the json tag is "-" for the origin.
	expectedRequest.Origin = "other"

	ctx := context.Background()

	builder := &BillingDataQueryBuilder{}

	got, err := builder.GetBillingDataQuery(ctx, dateAsTime("2022-01-01"), []string{"31325"}, "amazon-web-services")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expectedRequest, *got)
}

func dateAsTime(date string) time.Time {
	parse, err := time.Parse("2006-01-02", date)
	if err != nil {
		panic(err)
	}

	return parse
}

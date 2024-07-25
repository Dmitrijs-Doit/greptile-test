package service

import (
	"reflect"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func Test_getSortedCategoriesAtoZ(t *testing.T) {
	type args struct {
		categoryKeys [][]string
		requestCols  []*domainQuery.QueryRequestX
	}

	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "Simple sort single col",
			args: args{
				categoryKeys: [][]string{
					{"Source Repository"},
					{"Kubernetes Engine"},
					{"Artifact Registry"},
					{"Cloud Run"},
					{"App Engine"},
					{"BigQuery"},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "fixed",
						Position: "col",
						ID:       "fixed:service_description",
						Field:    "T.service_description",
						Key:      "service_description",
						Label:    "Service",
					},
				},
			},
			want: [][]string{
				{"App Engine"}, {"Artifact Registry"}, {"BigQuery"}, {"Cloud Run"}, {"Kubernetes Engine"}, {"Source Repository"},
			},
		},
		{
			name: "Simple sort single col weekday",
			args: args{
				categoryKeys: [][]string{
					{"Friday"},
					{"Monday"},
					{"Saturday"},
					{"Sunday"},
					{"Thursday"},
					{"Tuesday"},
					{"Wednesday"},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "datetime",
						Position: "col",
						ID:       "datetime:week_day",
						Field:    "T.usage_start_time",
						Key:      "week_day",
						Label:    "Weekday",
					},
				},
			},
			want: [][]string{
				{"Monday"}, {"Tuesday"}, {"Wednesday"}, {"Thursday"}, {"Friday"}, {"Saturday"}, {"Sunday"},
			},
		},
		{
			name: "Simple sort two cols with weekday",
			args: args{
				categoryKeys: [][]string{
					{"A", "Friday"},
					{"B", "Friday"},
					{"B", "Monday"},
					{"A", "Monday"},
					{"A", "Thursday"},
					{"B", "Sunday"},
					{"A", "Tuesday"},
					{"B", "Saturday"},
					{"B", "Thursday"},
					{"A", "Saturday"},
					{"A", "Sunday"},
					{"B", "Tuesday"},
					{"A", "Wednesday"},
					{"B", "Wednesday"},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "fixed",
						Position: "col",
						ID:       "fixed:billing_account_id",
						Field:    "T.billing_account_id",
						Key:      "billing_account_id",
						Label:    "Account",
					},
					{
						Type:     "datetime",
						Position: "col",
						ID:       "datetime:week_day",
						Field:    "T.usage_start_time",
						Key:      "week_day",
						Label:    "Weekday",
					},
					{Type: "datetime", Position: "col", ID: "datetime:week_day", Field: "T.usage_start_time", Key: "week_day", Label: "Weekday", IncludeInFilter: false, AllowNull: false, Inverse: false, Regexp: (*string)(nil), Values: (*[]string)(nil), Composite: []*domainQuery.QueryRequestX(nil)},
				},
			},
			want: [][]string{
				{"A", "Monday"}, {"A", "Tuesday"}, {"A", "Wednesday"}, {"A", "Thursday"}, {"A", "Friday"}, {"A", "Saturday"}, {"A", "Sunday"},
				{"B", "Monday"}, {"B", "Tuesday"}, {"B", "Wednesday"}, {"B", "Thursday"}, {"B", "Friday"}, {"B", "Saturday"}, {"B", "Sunday"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortCategoryKeysAtoZ(tt.args.categoryKeys, tt.args.requestCols)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortCategoryKeysAtoZ() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSortedCategories(t *testing.T) {
	type args struct {
		requestRows    int
		requestCols    []*domainQuery.QueryRequestX
		requestMetric  report.Metric
		resultRows     [][]bigquery.Value
		reportColOrder string
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Error for non-string value",
			args: args{
				requestRows: 0,
				resultRows: [][]bigquery.Value{
					{1, 0, 0.004090687820327373, 0},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "fixed",
						Position: "col",
						ID:       "fixed:service_description",
						Field:    "T.service_description",
						Key:      "service_description",
						Label:    "Service",
					},
				},
				reportColOrder: "a_to_z",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Simple sort single col",
			args: args{
				requestRows: 0,
				resultRows: [][]bigquery.Value{
					{"Source Repository", 0, 0.004090687820327373, 0},
					{"Kubernetes Engine", 143.86361010000005, 1438.6470000000004, 0},
					{"Artifact Registry", 1.50027885, 20.534938346599308, 0},
					{"Cloud Run", 0.0008449500000000005, 964.1898482274096, 0},
					{"App Engine", 139.78606530000002, 517178.14109050843, 0},
					{"BigQuery", 34714.08032624143, 6.67151304241738e+06, 0},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "fixed",
						Position: "col",
						ID:       "fixed:service_description",
						Field:    "T.service_description",
						Key:      "service_description",
						Label:    "Service",
					},
				},
				reportColOrder: "a_to_z",
			},
			want: []string{
				"App Engine", "Artifact Registry", "BigQuery", "Cloud Run", "Kubernetes Engine", "Source Repository",
			},
			wantErr: false,
		},
		{
			name: "Simple sort single col weekday",
			args: args{
				requestRows: 0,
				resultRows: [][]bigquery.Value{
					{"Friday", 5709.690252503618, 6.3489853945621386e+07, 0},
					{"Monday", 14284.011722335155, 1.819409541368732e+08, 0},
					{"Saturday", 5717.732391607466, 6.454168707605365e+07, 0},
					{"Sunday", 6402.486734214798, 7.722968274189985e+07, 0},
					{"Thursday", 6520.488162569263, 7.867241505347583e+07, 0},
					{"Tuesday", 13964.558448860453, 1.7868157848608494e+08, 0},
					{"Wednesday", 10052.631318350555, 1.3026489409614535e+08, 0},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "datetime",
						Position: "col",
						ID:       "datetime:week_day",
						Field:    "T.usage_start_time",
						Key:      "week_day",
						Label:    "Weekday",
					},
				},
				reportColOrder: "a_to_z",
			},
			want: []string{
				"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday",
			},
			wantErr: false,
		},
		{
			name: "Simple sort two cols with weekday",
			args: args{
				requestRows: 0,
				resultRows: [][]bigquery.Value{
					{"A", "Friday", 5709.690252503618, 6.3489853945621386e+07, 0},
					{"B", "Friday", 5709.690252503618, 6.3489853945621386e+07, 0},
					{"B", "Monday", 14284.011722335155, 1.819409541368732e+08, 0},
					{"A", "Monday", 14284.011722335155, 1.819409541368732e+08, 0},
					{"A", "Thursday", 6520.488162569263, 7.867241505347583e+07, 0},
					{"B", "Sunday", 6402.486734214798, 7.722968274189985e+07, 0},
					{"A", "Tuesday", 13964.558448860453, 1.7868157848608494e+08, 0},
					{"B", "Saturday", 5717.732391607466, 6.454168707605365e+07, 0},
					{"B", "Thursday", 6520.488162569263, 7.867241505347583e+07, 0},
					{"A", "Saturday", 5717.732391607466, 6.454168707605365e+07, 0},
					{"A", "Sunday", 6402.486734214798, 7.722968274189985e+07, 0},
					{"B", "Tuesday", 13964.558448860453, 1.7868157848608494e+08, 0},
					{"A", "Wednesday", 10052.631318350555, 1.3026489409614535e+08, 0},
					{"B", "Wednesday", 10052.631318350555, 1.3026489409614535e+08, 0},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{
						Type:     "fixed",
						Position: "col",
						ID:       "fixed:billing_account_id",
						Field:    "T.billing_account_id",
						Key:      "billing_account_id",
						Label:    "Account",
					},
					{
						Type:     "datetime",
						Position: "col",
						ID:       "datetime:week_day",
						Field:    "T.usage_start_time",
						Key:      "week_day",
						Label:    "Weekday",
					},
				},
				reportColOrder: "a_to_z",
			},
			want: []string{
				"A-Monday", "A-Tuesday", "A-Wednesday", "A-Thursday", "A-Friday", "A-Saturday", "A-Sunday",
				"B-Monday", "B-Tuesday", "B-Wednesday", "B-Thursday", "B-Friday", "B-Saturday", "B-Sunday",
			},
			wantErr: false,
		},
		{
			name: "Row with first col groupBy,  sort two cols with weekday",
			args: args{
				requestRows: 1,
				resultRows: [][]bigquery.Value{
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Friday", 4280.5552213536175, 4.395163468758442e+07, 0},
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Monday", 11162.850724535221, 1.4032351381853685e+08, 0},
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Saturday", 4379.781961057469, 4.519501912593204e+07, 0},
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Sunday", 5001.161740964787, 5.703397254034938e+07, 0},
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Thursday", 5030.224027719269, 5.807468971923019e+07, 0},
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Tuesday", 10892.941252460454, 1.3694393364637095e+08, 0},
					{bigquery.Value(nil), "006C3F-3613C3-8C2169", "Wednesday", 9073.275710021084, 1.1406504100861637e+08, 0},
					{"asia-east1", "006C3F-3613C3-8C2169", "Monday", 5.9227875, 2632.3499999999995, 0},
					{"asia-east1", "006C3F-3613C3-8C2169", "Sunday", 1.8262125, 811.65, 0},
					{"asia-east1", "006C3F-3613C3-8C2169", "Tuesday", 1.063125, 472.5, 0},
					{"asia-east1", "006C3F-3613C3-8C2169", "Wednesday", 0, 0, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Friday", 21.30512965, 4729.652156212645, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Monday", 49.77869504999998, 17884.57813167399, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Saturday", 21.264161199999997, 6118.21435801479, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Sunday", 23.69254565, 8899.014050095711, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Thursday", 26.356697700000005, 10040.207814135629, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Tuesday", 50.63843784999999, 15416.08964358642, 0},
					{"asia-east2", "006C3F-3613C3-8C2169", "Wednesday", 44.00073179999998, 10419.487524440126, 0},
					{"europe-north1", "006C3F-3613C3-8C2169", "Friday", 59.22372539999998, 39270.348928192936, 0},
					{"europe-north1", "006C3F-3613C3-8C2169", "Monday", 168.9640008000001, 112218.39785638583, 0},
					{"europe-north1", "006C3F-3613C3-8C2169", "Saturday", 39.966201449999986, 26431.998928192937, 0},
					{"europe-north1", "006C3F-3613C3-8C2169", "Sunday", 61.14207539999998, 40549.24892819293, 0},
					{"europe-north1", "006C3F-3613C3-8C2169", "Thursday", 95.16207645000001, 63229.24892819293, 0},
					{"europe-north1", "006C3F-3613C3-8C2169", "Tuesday", 136.47805080000003, 90561.0978563859, 0},
				},
				requestCols: []*domainQuery.QueryRequestX{
					{Type: "fixed", Position: "col", ID: "fixed:billing_account_id", Field: "T.billing_account_id", Key: "billing_account_id", Label: "Account"},
					{Type: "datetime", Position: "col", ID: "datetime:week_day", Field: "T.usage_start_time", Key: "week_day", Label: "Weekday"},
				},
				reportColOrder: "a_to_z",
			},
			want:    []string{"006C3F-3613C3-8C2169-Monday", "006C3F-3613C3-8C2169-Tuesday", "006C3F-3613C3-8C2169-Wednesday", "006C3F-3613C3-8C2169-Thursday", "006C3F-3613C3-8C2169-Friday", "006C3F-3613C3-8C2169-Saturday", "006C3F-3613C3-8C2169-Sunday"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSortedCategories(tt.args.requestRows, tt.args.requestCols, tt.args.requestMetric, tt.args.resultRows, tt.args.reportColOrder)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSortedCategories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSortedCategories() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldReverseYAxis(t *testing.T) {
	series1 := HighchartsDataSeries{
		Data: []float64{-0.3, -5, -7},
	}
	series2 := HighchartsDataSeries{
		Data: []float64{0.3, 6, 7},
	}
	positiveDataSeries := []*HighchartsDataSeries{&series1, &series2}

	reversed, err := shouldReverseYAxis(positiveDataSeries)
	if err != nil {
		t.Error("error shouldReverseYAxis")
	}

	assert.Equal(t, reversed, false)

	negativeDateSeries := []*HighchartsDataSeries{&series1, &series1}

	reversed, err = shouldReverseYAxis(negativeDateSeries)
	if err != nil {
		t.Error("error shouldReverseYAxis")
	}

	assert.Equal(t, reversed, true)

	emptyDateSeries := []*HighchartsDataSeries{}

	reversed, err = shouldReverseYAxis(emptyDateSeries)
	if err != nil {
		t.Error("error shouldReverseYAxis")
	}

	assert.Equal(t, reversed, false)
}

func Test_sortDataSeries(t *testing.T) {
	series1 := HighchartsDataSeries{
		Data: []float64{-0.3, -5, -7},
		Name: "Negatives",
	}
	series2 := HighchartsDataSeries{
		Data: []float64{0.4, 6, 7},
		Name: "Positives",
	}
	series3 := HighchartsDataSeries{
		Data: []float64{0.3, 5, -7},
		Name: "Mixed",
	}

	dataSeries := []*HighchartsDataSeries{&series1, &series2, &series3}
	sortedAscending := sortDataSeries(dataSeries, true)
	assert.Equal(t, sortedAscending[0].Name, "Negatives")
	assert.Equal(t, sortedAscending[1].Name, "Mixed")
	assert.Equal(t, sortedAscending[2].Name, "Positives")

	sortedDescending := sortDataSeries(dataSeries, false)
	assert.Equal(t, sortedDescending[0].Name, "Positives")
	assert.Equal(t, sortedDescending[1].Name, "Mixed")
	assert.Equal(t, sortedDescending[2].Name, "Negatives")

	var emptyDataSeries []*HighchartsDataSeries
	sortedEmpty := sortDataSeries(emptyDataSeries, true)
	assert.Equal(t, len(sortedEmpty), 0)
}

package executor

import (
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

var mockTime = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)

func TestTransformCostFromTableTypes(t *testing.T) {
	var (
		clusterTotalTB       = bigquery.NullFloat64{Float64: 1.0}
		clusterTotalTB2      = bigquery.NullFloat64{Float64: 2.0}
		noPartitionTotalTB   = bigquery.NullFloat64{Float64: 2.0}
		withPartitionTotalTB = bigquery.NullFloat64{Float64: 0.0}

		tableName1 = bigquery.NullString{StringVal: "table1", Valid: true}
		tableName2 = bigquery.NullString{StringVal: "table2", Valid: true}
		tableName3 = bigquery.NullString{StringVal: "table3", Valid: true}
		tableName4 = bigquery.NullString{StringVal: "table4", Valid: true}

		data = []bqmodels.CostFromTableTypesResult{
			{TableType: "clustered", TableName: tableName1, TotalTB: clusterTotalTB},
			{TableType: "noPartition", TableName: tableName2, TotalTB: noPartitionTotalTB},
		}

		expectedPercentageClustered   = 33.33
		expectedPercentageNoPartition = 66.67

		duplicateData = []bqmodels.CostFromTableTypesResult{
			{TableType: "clustered", TableName: tableName1, TotalTB: clusterTotalTB},
			{TableType: "noPartition", TableName: tableName2, TotalTB: noPartitionTotalTB},
			{TableType: "clustered", TableName: tableName3, TotalTB: clusterTotalTB2},
			{TableType: "withPartition", TableName: tableName4, TotalTB: withPartitionTotalTB},
		}
		duplicateExpectedPercentageClustered     = 60.0
		duplicateExpectedPercentageNoPartition   = 40.0
		duplicateExpectedPercentageWithPartition = 0.0
	)

	type args struct {
		timeRange bqmodels.TimeRange
		data      []bqmodels.CostFromTableTypesResult
	}

	tests := []struct {
		name    string
		args    args
		want    dal.RecommendationSummary
		wantErr error
	}{
		{
			name: "transformation with no duplicates",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data:      data,
			},
			want: dal.RecommendationSummary{
				bqmodels.CostFromTableTypes: dal.TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.CostFromTableTypeDocument{
						Data: map[string]fsModels.CostFromTableType{
							"clustered": {
								Tables:     []fsModels.TableDetail{{TableName: &tableName1.StringVal, Value: 1.0}},
								TB:         1.0,
								Percentage: expectedPercentageClustered,
							},
							"noPartition": {
								Tables:     []fsModels.TableDetail{{TableName: &tableName2.StringVal, Value: 2.0}},
								TB:         2.0,
								Percentage: expectedPercentageNoPartition,
							},
						},
						LastUpdate: mockTime,
					},
				},
			},
		},
		{
			name: "transformation with duplicates and zero values",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data:      duplicateData,
			},
			want: dal.RecommendationSummary{
				bqmodels.CostFromTableTypes: dal.TimeRangeRecommendation{
					bqmodels.TimeRangeMonth: fsModels.CostFromTableTypeDocument{
						Data: map[string]fsModels.CostFromTableType{
							"clustered": {
								Tables: []fsModels.TableDetail{
									{TableName: &tableName1.StringVal, Value: 1.0},
									{TableName: &tableName3.StringVal, Value: 2.0},
								},
								TB:         3.0,
								Percentage: duplicateExpectedPercentageClustered,
							},
							"noPartition": {
								Tables:     []fsModels.TableDetail{{TableName: &tableName2.StringVal, Value: 2.0}},
								TB:         2.0,
								Percentage: duplicateExpectedPercentageNoPartition,
							},
							"withPartition": {
								Tables:     []fsModels.TableDetail{{TableName: &tableName4.StringVal, Value: 0.0}},
								TB:         0.0,
								Percentage: duplicateExpectedPercentageWithPartition,
							},
						},
						LastUpdate: mockTime,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, TransformCostFromTableTypes(tt.args.timeRange, tt.args.data, mockTime))
		})
	}
}

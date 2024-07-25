package executor

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformODataset(t *testing.T) {
	var (
		mockNowTime = time.Date(2024, 5, 18, 0, 0, 0, 0, time.UTC)
		discount    = 1.0
		isValid     = true

		mockBilllingProjectID = "billing_project_id"
		mockDatasetID         = "dataset_id"
		mockProjectID         = "project_id"
		mockTableID           = "table_id"
		mockUserEmail         = "user_email"
		mockJobID             = "job_id"
		mockLocation          = "location"
	)

	type args struct {
		timeRange        bqmodels.TimeRange
		customerDiscount float64
		data             *bqmodels.RunODDatasetResult
		now              time.Time
	}

	tests := []struct {
		name    string
		args    args
		want    dal.RecommendationSummary
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "nil input",
			args: args{
				timeRange:        bqmodels.TimeRangeMonth,
				customerDiscount: discount,
				data:             nil,
				now:              mockNowTime,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "empty input",
			args: args{
				timeRange:        bqmodels.TimeRangeMonth,
				customerDiscount: discount,
				data:             &bqmodels.RunODDatasetResult{},
				now:              mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.DatasetScanPrice: {bqmodels.TimeRangeMonth: fsModels.DatasetScanPriceDocument{}},
				bqmodels.DatasetScanTB:    {bqmodels.TimeRangeMonth: fsModels.DatasetScanTBDocument{}},
			},
			wantErr: assert.NoError,
		},
		{
			name: "with data",
			args: args{
				timeRange:        bqmodels.TimeRangeMonth,
				customerDiscount: discount,
				data: &bqmodels.RunODDatasetResult{
					Dataset: []bqmodels.DatasetResult{
						{
							ProjectID: mockProjectID,
							DatasetID: mockDatasetID,
							ScanTB:    bigquery.NullFloat64{Float64: 1, Valid: isValid},
						},
					},
					DatasetTopQueries: []bqmodels.DatasetTopQueriesResult{
						{
							ProjectID:     mockProjectID,
							DatasetID:     mockDatasetID,
							DatasetFullID: "",
							TopQueriesResult: bqmodels.TopQueriesResult{
								BillingProjectID:      mockBilllingProjectID,
								UserID:                mockUserEmail,
								JobID:                 mockJobID,
								Location:              mockLocation,
								ExecutedQueries:       1,
								AvgExecutionTimeSec:   bigquery.NullFloat64{Float64: 1, Valid: isValid},
								TotalExecutionTimeSec: bigquery.NullFloat64{Float64: 1, Valid: isValid},
								AvgSlots:              bigquery.NullFloat64{Float64: 1, Valid: isValid},
								AvgScanTB:             bigquery.NullFloat64{Float64: 1, Valid: isValid},
								TotalScanTB:           bigquery.NullFloat64{Float64: 1, Valid: isValid},
							},
						},
					},
					DatasetTopUsers: []bqmodels.DatasetTopUsersResult{
						{
							ProjectID:     mockProjectID,
							DatasetID:     mockDatasetID,
							DatasetFullID: "",
							UserEmail:     mockUserEmail,
							ScanTB:        bigquery.NullFloat64{Float64: 1, Valid: isValid},
						},
					},
					DatasetTopTables: []bqmodels.DatasetTopTablesResult{
						{
							DatasetFullID: "",
							DatasetID:     mockDatasetID,
							TableID:       mockTableID,
							ScanTB:        bigquery.NullFloat64{Float64: 1, Valid: isValid},
						},
					},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.DatasetScanPrice: {bqmodels.TimeRangeMonth: fsModels.DatasetScanPriceDocument{
					mockDatasetID: fsModels.DatasetScanPrice{
						ProjectID: mockProjectID,
						DatasetID: mockDatasetID,
						ScanPrice: PricePerTBScan,
						TopQuery: map[string]fsModels.DatasetTopQueryPrice{
							mockJobID: {
								AvgScanPrice:   PricePerTBScan,
								DatasetID:      mockDatasetID,
								Location:       mockLocation,
								ProjectID:      mockProjectID,
								TotalScanPrice: PricePerTBScan,
								UserID:         mockUserEmail,
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1,
									AvgSlots:              1,
									ExecutedQueries:       1,
									TotalExecutionTimeSec: 1,
									BillingProjectID:      mockBilllingProjectID,
								},
							},
						},
						TopUsers: map[string]float64{
							mockUserEmail: PricePerTBScan,
						},
						TopTable: map[string]float64{
							mockTableID: PricePerTBScan,
						},
						LastUpdate: mockNowTime,
					},
				}},
				bqmodels.DatasetScanTB: {bqmodels.TimeRangeMonth: fsModels.DatasetScanTBDocument{
					mockDatasetID: fsModels.DatasetScanTB{
						ProjectID: mockProjectID,
						DatasetID: mockDatasetID,
						ScanTB:    1,
						TopQuery: map[string]fsModels.DatasetTopQueryTB{
							mockJobID: {
								AvgScanTB:   1,
								DatasetID:   mockDatasetID,
								Location:    mockLocation,
								ProjectID:   mockProjectID,
								TotalScanTB: 1,
								UserID:      mockUserEmail,
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1,
									AvgSlots:              1,
									ExecutedQueries:       1,
									TotalExecutionTimeSec: 1,
									BillingProjectID:      mockBilllingProjectID,
								},
							},
						},
						TopUsers: map[string]float64{
							mockUserEmail: 1,
						},
						TopTable: map[string]float64{
							mockTableID: 1,
						},
						LastUpdate: mockNowTime,
					},
				}},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformODataset(tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)
			if !tt.wantErr(t, err, fmt.Sprintf("TransformODataset() error = %v, wantErr %v", err, tt.wantErr)) {
				t.Errorf("TransformODataset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equalf(t, tt.want, got, "TransformODataset() = %v, want %v", got, tt.want)

		})
	}
}

package executor

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformODUser1(t *testing.T) {
	var (
		mockNowTime = time.Date(2024, 5, 18, 0, 0, 0, 0, time.UTC)
		discount    = 1.0
		isValid     = true
		userID      = "user1"

		mockUser = []bqmodels.UserResult{
			{
				UserID: userID,
				ScanTB: bqNullFloat(123.45, isValid),
			},
		}

		mockQueries = []bqmodels.TopQueriesResult{
			{
				UserID:                userID,
				JobID:                 "job1",
				AvgScanTB:             bqNullFloat(12.34, isValid),
				Location:              "location1",
				TotalScanTB:           bqNullFloat(123.45, isValid),
				AvgExecutionTimeSec:   bqNullFloat(1.23, isValid),
				AvgSlots:              bqNullFloat(1.2, isValid),
				ExecutedQueries:       10,
				TotalExecutionTimeSec: bqNullFloat(12.3, isValid),
				BillingProjectID:      "project1",
			},
		}

		mockProjects = []bqmodels.UserTopProjectsResult{
			{
				UserID:    userID,
				ProjectID: "project1",
				ScanTB:    bqNullFloat(123.45, isValid),
			},
		}

		mockDatasets = []bqmodels.UserTopDatasetsResult{
			{
				UserID:    userID,
				DatasetID: "dataset1",
				ScanTB:    bqNullFloat(123.45, isValid),
			},
		}

		mockTables = []bqmodels.UserTopTablesResult{
			{
				UserID:  userID,
				TableID: "table1",
				ScanTB:  bqNullFloat(123.45, isValid),
			},
		}
	)

	type args struct {
		timeRange        bqmodels.TimeRange
		customerDiscount float64
		data             *bqmodels.RunODUserResult
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
				data:             &bqmodels.RunODUserResult{},
				now:              mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.UserScanPrice: {bqmodels.TimeRangeMonth: fsModels.UserScanPriceDocument{}},
				bqmodels.UserScanTB:    {bqmodels.TimeRangeMonth: fsModels.UserScanTBDocument{}},
			},
			wantErr: assert.NoError,
		},
		{
			name: "full input",
			args: args{
				timeRange:        bqmodels.TimeRangeMonth,
				customerDiscount: discount,
				data: &bqmodels.RunODUserResult{
					User:    mockUser,
					Queries: mockQueries,
					Project: mockProjects,
					Dataset: mockDatasets,
					Table:   mockTables,
				},
				now: mockNowTime,
			},
			wantErr: assert.NoError,
			want: dal.RecommendationSummary{
				bqmodels.UserScanPrice: {bqmodels.TimeRangeMonth: fsModels.UserScanPriceDocument{
					userID: fsModels.UserScanPrice{
						UserID:     userID,
						ScanPrice:  getScanPrice(discount, 123.45),
						LastUpdate: mockNowTime,
						TopQuery: map[string]fsModels.UserTopQueryPrice{
							"job1": {
								AvgScanPrice:   getScanPrice(discount, 12.34),
								Location:       "location1",
								TotalScanPrice: getScanPrice(discount, 123.45),
								UserID:         userID,
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1.23,
									AvgSlots:              1.2,
									ExecutedQueries:       10,
									TotalExecutionTimeSec: 12.3,
									BillingProjectID:      "project1",
								},
							},
						},
						TopProject: map[string]float64{
							"project1": getScanPrice(discount, 123.45),
						},
						TopDataset: map[string]float64{
							"dataset1": getScanPrice(discount, 123.45),
						},
						TopTable: map[string]float64{
							"table1": getScanPrice(discount, 123.45),
						},
					},
				}},
				bqmodels.UserScanTB: {bqmodels.TimeRangeMonth: fsModels.UserScanTBDocument{
					userID: fsModels.UserScanTB{
						UserID:     userID,
						ScanTB:     123.45,
						LastUpdate: mockNowTime,
						TopQuery: map[string]fsModels.UserTopQueryTB{
							"job1": {
								AvgScanTB:   12.34,
								Location:    "location1",
								TotalScanTB: 123.45,
								UserID:      userID,
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1.23,
									AvgSlots:              1.2,
									ExecutedQueries:       10,
									TotalExecutionTimeSec: 12.3,
									BillingProjectID:      "project1",
								},
							},
						},
						TopProject: map[string]float64{
							"project1": 123.45,
						},
						TopDataset: map[string]float64{
							"dataset1": 123.45,
						},
						TopTable: map[string]float64{
							"table1": 123.45,
						},
					},
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformODUser(tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)
			if !tt.wantErr(t, err, fmt.Sprintf("TransformODUser(%v, %v, %v, %v)", tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)) {
				return
			}

			assert.Equalf(t, tt.want, got, "TransformODUser(%v, %v, %v, %v)", tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)
		})
	}
}

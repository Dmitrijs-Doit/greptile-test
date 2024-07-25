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

func TestTransformODBillingProject(t *testing.T) {
	var (
		mockNowTime = time.Date(2024, 5, 18, 0, 0, 0, 0, time.UTC)
		discount    = 1.0
		isValid     = true
	)

	type args struct {
		timeRange        bqmodels.TimeRange
		customerDiscount float64
		data             *bqmodels.RunODBillingProjectResult
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
				customerDiscount: 1.0,
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
				customerDiscount: 1.0,
				data:             &bqmodels.RunODBillingProjectResult{},
				now:              mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.BillingProjectScanPrice: {bqmodels.TimeRangeMonth: fsModels.BillingProjectScanPriceDocument{}},
				bqmodels.BillingProjectScanTB:    {bqmodels.TimeRangeMonth: fsModels.BillingProjectScanTBDocument{}},
			},
			wantErr: assert.NoError,
		},
		{
			name: "full input",
			args: args{
				timeRange:        bqmodels.TimeRangeMonth,
				customerDiscount: 1.0,
				data: &bqmodels.RunODBillingProjectResult{
					BillingProject: []bqmodels.BillingProjectResult{
						{BillingProjectID: "project1", ScanTB: bqNullFloat(100.0, isValid)},
					},
					TopQueries: []bqmodels.TopQueriesResult{
						{
							BillingProjectID:      "project1",
							JobID:                 "job1",
							AvgScanTB:             bqNullFloat(10.0, isValid),
							TotalScanTB:           bqNullFloat(100.0, isValid),
							UserID:                "user1",
							Location:              "us-central1",
							AvgExecutionTimeSec:   bqNullFloat(1.5, isValid),
							AvgSlots:              bqNullFloat(5.0, isValid),
							ExecutedQueries:       10,
							TotalExecutionTimeSec: bqNullFloat(15.0, isValid),
						},
					},
					TopUsers: []bqmodels.BillingProjectTopUsersResult{
						{BillingProjectID: "project1", UserEmail: "user1@example.com", ScanTB: bqNullFloat(50.0, isValid)},
					},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.BillingProjectScanPrice: {bqmodels.TimeRangeMonth: fsModels.BillingProjectScanPriceDocument{
					"project1": {
						BillingProjectID: "project1",
						ScanPrice:        getScanPrice(discount, 100.0),
						LastUpdate:       mockNowTime,
						TopQueries: map[string]fsModels.BillingProjectTopQueryPrice{
							"job1": {
								AvgScanPrice:   getScanPrice(discount, 10.0),
								Location:       "us-central1",
								TotalScanPrice: getScanPrice(discount, 100.0),
								UserID:         "user1",
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1.5,
									AvgSlots:              5,
									ExecutedQueries:       10,
									TotalExecutionTimeSec: 15.0,
									BillingProjectID:      "project1",
								},
							},
						},
						TopUsers: map[string]float64{
							"user1@example.com": getScanPrice(discount, 50.0),
						},
					},
				}},
				bqmodels.BillingProjectScanTB: {bqmodels.TimeRangeMonth: fsModels.BillingProjectScanTBDocument{
					"project1": {
						BillingProjectID: "project1",
						ScanTB:           100.0,
						LastUpdate:       mockNowTime,
						TopQueries: map[string]fsModels.BillingProjectTopQueryTB{
							"job1": {
								AvgScanTB:   10.0,
								Location:    "us-central1",
								TotalScanTB: 100.0,
								UserID:      "user1",
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1.5,
									AvgSlots:              5,
									ExecutedQueries:       10,
									TotalExecutionTimeSec: 15.0,
									BillingProjectID:      "project1",
								},
							},
						},
						TopUsers: map[string]float64{
							"user1@example.com": 50.0,
						},
					},
				}},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformODBillingProject(tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)
			if !tt.wantErr(t, err, fmt.Sprintf("TransformOnDemandSlotsExplorer(%v, %v, %v)", tt.args.timeRange, tt.args.data, tt.args.now)) {
				return
			}

			assert.Equalf(t, tt.want, got, "TransformODBillingProject(%v, %v, %v, %v)", tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)
		})
	}
}

func bqNullFloat(floatValue float64, isValid bool) bigquery.NullFloat64 {
	return bigquery.NullFloat64{
		Float64: floatValue,
		Valid:   isValid,
	}
}

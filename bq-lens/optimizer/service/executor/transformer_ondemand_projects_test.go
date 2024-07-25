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

func TestTransformODProject(t *testing.T) {
	var (
		mockNowTime = time.Date(2024, 5, 18, 0, 0, 0, 0, time.UTC)
		discount    = 1.0
		isValid     = true
	)

	type args struct {
		timeRange        bqmodels.TimeRange
		customerDiscount float64
		data             *bqmodels.RunODProjectResult
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
				data:             &bqmodels.RunODProjectResult{},
				now:              mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.ProjectScanPrice: {bqmodels.TimeRangeMonth: fsModels.ProjectScanPriceDocument{}},
				bqmodels.ProjectScanTB:    {bqmodels.TimeRangeMonth: fsModels.ProjectScanTBDocument{}},
			},
			wantErr: assert.NoError,
		},
		{
			name: "with data",
			args: args{
				timeRange:        bqmodels.TimeRangeMonth,
				customerDiscount: discount,
				data: &bqmodels.RunODProjectResult{
					Project: []bqmodels.ProjectResult{{ProjectID: "project1", ScanTB: bigquery.NullFloat64{Float64: 1.0, Valid: isValid}}},
					ProjectTopTables: []bqmodels.ProjectTopTablesResult{
						{
							ProjectID: "project1",
							TableID:   "table1",
							ScanTB:    bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
						},
					},
					ProjectTopDatasets: []bqmodels.ProjectTopDatasetsResult{
						{
							ProjectID: "project1",
							DatasetID: "dataset1",
							ScanTB:    bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
						},
					},
					ProjectTopQueries: []bqmodels.ProjectTopQueriesResult{
						{
							ProjectID: "project1",
							TopQueriesResult: bqmodels.TopQueriesResult{
								BillingProjectID:      "project1",
								UserID:                "email@mail.com",
								JobID:                 "JobID",
								Location:              "us-central1",
								ExecutedQueries:       1,
								AvgExecutionTimeSec:   bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
								TotalExecutionTimeSec: bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
								AvgSlots:              bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
								AvgScanTB:             bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
								TotalScanTB:           bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
							},
						},
					},
					ProjectTopUsers: []bqmodels.ProjectTopUsersResult{
						{
							ProjectID: "project1",
							UserEmail: "email@mail.com",
							ScanTB:    bigquery.NullFloat64{Float64: 1.0, Valid: isValid},
						},
					},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.ProjectScanPrice: {bqmodels.TimeRangeMonth: fsModels.ProjectScanPriceDocument{
					"project1": {
						ProjectID:  "project1",
						ScanPrice:  PricePerTBScan,
						LastUpdate: mockNowTime,
						TopQuery: map[string]fsModels.ProjectTopQueryPrice{
							"JobID": {
								AvgScanPrice:   PricePerTBScan,
								Location:       "us-central1",
								TotalScanPrice: PricePerTBScan,
								UserID:         "email@mail.com",
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   1.0,
									AvgSlots:              1.0,
									ExecutedQueries:       1,
									TotalExecutionTimeSec: 1.0,
									BillingProjectID:      "project1",
								},
							},
						},
						TopUsers: map[string]float64{
							"email@mail.com": PricePerTBScan,
						},
						TopTable: map[string]float64{
							"table1": PricePerTBScan,
						},
						TopDataset: map[string]float64{
							"dataset1": PricePerTBScan,
						},
					},
				}},
				bqmodels.ProjectScanTB: {bqmodels.TimeRangeMonth: fsModels.ProjectScanTBDocument{
					"project1": {
						ProjectID:  "project1",
						ScanTB:     1,
						LastUpdate: mockNowTime,
						TopQuery: map[string]fsModels.ProjectTopQueryPrice{
							"JobID": {
								AvgScanPrice:   1,
								Location:       "us-central1",
								ProjectID:      "project1",
								TotalScanPrice: 1,
								UserID:         "email@mail.com",
								CommonTopQuery: fsModels.CommonTopQuery{AvgExecutionTimeSec: 1.0, AvgSlots: 1.0, ExecutedQueries: 1, TotalExecutionTimeSec: 1.0, BillingProjectID: "project1"},
							},
						},
						TopUsers: map[string]float64{
							"email@mail.com": 1.0,
						},
						TopTable: map[string]float64{
							"table1": 1.0,
						},
						TopDataset: map[string]float64{
							"dataset1": 1.0,
						},
					},
				}},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformODProject(tt.args.timeRange, tt.args.customerDiscount, tt.args.data, tt.args.now)
			if !tt.wantErr(t, err, fmt.Sprintf("TransformODProject() error = %v, wantErr %v", err, tt.wantErr)) {
				t.Errorf("TransformODProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equalf(t, tt.want, got, "TransformODProject() = %v, want %v", got, tt.want)
		})
	}
}

package executor

import (
	"reflect"
	"testing"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformBillingProject(t *testing.T) {
	timeRange := bqmodels.TimeRangeDay
	now := time.Now()

	type args struct {
		timeRange bqmodels.TimeRange
		data      bqmodels.RunBillingProjectResult
		now       time.Time
	}

	tests := []struct {
		name    string
		args    args
		want    dal.RecommendationSummary
		wantErr bool
	}{
		{
			name: "Test TransformBillingProject",
			args: args{
				timeRange: timeRange,
				data: bqmodels.RunBillingProjectResult{
					Slots: []bqmodels.BillingProjectSlotsResult{
						{
							BillingProjectID: "test1",
							Slots:            1,
						},
						{
							BillingProjectID: "test2",
							Slots:            2,
						},
					},
					TopUsers: []bqmodels.BillingProjectSlotsTopUsersResult{
						{
							BillingProjectID: "test1",
							UserEmail:        "user1",
							Slots:            1,
						},
						{
							BillingProjectID: "test1",
							UserEmail:        "user2",
							Slots:            12,
						},
						{
							BillingProjectID: "test2",
							UserEmail:        "user2",
							Slots:            2,
						},
					},
					TopQueries: []bqmodels.BillingProjectSlotsTopQueriesResult{
						{
							BillingProjectID:      "test1",
							UserID:                "user1",
							JobID:                 "job1",
							Location:              "Location1",
							ExecutedQueries:       1,
							AvgExecutionTimeSec:   2,
							TotalExecutionTimeSec: 3,
							AvgSlots:              4,
							AvgScanTB:             5,
							TotalScanTB:           6,
						},
						{
							BillingProjectID:      "test1",
							UserID:                "user2",
							JobID:                 "job2",
							Location:              "Location2",
							ExecutedQueries:       11,
							AvgExecutionTimeSec:   12,
							TotalExecutionTimeSec: 13,
							AvgSlots:              14,
							AvgScanTB:             15,
							TotalScanTB:           16,
						},
					},
				},
				now: now,
			},
			want: dal.RecommendationSummary{
				bqmodels.BillingProjectSlots: {timeRange: fsModels.BillingProjectDocument{
					"test1": {
						BillingProjectID: "test1",
						Slots:            1,
						TopUsers: map[string]float64{
							"user1": 1,
							"user2": 12,
						},
						TopQuery: map[string]fsModels.BillingProjectSlotsTopQuery{
							"job1": {
								AvgScanTB:   5,
								Location:    "Location1",
								TotalScanTB: 6,
								UserID:      "user1",
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   2,
									AvgSlots:              4,
									ExecutedQueries:       1,
									TotalExecutionTimeSec: 3,
									BillingProjectID:      "test1",
								},
							},
							"job2": {
								AvgScanTB:   15,
								Location:    "Location2",
								TotalScanTB: 16,
								UserID:      "user2",
								CommonTopQuery: fsModels.CommonTopQuery{
									AvgExecutionTimeSec:   12,
									AvgSlots:              14,
									ExecutedQueries:       11,
									TotalExecutionTimeSec: 13,
									BillingProjectID:      "test1",
								},
							},
						},
						LastUpdate: now,
					},
					"test2": {
						BillingProjectID: "test2",
						LastUpdate:       now,
						Slots:            2,
						TopUsers: map[string]float64{
							"user2": 2,
						},
					},
				}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformBillingProject(tt.args.timeRange, &tt.args.data, tt.args.now)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransformBillingProject() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransformBillingProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformerUserSlots(t *testing.T) {
	timeRange := bqmodels.TimeRangeDay

	userID1 := "userID-1"
	userID2 := "userID-2"
	jobID1 := "jobID-1"
	jobID2 := "jobID-2"
	jobID3 := "jobID-3"
	location := "location-1"
	billingProjectID := "project-1"

	type args struct {
		timeRange bqmodels.TimeRange
		data      *bqmodels.RunUserSlotsResult
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "Collect user slots statistics",
			args: args{
				timeRange: timeRange,
				data: &bqmodels.RunUserSlotsResult{
					UserSlotsTopQueries: []bqmodels.UserSlotsTopQueriesResult{
						{
							UserID:                userID1,
							JobID:                 jobID1,
							Location:              location,
							BillingProjectID:      billingProjectID,
							ExecutedQueries:       100,
							AvgExecutionTimeSec:   3.14,
							TotalExecutionTimeSec: 1138,
							AvgSlots:              1.41,
							AvgScanTB:             6.022,
							TotalScanTB:           1000,
						},
						{
							UserID:                userID1,
							JobID:                 jobID2,
							Location:              location,
							BillingProjectID:      billingProjectID,
							ExecutedQueries:       200,
							AvgExecutionTimeSec:   3.14,
							TotalExecutionTimeSec: 1138,
							AvgSlots:              1.41,
							AvgScanTB:             6.022,
							TotalScanTB:           2000,
						},
						{
							UserID:                userID2,
							JobID:                 jobID3,
							Location:              location,
							BillingProjectID:      billingProjectID,
							ExecutedQueries:       300,
							AvgExecutionTimeSec:   3.14,
							TotalExecutionTimeSec: 1138,
							AvgSlots:              1.41,
							AvgScanTB:             6.022,
							TotalScanTB:           3000,
						},
					},
					UserSlots: []bqmodels.UserSlotsResult{
						{
							UserID: userID1,
							Slots:  42,
						},
						{
							UserID: userID2,
							Slots:  99,
						},
					},
				},
			},
			want: dal.RecommendationSummary{
				bqmodels.UserSlots: {
					bqmodels.TimeRangeDay: firestoremodels.UserSlotsDocument{
						userID1: {
							UserID: userID1,
							Slots:  42,
							TopQueries: map[string]firestoremodels.UserTopQuery{
								jobID1: {
									AvgScanTB:   6.022,
									Location:    location,
									TotalScanTB: 1000,
									UserID:      userID1,
									CommonTopQuery: firestoremodels.CommonTopQuery{
										AvgExecutionTimeSec:   3.14,
										AvgSlots:              1.41,
										ExecutedQueries:       100,
										TotalExecutionTimeSec: 1138,
										BillingProjectID:      billingProjectID,
									},
								},
								jobID2: {
									AvgScanTB:   6.022,
									Location:    location,
									TotalScanTB: 2000,
									UserID:      userID1,
									CommonTopQuery: firestoremodels.CommonTopQuery{
										AvgExecutionTimeSec:   3.14,
										AvgSlots:              1.41,
										ExecutedQueries:       200,
										TotalExecutionTimeSec: 1138,
										BillingProjectID:      billingProjectID,
									},
								},
							},
							LastUpdate: mockTime,
						},
						userID2: {
							UserID: userID2,
							Slots:  99,
							TopQueries: map[string]firestoremodels.UserTopQuery{
								jobID3: {
									AvgScanTB:   6.022,
									Location:    location,
									TotalScanTB: 3000,
									UserID:      userID2,
									CommonTopQuery: firestoremodels.CommonTopQuery{
										AvgExecutionTimeSec:   3.14,
										AvgSlots:              1.41,
										ExecutedQueries:       300,
										TotalExecutionTimeSec: 1138,
										BillingProjectID:      billingProjectID,
									},
								},
							},
							LastUpdate: mockTime,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummary, _ := TransformUserSlots(tt.args.timeRange, tt.args.data, mockTime)
			assert.Equal(t, tt.want, gotSummary)
		})
	}
}

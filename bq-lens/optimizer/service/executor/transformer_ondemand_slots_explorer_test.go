package executor

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"github.com/stretchr/testify/assert"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	firestoremodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformOnDemandSlotsExplorer(t *testing.T) {
	var (
		sixteenMockDate   = civil.Date{Year: 2024, Month: 5, Day: 16}
		seventeenMockDate = civil.Date{Year: 2024, Month: 5, Day: 17}
		mockNowTime       = time.Date(2024, 5, 18, 0, 0, 0, 0, time.UTC)
	)

	type args struct {
		timeRange bqmodels.TimeRange
		data      []bqmodels.OnDemandSlotsExplorerResult
		now       time.Time
	}

	tests := []struct {
		name    string
		args    args
		want    dal.RecommendationSummary
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty input",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data:      []bqmodels.OnDemandSlotsExplorerResult{},
				now:       mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerOnDemand: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day:        firestoremodels.TimeSeriesData{},
						Hour:       firestoremodels.TimeSeriesData{},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "single entry",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.OnDemandSlotsExplorerResult{
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 5, AvgSlots: 10, MaxSlots: 20},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerOnDemand: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day: firestoremodels.TimeSeriesData{
							XAxis: []string{"2024-05-16"},
							Bar:   []float64{10},
							Line:  []float64{20},
						},
						Hour: firestoremodels.TimeSeriesData{
							XAxis: []string{"5"},
							Bar:   []float64{10},
							Line:  []float64{20},
						},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "multiple entries",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.OnDemandSlotsExplorerResult{
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 5, AvgSlots: 10, MaxSlots: 20},
					{Day: bigquery.NullDate{Date: seventeenMockDate, Valid: true}, Hour: 6, AvgSlots: 15, MaxSlots: 25},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerOnDemand: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day: firestoremodels.TimeSeriesData{
							XAxis: []string{"2024-05-16", "2024-05-17"},
							Bar:   []float64{10, 15},
							Line:  []float64{20, 25},
						},
						Hour: firestoremodels.TimeSeriesData{
							XAxis: []string{"5", "6"},
							Bar:   []float64{10, 15},
							Line:  []float64{20, 25},
						},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "entries with same day and different hours",
			args: args{
				timeRange: bqmodels.TimeRangeMonth,
				data: []bqmodels.OnDemandSlotsExplorerResult{
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 5, AvgSlots: 10, MaxSlots: 20},
					{Day: bigquery.NullDate{Date: sixteenMockDate, Valid: true}, Hour: 6, AvgSlots: 15, MaxSlots: 25},
				},
				now: mockNowTime,
			},
			want: dal.RecommendationSummary{
				bqmodels.SlotsExplorerOnDemand: {
					bqmodels.TimeRangeMonth: firestoremodels.ExplorerDocument{
						Day: firestoremodels.TimeSeriesData{
							XAxis: []string{"2024-05-16"},
							Bar:   []float64{12.5},
							Line:  []float64{25},
						},
						Hour: firestoremodels.TimeSeriesData{
							XAxis: []string{"5", "6"},
							Bar:   []float64{10, 15},
							Line:  []float64{20, 25},
						},
						LastUpdate: mockNowTime,
					},
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransformOnDemandSlotsExplorer(tt.args.timeRange, tt.args.data, tt.args.now)
			if !tt.wantErr(t, err, fmt.Sprintf("TransformOnDemandSlotsExplorer(%v, %v, %v)", tt.args.timeRange, tt.args.data, tt.args.now)) {
				return
			}

			assert.Equalf(t, tt.want, got, "TransformOnDemandSlotsExplorer(%v, %v, %v)", tt.args.timeRange, tt.args.data, tt.args.now)
		})
	}
}

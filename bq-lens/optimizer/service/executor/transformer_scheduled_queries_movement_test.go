package executor

import (
	"testing"
	"time"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	fsModels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/firestore"
)

func TestTransformScheduledQueriesMovement(t *testing.T) {
	var (
		data = []bqmodels.ScheduledQueriesMovementResult{
			{
				JobID:            "JobID1",
				Location:         "Location1",
				BillingProjectID: "BillingProjectID1",
				ScheduledTime:    "2022-01-01T12:00:00Z",
				AllJobs:          1,
				Slots:            1.0,
				SavingsPrice:     1.0,
			},
			{
				JobID:            "JobID2",
				Location:         "Location2",
				BillingProjectID: "BillingProjectID2",
				ScheduledTime:    "2022-01-01T12:00:00Z",
				AllJobs:          2,
				Slots:            2.0,
				SavingsPrice:     2.0, // sould be 1.0 it will ignored and taken from first row
			},
			{
				JobID:            "JobID3",
				Location:         "Location3",
				BillingProjectID: "BillingProjectID3",
				ScheduledTime:    "2022-01-01T12:00:00Z",
				AllJobs:          3,
				Slots:            3.0,
				SavingsPrice:     3.0, // sould be 1.0 it will ignored and taken from first row
			},
		}
	)

	periodTotalPriceMapping := domain.PeriodTotalPrice{
		bqmodels.TimeRangeDay:   {TotalScanPrice: 1.0},
		bqmodels.TimeRangeWeek:  {TotalScanPrice: 7.0},
		bqmodels.TimeRangeMonth: {TotalScanPrice: 30.0},
	}

	detailedTableFieldsMapping := map[string]fsModels.FieldDetail{
		"jobId":            {Order: 0, Visible: true},
		"location":         {Order: 1, Visible: false},
		"billingProjectId": {Order: 2, Visible: false},
		"scheduledTime":    {Order: 3, Visible: true},
		"allJobs":          {Order: 4, Visible: true},
		"slots":            {Order: 5, Visible: true},
	}

	now := time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)

	type args struct {
		timeRange               bqmodels.TimeRange
		customerDiscount        float64
		periodTotalPriceMapping domain.PeriodTotalPrice
		data                    []bqmodels.ScheduledQueriesMovementResult
		now                     time.Time
	}

	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "with 'past-1-day' time range",
			args: args{
				timeRange:               bqmodels.TimeRangeDay,
				customerDiscount:        0.5,
				periodTotalPriceMapping: periodTotalPriceMapping,
				data:                    data,
				now:                     now,
			},
			want: dal.RecommendationSummary{
				bqmodels.ScheduledQueriesMovement: {bqmodels.TimeRangeDay: fsModels.ScheduledQueriesMovement{
					DetailedTable: []fsModels.ScheduledQueriesDetailTable{
						{
							JobID:            "JobID1",
							Location:         "Location1",
							BillingProjectID: "BillingProjectID1",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          1,
							Slots:            1.0,
						},
						{
							JobID:            "JobID2",
							Location:         "Location2",
							BillingProjectID: "BillingProjectID2",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          2,
							Slots:            2.0,
						},
						{
							JobID:            "JobID3",
							Location:         "Location3",
							BillingProjectID: "BillingProjectID3",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          3,
							Slots:            3.0,
						},
					},
					DetailedTableFieldsMapping: detailedTableFieldsMapping,
					Recommendation:             scheduledQueriesMovementRecommendation,
					SavingsPercentage:          50,
					SavingsPrice:               0.5,
				}},
			},
		},
		{
			name: "with 'past-7-day' time range",
			args: args{
				timeRange:               bqmodels.TimeRangeWeek,
				customerDiscount:        0.5,
				periodTotalPriceMapping: periodTotalPriceMapping,
				data:                    data,
				now:                     now,
			},
			want: dal.RecommendationSummary{
				bqmodels.ScheduledQueriesMovement: {bqmodels.TimeRangeWeek: fsModels.ScheduledQueriesMovement{
					DetailedTable: []fsModels.ScheduledQueriesDetailTable{
						{
							JobID:            "JobID1",
							Location:         "Location1",
							BillingProjectID: "BillingProjectID1",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          1,
							Slots:            1.0,
						},
						{
							JobID:            "JobID2",
							Location:         "Location2",
							BillingProjectID: "BillingProjectID2",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          2,
							Slots:            2.0,
						},
						{
							JobID:            "JobID3",
							Location:         "Location3",
							BillingProjectID: "BillingProjectID3",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          3,
							Slots:            3.0,
						},
					},
					DetailedTableFieldsMapping: detailedTableFieldsMapping,
					Recommendation:             scheduledQueriesMovementRecommendation,
					SavingsPercentage:          50,
					SavingsPrice:               3.5,
				}},
			},
		},
		{
			name: "with 'past-30-days' time range",
			args: args{
				timeRange:               bqmodels.TimeRangeMonth,
				customerDiscount:        0.5,
				periodTotalPriceMapping: periodTotalPriceMapping,
				data:                    data,
				now:                     now,
			},
			want: dal.RecommendationSummary{
				bqmodels.ScheduledQueriesMovement: {bqmodels.TimeRangeMonth: fsModels.ScheduledQueriesMovement{
					DetailedTable: []fsModels.ScheduledQueriesDetailTable{
						{
							JobID:            "JobID1",
							Location:         "Location1",
							BillingProjectID: "BillingProjectID1",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          1,
							Slots:            1.0,
						},
						{
							JobID:            "JobID2",
							Location:         "Location2",
							BillingProjectID: "BillingProjectID2",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          2,
							Slots:            2.0,
						},
						{
							JobID:            "JobID3",
							Location:         "Location3",
							BillingProjectID: "BillingProjectID3",
							ScheduledTime:    "2022-01-01T12:00:00Z",
							AllJobs:          3,
							Slots:            3.0,
						},
					},
					DetailedTableFieldsMapping: detailedTableFieldsMapping,
					Recommendation:             scheduledQueriesMovementRecommendation,
					SavingsPercentage:          50,
					SavingsPrice:               15,
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TransformScheduledQueriesMovement(tt.args.timeRange, tt.args.customerDiscount, tt.args.periodTotalPriceMapping, tt.args.data, tt.args.now)

			if _, ok := got[bqmodels.ScheduledQueriesMovement]; !ok {
				t.Errorf("Missing map key, want %v", bqmodels.ScheduledQueriesMovement)
				return
			}

			if _, ok := got[bqmodels.ScheduledQueriesMovement][tt.args.timeRange]; !ok {
				t.Errorf("Missing timerange map key, want %v", tt.args.timeRange)
				return
			}

			document := got[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesDocument)

			// DetailedTableFieldsMapping
			if len(document.Data.DetailedTableFieldsMapping) != len(tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).DetailedTableFieldsMapping) {
				t.Errorf("DetailedTableFieldsMapping length mismatch, want %v, got %v", len(tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).DetailedTableFieldsMapping), len(document.Data.DetailedTableFieldsMapping))
				return
			}

			for key, want := range tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).DetailedTableFieldsMapping {
				if _, ok := document.Data.DetailedTableFieldsMapping[key]; !ok {
					t.Errorf("Missing key %v in DetailedTableFieldsMapping", key)
					continue
				}

				if document.Data.DetailedTableFieldsMapping[key].Order != want.Order {
					t.Errorf("DetailedTableFieldsMapping order mismatch, want %v, got %v", want.Order, document.Data.DetailedTableFieldsMapping[key].Order)
				}

				if document.Data.DetailedTableFieldsMapping[key].Visible != want.Visible {
					t.Errorf("DetailedTableFieldsMapping visible mismatch, want %v, got %v", want.Visible, document.Data.DetailedTableFieldsMapping[key].Visible)
				}
			}

			// DetailedTable
			if len(document.Data.DetailedTable) != len(tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).DetailedTable) {
				t.Errorf("DetailedTable length mismatch, want %v, got %v", len(tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).DetailedTable), len(document.Data.DetailedTable))
				return
			}

			for i, want := range tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).DetailedTable {
				if document.Data.DetailedTable[i].JobID != want.JobID {
					t.Errorf("DetailedTable JobID mismatch, want %v, got %v", want.JobID, document.Data.DetailedTable[i].JobID)
				}

				if document.Data.DetailedTable[i].Location != want.Location {
					t.Errorf("DetailedTable Location mismatch, want %v, got %v", want.Location, document.Data.DetailedTable[i].Location)
				}

				if document.Data.DetailedTable[i].BillingProjectID != want.BillingProjectID {
					t.Errorf("DetailedTable BillingProjectID mismatch, want %v, got %v", want.BillingProjectID, document.Data.DetailedTable[i].BillingProjectID)
				}

				if document.Data.DetailedTable[i].ScheduledTime != want.ScheduledTime {
					t.Errorf("DetailedTable ScheduledTime mismatch, want %v, got %v", want.ScheduledTime, document.Data.DetailedTable[i].ScheduledTime)
				}

				if document.Data.DetailedTable[i].AllJobs != want.AllJobs {
					t.Errorf("DetailedTable AllJobs mismatch, want %v, got %v", want.AllJobs, document.Data.DetailedTable[i].AllJobs)
				}

				if document.Data.DetailedTable[i].Slots != want.Slots {
					t.Errorf("DetailedTable Slots mismatch, want %v, got %v", want.Slots, document.Data.DetailedTable[i].Slots)
				}
			}

			// SavingsPrice
			if document.Data.SavingsPrice != tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).SavingsPrice {
				t.Errorf("SavingsPrice mismatch, want %v, got %v", tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).SavingsPrice, document.Data.SavingsPrice)
			}

			// SavingsPercentage
			if document.Data.SavingsPercentage != tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).SavingsPercentage {
				t.Errorf("SavingsPercentage mismatch, want %v, got %v", tt.want[bqmodels.ScheduledQueriesMovement][tt.args.timeRange].(fsModels.ScheduledQueriesMovement).SavingsPercentage, document.Data.SavingsPercentage)
			}
		})
	}
}

package bqlens

import (
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"github.com/stretchr/testify/assert"

	optimizerDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
)

var (
	testReservationID1       = "projects/test-project-1/locations/test-location-1/reservations/test-reservation-1"
	testReservationID2       = "projects/test-project-2/locations/test-location-2/reservations/test-reservation-2"
	parsedTestReservationID1 = "test-project-1:test-location-1.test-reservation-1"
	parsedTestReservationID2 = "test-project-2:test-location-2.test-reservation-2"

	testRegion1 = "test-region-1"
	testRegion2 = "test-region-2"
)

func TestCostForReservationID(t *testing.T) {
	tests := []struct {
		name    string
		args    *BQLensQueryArgs
		want    ReservationsCosts
		wantErr bool
	}{
		{
			name: "a few assignments",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID2,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
				},
				CapacityCommitments: []optimizerDomain.CapacityCommitment{
					{
						Plan:                reservationpb.CapacityCommitment_MONTHLY,
						Edition:             reservationpb.Edition_ENTERPRISE,
						CommitmentStartTime: time.Date(2024, 01, 01, 0, 0, 0, 0, time.UTC),
						CommitmentEndTime:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
					},
					{
						Plan:                reservationpb.CapacityCommitment_ANNUAL,
						Edition:             reservationpb.Edition_ENTERPRISE,
						CommitmentStartTime: time.Date(2025, 01, 01, 0, 0, 0, 0, time.UTC),
						CommitmentEndTime:   time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
					},
				},
				Pricebooks: pricebookDomain.PriceBooksByEdition{
					pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.OnDemand): map[string]float64{
							testRegion1: 1,
							testRegion2: 2,
						},
					},
					pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.OnDemand): map[string]float64{
							testRegion1: 3,
							testRegion2: 4,
						},
					},
				},
				StartTime: time.Date(2024, 06, 04, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 06, 13, 0, 0, 0, 0, time.UTC),
			},
			want: ReservationsCosts{
				ReservationID(parsedTestReservationID1): map[string]float64{
					testRegion1: 3,
					testRegion2: 4,
				},
				ReservationID(parsedTestReservationID2): map[string]float64{
					testRegion1: 3,
					testRegion2: 4,
				},
				defaultPipelineReservationID: map[string]float64{
					testRegion1: 3,
					testRegion2: 4,
				},
			},
		},
		{
			name: "a few assignments, customer has legacy flat rate",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID2,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
				},
				Pricebooks: pricebookDomain.PriceBooksByEdition{
					pricebookDomain.LegacyFlatRate: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.Commit1Mo): map[string]float64{
							testRegion1: 5,
							testRegion2: 6,
						},
					},
				},
				FlatRateUsageTypes: []pricebookDomain.UsageType{pricebookDomain.Commit1Mo},
				StartTime:          time.Date(2024, 06, 04, 0, 0, 0, 0, time.UTC),
				EndTime:            time.Date(2024, 06, 13, 0, 0, 0, 0, time.UTC),
			},
			want: ReservationsCosts{
				ReservationID(parsedTestReservationID1): map[string]float64{
					testRegion1: 5,
					testRegion2: 6,
				},
				ReservationID(parsedTestReservationID2): map[string]float64{
					testRegion1: 5,
					testRegion2: 6,
				},
				defaultPipelineReservationID: map[string]float64{
					testRegion1: 5,
					testRegion2: 6,
				},
			},
		},
		{
			name: "pricebook not found",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
				},
				Pricebooks: pricebookDomain.PriceBooksByEdition{
					pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.OnDemand): map[string]float64{
							testRegion1: 1,
							testRegion2: 2,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no capacity commitment plan. Default to monthly",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
				},
				Pricebooks: pricebookDomain.PriceBooksByEdition{
					pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.OnDemand): map[string]float64{
							testRegion1: 1,
							testRegion2: 2,
						},
					},
				},
			},
			want: ReservationsCosts{
				defaultPipelineReservationID: map[string]float64{
					testRegion1: 1,
					testRegion2: 2,
				},
				ReservationID(parsedTestReservationID1): map[string]float64{
					testRegion1: 1,
					testRegion2: 2,
				},
			},
		},
		{
			name: "unsupported edition",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_EDITION_UNSPECIFIED,
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := costForReservationID(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("CostForReservationID() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetBQEditionsAnalysisPrices(t *testing.T) {
	tests := []struct {
		name    string
		args    *BQLensQueryArgs
		want    []string
		wantErr bool
	}{
		{
			name: "a few assignments",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
				},
				CapacityCommitments: []optimizerDomain.CapacityCommitment{
					{
						Plan:                reservationpb.CapacityCommitment_ANNUAL,
						Edition:             reservationpb.Edition_ENTERPRISE,
						CommitmentStartTime: time.Date(2024, 01, 01, 0, 0, 0, 0, time.UTC),
						CommitmentEndTime:   time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
					},
				},
				Pricebooks: pricebookDomain.PriceBooksByEdition{
					pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.Commit1Yr): map[string]float64{
							testRegion1: 3,
						},
					},
				},
				StartTime: time.Date(2024, 06, 04, 0, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2024, 06, 13, 0, 0, 0, 0, time.UTC),
			},
			want: []string{"CASE", "WHEN reservation = \"test-project-1:test-location-1.test-reservation-1\" AND location = \"test-region-1\" THEN 3", "WHEN reservation = \"default-pipeline\" AND location = \"test-region-1\" THEN 3", "ELSE 0.0\nEND"},
		},
		{
			name: "pricebook not found",
			args: &BQLensQueryArgs{
				ReservationAssignments: []optimizerDomain.ReservationAssignment{
					{
						Reservation: optimizerDomain.Reservation{
							Name:    testReservationID1,
							Edition: reservationpb.Edition_ENTERPRISE,
						},
					},
				},
				Pricebooks: pricebookDomain.PriceBooksByEdition{
					pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
						string(pricebookDomain.OnDemand): map[string]float64{
							testRegion1: 1,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no reservations",
			args: &BQLensQueryArgs{},
			want: []string{noReservationsSubQueryStr},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBQEditionsAnalysisPrices(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBQEditionsAnalysisPrices() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.True(t, checkQueryFields(got, tt.want))
		})
	}
}

func checkQueryFields(query string, substrings []string) bool {
	for _, s := range substrings {
		if !strings.Contains(query, s) {
			return false
		}
	}

	return true
}

package domain

import (
	"time"

	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

type Payload struct {
	BillingProjectWithReservation []BillingProjectWithReservation `json:"billingProjectWithReservation" validate:"dive"`
	Discount                      float64                         `json:"discount" validate:"required"`
}

type BillingProjectWithReservation struct {
	Project  string `json:"project" validate:"required"`
	Location string `json:"location" validate:"required"`
}

// Replacements encapsulates all fields that require substitution for all Executor queries
type Replacements struct {
	ProjectID                string
	DatasetID                string
	TablesDiscoveryTable     string
	StartDate                string
	EndDate                  string
	HistoricalJobs           []string
	ProjectsWithReservations []string
	ProjectsByEdition        map[reservationpb.Edition][]string
	MinDate                  time.Time
	MaxDate                  time.Time
	Location                 string
}

type TotalPrice struct {
	TotalScanPrice    float64 `json:"totalScanPrice" validate:"required"`
	TotalStoragePrice float64 `json:"totalStoragePrice" validate:"required"`
	TotalPrice        float64 `json:"totalPrice" validate:"required"`
}

type PeriodTotalPrice map[bqmodels.TimeRange]TotalPrice

type TransformerContext struct {
	Discount                float64
	TotalScanPricePerPeriod PeriodTotalPrice
}

type Reservation struct {
	Name    string
	Edition reservationpb.Edition
}

type CapacityCommitment struct {
	Name                string
	SlotCount           int64
	Plan                reservationpb.CapacityCommitment_CommitmentPlan
	CommitmentStartTime time.Time
	CommitmentEndTime   time.Time
	Edition             reservationpb.Edition
}
type ReservationAssignment struct {
	Reservation  Reservation
	ProjectsList []string
}

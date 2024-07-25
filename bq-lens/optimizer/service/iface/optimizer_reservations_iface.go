package iface

import (
	"context"

	reservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"google.golang.org/api/option"

	"github.com/doitintl/cloudresourcemanager/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
)

//go:generate mockery --name Reservations --output ../mocks --case=underscore
type Reservations interface {
	NewClient(ctx context.Context, opts []option.ClientOption) (*reservation.Client, error)
	GetProjectsWithReservations(
		ctx context.Context,
		customerID string,
		client *reservation.Client,
		crm iface.CloudResourceManager,
		billingProjectsWithReservations []domain.BillingProjectWithReservation,
	) ([]string, []domain.ReservationAssignment)
	GetCapacityCommitments(
		ctx context.Context,
		customerID string,
		client *reservation.Client,
		billingProjectsWithReservations []domain.BillingProjectWithReservation,
	) map[string][]domain.CapacityCommitment
}

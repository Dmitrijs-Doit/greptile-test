//go:generate mockery --name Reservations --output ../mocks --outpkg mocks --case=underscore

package iface

import (
	"context"

	reservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
)

type Reservations interface {
	ListReservations(
		ctx context.Context,
		client *reservation.Client,
		projectID string,
		location string,
	) ([]*reservationpb.Reservation, error)
	ListAssignments(
		ctx context.Context,
		client *reservation.Client,
		reservationID string,
	) ([]*reservationpb.Assignment, error)
	ListCapacityCommitments(
		ctx context.Context,
		client *reservation.Client,
		projectID string,
		location string,
	) ([]*reservationpb.CapacityCommitment, error)
}

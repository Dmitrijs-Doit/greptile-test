package dal

import (
	"context"
	"errors"
	"fmt"

	reservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"google.golang.org/api/iterator"
)

// Reservations is thin layer on top of the Bigquery Reservations API
// which allows for easy mocking. The client lifecycle is managed by the
// caller, which should close it after its done using it.
type Reservations struct{}

func NewReservations() *Reservations {
	return &Reservations{}
}

// ListReservations returns a slice of reservationpb.Reservation
// for the supplied projectID and location.
func (d *Reservations) ListReservations(
	ctx context.Context,
	client *reservation.Client,
	projectID string,
	location string,
) ([]*reservationpb.Reservation, error) {
	reservations := []*reservationpb.Reservation{}

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	req := &reservationpb.ListReservationsRequest{
		Parent: parent,
	}

	it := client.ListReservations(ctx, req)

	for {
		reservation, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		reservations = append(reservations, reservation)
	}

	return reservations, nil
}

// ListAssignments returns a slice of reservationpb.Assignment for
// the supplied reservation, which corresponds to the Name field in the
// reservationpb.Reservation protobuf.
func (d *Reservations) ListAssignments(
	ctx context.Context,
	client *reservation.Client,
	reservationID string,
) ([]*reservationpb.Assignment, error) {
	assignments := []*reservationpb.Assignment{}

	req := &reservationpb.ListAssignmentsRequest{
		Parent: reservationID,
	}

	it := client.ListAssignments(ctx, req)

	for {
		assignment, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		assignments = append(assignments, assignment)
	}

	return assignments, nil
}

// ListCapacityCommitments returns a slice of active commitments
// for the supplied projectID and location.
func (d *Reservations) ListCapacityCommitments(
	ctx context.Context,
	client *reservation.Client,
	projectID string,
	location string,
) ([]*reservationpb.CapacityCommitment, error) {
	commitments := []*reservationpb.CapacityCommitment{}

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)
	req := &reservationpb.ListCapacityCommitmentsRequest{
		Parent: parent,
	}

	it := client.ListCapacityCommitments(ctx, req)

	for {
		commitment, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		if commitment.State == reservationpb.CapacityCommitment_ACTIVE {
			commitments = append(commitments, commitment)
		}
	}

	return commitments, nil
}

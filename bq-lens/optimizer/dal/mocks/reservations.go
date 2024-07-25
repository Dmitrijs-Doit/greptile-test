// Code generated by mockery v2.42.3. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	reservation "cloud.google.com/go/bigquery/reservation/apiv1"

	reservationpb "cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
)

// Reservations is an autogenerated mock type for the Reservations type
type Reservations struct {
	mock.Mock
}

// ListAssignments provides a mock function with given fields: ctx, client, reservationID
func (_m *Reservations) ListAssignments(ctx context.Context, client *reservation.Client, reservationID string) ([]*reservationpb.Assignment, error) {
	ret := _m.Called(ctx, client, reservationID)

	if len(ret) == 0 {
		panic("no return value specified for ListAssignments")
	}

	var r0 []*reservationpb.Assignment
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *reservation.Client, string) ([]*reservationpb.Assignment, error)); ok {
		return rf(ctx, client, reservationID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *reservation.Client, string) []*reservationpb.Assignment); ok {
		r0 = rf(ctx, client, reservationID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*reservationpb.Assignment)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *reservation.Client, string) error); ok {
		r1 = rf(ctx, client, reservationID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListCapacityCommitments provides a mock function with given fields: ctx, client, projectID, location
func (_m *Reservations) ListCapacityCommitments(ctx context.Context, client *reservation.Client, projectID string, location string) ([]*reservationpb.CapacityCommitment, error) {
	ret := _m.Called(ctx, client, projectID, location)

	if len(ret) == 0 {
		panic("no return value specified for ListCapacityCommitments")
	}

	var r0 []*reservationpb.CapacityCommitment
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *reservation.Client, string, string) ([]*reservationpb.CapacityCommitment, error)); ok {
		return rf(ctx, client, projectID, location)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *reservation.Client, string, string) []*reservationpb.CapacityCommitment); ok {
		r0 = rf(ctx, client, projectID, location)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*reservationpb.CapacityCommitment)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *reservation.Client, string, string) error); ok {
		r1 = rf(ctx, client, projectID, location)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListReservations provides a mock function with given fields: ctx, client, projectID, location
func (_m *Reservations) ListReservations(ctx context.Context, client *reservation.Client, projectID string, location string) ([]*reservationpb.Reservation, error) {
	ret := _m.Called(ctx, client, projectID, location)

	if len(ret) == 0 {
		panic("no return value specified for ListReservations")
	}

	var r0 []*reservationpb.Reservation
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *reservation.Client, string, string) ([]*reservationpb.Reservation, error)); ok {
		return rf(ctx, client, projectID, location)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *reservation.Client, string, string) []*reservationpb.Reservation); ok {
		r0 = rf(ctx, client, projectID, location)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*reservationpb.Reservation)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *reservation.Client, string, string) error); ok {
		r1 = rf(ctx, client, projectID, location)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewReservations creates a new instance of Reservations. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReservations(t interface {
	mock.TestingT
	Cleanup(func())
}) *Reservations {
	mock := &Reservations{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

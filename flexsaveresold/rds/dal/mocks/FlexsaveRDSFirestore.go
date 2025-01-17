// Code generated by mockery v2.33.0. DO NOT EDIT.

package mocks

import (
	context "context"

	iface "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// FlexsaveRDSFirestore is an autogenerated mock type for the FlexsaveRDSFirestore type
type FlexsaveRDSFirestore struct {
	mock.Mock
}

// AddReasonCantEnable provides a mock function with given fields: ctx, customerID, reason
func (_m *FlexsaveRDSFirestore) AddReasonCantEnable(ctx context.Context, customerID string, reason iface.FlexsaveRDSReasonCantEnable) error {
	ret := _m.Called(ctx, customerID, reason)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, iface.FlexsaveRDSReasonCantEnable) error); ok {
		r0 = rf(ctx, customerID, reason)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Create provides a mock function with given fields: ctx, customerID
func (_m *FlexsaveRDSFirestore) Create(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Enable provides a mock function with given fields: ctx, customerID, timeEnabled
func (_m *FlexsaveRDSFirestore) Enable(ctx context.Context, customerID string, timeEnabled time.Time) error {
	ret := _m.Called(ctx, customerID, timeEnabled)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, time.Time) error); ok {
		r0 = rf(ctx, customerID, timeEnabled)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Exists provides a mock function with given fields: ctx, customerID
func (_m *FlexsaveRDSFirestore) Exists(ctx context.Context, customerID string) (bool, error) {
	ret := _m.Called(ctx, customerID)

	var r0 bool

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (bool, error)); ok {
		return rf(ctx, customerID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) bool); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Get provides a mock function with given fields: ctx, customerID
func (_m *FlexsaveRDSFirestore) Get(ctx context.Context, customerID string) (*iface.FlexsaveRDSCache, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *iface.FlexsaveRDSCache

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (*iface.FlexsaveRDSCache, error)); ok {
		return rf(ctx, customerID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) *iface.FlexsaveRDSCache); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*iface.FlexsaveRDSCache)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update provides a mock function with given fields: ctx, customerID, data
func (_m *FlexsaveRDSFirestore) Update(ctx context.Context, customerID string, data interface{}) error {
	ret := _m.Called(ctx, customerID, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, interface{}) error); ok {
		r0 = rf(ctx, customerID, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewFlexsaveRDSFirestore creates a new instance of FlexsaveRDSFirestore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewFlexsaveRDSFirestore(t interface {
	mock.TestingT
	Cleanup(func())
}) *FlexsaveRDSFirestore {
	mock := &FlexsaveRDSFirestore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.35.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// ICsmEngagement is an autogenerated mock type for the ICsmEngagement type
type ICsmEngagement struct {
	mock.Mock
}

// CustomerHasAssets provides a mock function with given fields: ctx, customerID
func (_m *ICsmEngagement) CustomerHasAssets(ctx context.Context, customerID string) (bool, error) {
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

// GetCustomerMRR provides a mock function with given fields: ctx, customerID, exculdeStandalone
func (_m *ICsmEngagement) GetCustomerMRR(ctx context.Context, customerID string, exculdeStandalone bool) (float64, error) {
	ret := _m.Called(ctx, customerID, exculdeStandalone)

	var r0 float64

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string, bool) (float64, error)); ok {
		return rf(ctx, customerID, exculdeStandalone)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string, bool) float64); ok {
		r0 = rf(ctx, customerID, exculdeStandalone)
	} else {
		r0 = ret.Get(0).(float64)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, bool) error); ok {
		r1 = rf(ctx, customerID, exculdeStandalone)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNewCustomersIDs provides a mock function with given fields: ctx, createdAfter
func (_m *ICsmEngagement) GetNewCustomersIDs(ctx context.Context, createdAfter time.Time) ([]string, error) {
	ret := _m.Called(ctx, createdAfter)

	var r0 []string

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, time.Time) ([]string, error)); ok {
		return rf(ctx, createdAfter)
	}

	if rf, ok := ret.Get(0).(func(context.Context, time.Time) []string); ok {
		r0 = rf(ctx, createdAfter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, time.Time) error); ok {
		r1 = rf(ctx, createdAfter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsCustomerResold provides a mock function with given fields: ctx, customerID
func (_m *ICsmEngagement) IsCustomerResold(ctx context.Context, customerID string) (bool, error) {
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

// NewICsmEngagement creates a new instance of ICsmEngagement. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewICsmEngagement(t interface {
	mock.TestingT
	Cleanup(func())
}) *ICsmEngagement {
	mock := &ICsmEngagement{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

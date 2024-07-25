// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	pkg "github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
)

// Onboard is an autogenerated mock type for the Onboard type
type Onboard struct {
	mock.Mock
}

// DeleteCostSimulationData provides a mock function with given fields: ctx, customerID
func (_m *Onboard) DeleteCostSimulationData(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for DeleteCostSimulationData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteOptimizerData provides a mock function with given fields: ctx, customerID
func (_m *Onboard) DeleteOptimizerData(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for DeleteOptimizerData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteSinkMetadata provides a mock function with given fields: ctx, jobID
func (_m *Onboard) DeleteSinkMetadata(ctx context.Context, jobID string) error {
	ret := _m.Called(ctx, jobID)

	if len(ret) == 0 {
		panic("no return value specified for DeleteSinkMetadata")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, jobID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetJobSinkMetadata provides a mock function with given fields: ctx, jobID
func (_m *Onboard) GetJobSinkMetadata(ctx context.Context, jobID string) (*pkg.SinkMetadata, error) {
	ret := _m.Called(ctx, jobID)

	if len(ret) == 0 {
		panic("no return value specified for GetJobSinkMetadata")
	}

	var r0 *pkg.SinkMetadata
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*pkg.SinkMetadata, error)); ok {
		return rf(ctx, jobID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *pkg.SinkMetadata); ok {
		r0 = rf(ctx, jobID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pkg.SinkMetadata)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, jobID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewOnboard creates a new instance of Onboard. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewOnboard(t interface {
	mock.TestingT
	Cleanup(func())
}) *Onboard {
	mock := &Onboard{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
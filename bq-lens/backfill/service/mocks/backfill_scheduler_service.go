// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// BackfillSchedulerService is an autogenerated mock type for the BackfillSchedulerService type
type BackfillSchedulerService struct {
	mock.Mock
}

// ScheduleBackfill provides a mock function with given fields: ctx, sinkID, testMode
func (_m *BackfillSchedulerService) ScheduleBackfill(ctx context.Context, sinkID string, testMode bool) error {
	ret := _m.Called(ctx, sinkID, testMode)

	if len(ret) == 0 {
		panic("no return value specified for ScheduleBackfill")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) error); ok {
		r0 = rf(ctx, sinkID, testMode)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewBackfillSchedulerService creates a new instance of BackfillSchedulerService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBackfillSchedulerService(t interface {
	mock.TestingT
	Cleanup(func())
}) *BackfillSchedulerService {
	mock := &BackfillSchedulerService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
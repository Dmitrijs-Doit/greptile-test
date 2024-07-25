// Code generated by mockery v2.43.2. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// TaskCreator is an autogenerated mock type for the TaskCreator type
type TaskCreator struct {
	mock.Mock
}

// CreateBackfillScheduleTask provides a mock function with given fields: ctx, sinkID
func (_m *TaskCreator) CreateBackfillScheduleTask(ctx context.Context, sinkID string) error {
	ret := _m.Called(ctx, sinkID)

	if len(ret) == 0 {
		panic("no return value specified for CreateBackfillScheduleTask")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, sinkID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateBackfillTask provides a mock function with given fields: ctx, dateBackfillInfo, backfillDate, backfillProject, customerID, sinkID
func (_m *TaskCreator) CreateBackfillTask(ctx context.Context, dateBackfillInfo domain.DateBackfillInfo, backfillDate time.Time, backfillProject string, customerID string, sinkID string) error {
	ret := _m.Called(ctx, dateBackfillInfo, backfillDate, backfillProject, customerID, sinkID)

	if len(ret) == 0 {
		panic("no return value specified for CreateBackfillTask")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.DateBackfillInfo, time.Time, string, string, string) error); ok {
		r0 = rf(ctx, dateBackfillInfo, backfillDate, backfillProject, customerID, sinkID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateTableDiscoveryTask provides a mock function with given fields: ctx, customerID
func (_m *TaskCreator) CreateTableDiscoveryTask(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for CreateTableDiscoveryTask")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewTaskCreator creates a new instance of TaskCreator. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewTaskCreator(t interface {
	mock.TestingT
	Cleanup(func())
}) *TaskCreator {
	mock := &TaskCreator{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

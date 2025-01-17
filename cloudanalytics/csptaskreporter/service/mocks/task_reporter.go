// Code generated by mockery v2.36.0. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/domain"

	mock "github.com/stretchr/testify/mock"
)

// TaskReporter is an autogenerated mock type for the TaskReporter type
type TaskReporter struct {
	mock.Mock
}

// LogTaskSummary provides a mock function with given fields: ctx, taskSummary
func (_m *TaskReporter) LogTaskSummary(ctx context.Context, taskSummary *domain.TaskSummary) {
	_m.Called(ctx, taskSummary)
}

// NewTaskReporter creates a new instance of TaskReporter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewTaskReporter(t interface {
	mock.TestingT
	Cleanup(func())
}) *TaskReporter {
	mock := &TaskReporter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

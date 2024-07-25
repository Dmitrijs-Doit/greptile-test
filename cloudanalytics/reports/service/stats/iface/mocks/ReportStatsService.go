// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// ReportStatsService is an autogenerated mock type for the ReportStatsService type
type ReportStatsService struct {
	mock.Mock
}

// UpdateReportStats provides a mock function with given fields: ctx, reportID, origin, resultDetails
func (_m *ReportStatsService) UpdateReportStats(ctx context.Context, reportID string, origin string, resultDetails map[string]interface{}) error {
	ret := _m.Called(ctx, reportID, origin, resultDetails)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, map[string]interface{}) error); ok {
		r0 = rf(ctx, reportID, origin, resultDetails)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewReportStatsService creates a new instance of ReportStatsService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewReportStatsService(t interface {
	mock.TestingT
	Cleanup(func())
}) *ReportStatsService {
	mock := &ReportStatsService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
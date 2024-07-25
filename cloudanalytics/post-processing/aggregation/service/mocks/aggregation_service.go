// Code generated by mockery v2.36.0. DO NOT EDIT.

package mocks

import (
	bigquery "cloud.google.com/go/bigquery"

	mock "github.com/stretchr/testify/mock"

	report "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// AggregationService is an autogenerated mock type for the AggregationService type
type AggregationService struct {
	mock.Mock
}

// ApplyAggregation provides a mock function with given fields: aggregator, numRows, numCols, resRows
func (_m *AggregationService) ApplyAggregation(aggregator report.Aggregator, numRows int, numCols int, resRows [][]bigquery.Value) error {
	ret := _m.Called(aggregator, numRows, numCols, resRows)

	var r0 error
	if rf, ok := ret.Get(0).(func(report.Aggregator, int, int, [][]bigquery.Value) error); ok {
		r0 = rf(aggregator, numRows, numCols, resRows)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAggregationService creates a new instance of AggregationService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAggregationService(t interface {
	mock.TestingT
	Cleanup(func())
}) *AggregationService {
	mock := &AggregationService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
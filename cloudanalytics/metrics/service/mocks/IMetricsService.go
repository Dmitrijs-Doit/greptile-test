// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	errormsg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"

	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"

	mock "github.com/stretchr/testify/mock"

	service "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service"
)

// IMetricsService is an autogenerated mock type for the IMetricsService type
type IMetricsService struct {
	mock.Mock
}

// DeleteMany provides a mock function with given fields: ctx, req
func (_m *IMetricsService) DeleteMany(ctx context.Context, req service.DeleteMetricsRequest) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, service.DeleteMetricsRequest) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ToExternal provides a mock function with given fields: params
func (_m *IMetricsService) ToExternal(params *metrics.InternalMetricParameters) (*metrics.ExternalMetric, []errormsg.ErrorMsg, error) {
	ret := _m.Called(params)

	var r0 *metrics.ExternalMetric
	var r1 []errormsg.ErrorMsg
	var r2 error
	if rf, ok := ret.Get(0).(func(*metrics.InternalMetricParameters) (*metrics.ExternalMetric, []errormsg.ErrorMsg, error)); ok {
		return rf(params)
	}
	if rf, ok := ret.Get(0).(func(*metrics.InternalMetricParameters) *metrics.ExternalMetric); ok {
		r0 = rf(params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*metrics.ExternalMetric)
		}
	}

	if rf, ok := ret.Get(1).(func(*metrics.InternalMetricParameters) []errormsg.ErrorMsg); ok {
		r1 = rf(params)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]errormsg.ErrorMsg)
		}
	}

	if rf, ok := ret.Get(2).(func(*metrics.InternalMetricParameters) error); ok {
		r2 = rf(params)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// ToInternal provides a mock function with given fields: ctx, customerID, externalMetric
func (_m *IMetricsService) ToInternal(ctx context.Context, customerID string, externalMetric *metrics.ExternalMetric) (*metrics.InternalMetricParameters, []errormsg.ErrorMsg, error) {
	ret := _m.Called(ctx, customerID, externalMetric)

	var r0 *metrics.InternalMetricParameters
	var r1 []errormsg.ErrorMsg
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *metrics.ExternalMetric) (*metrics.InternalMetricParameters, []errormsg.ErrorMsg, error)); ok {
		return rf(ctx, customerID, externalMetric)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *metrics.ExternalMetric) *metrics.InternalMetricParameters); ok {
		r0 = rf(ctx, customerID, externalMetric)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*metrics.InternalMetricParameters)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *metrics.ExternalMetric) []errormsg.ErrorMsg); ok {
		r1 = rf(ctx, customerID, externalMetric)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]errormsg.ErrorMsg)
		}
	}

	if rf, ok := ret.Get(2).(func(context.Context, string, *metrics.ExternalMetric) error); ok {
		r2 = rf(ctx, customerID, externalMetric)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// NewIMetricsService creates a new instance of IMetricsService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIMetricsService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IMetricsService {
	mock := &IMetricsService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
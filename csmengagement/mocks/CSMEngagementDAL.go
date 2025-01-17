// Code generated by mockery v2.40.1. DO NOT EDIT.

package mocks

import (
	context "context"

	dal "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// CSMEngagementDAL is an autogenerated mock type for the CSMEngagementDAL type
type CSMEngagementDAL struct {
	mock.Mock
}

// AddLastCustomerEngagementTime provides a mock function with given fields: ctx, customerID, _a2
func (_m *CSMEngagementDAL) AddLastCustomerEngagementTime(ctx context.Context, customerID string, _a2 time.Time) error {
	ret := _m.Called(ctx, customerID, _a2)

	if len(ret) == 0 {
		panic("no return value specified for AddLastCustomerEngagementTime")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, time.Time) error); ok {
		r0 = rf(ctx, customerID, _a2)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCustomerEngagementDetailsByCustomerID provides a mock function with given fields: ctx
func (_m *CSMEngagementDAL) GetCustomerEngagementDetailsByCustomerID(ctx context.Context) (map[string]dal.EngagementDetails, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetCustomerEngagementDetailsByCustomerID")
	}

	var r0 map[string]dal.EngagementDetails
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (map[string]dal.EngagementDetails, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) map[string]dal.EngagementDetails); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]dal.EngagementDetails)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewCSMEngagementDAL creates a new instance of CSMEngagementDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCSMEngagementDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *CSMEngagementDAL {
	mock := &CSMEngagementDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.32.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// BillingTableManagementService is an autogenerated mock type for the BillingTableManagementService type
type BillingTableManagementService struct {
	mock.Mock
}

// UpdateAggregatedTable provides a mock function with given fields: ctx, suffix, interval, allPartitions
func (_m *BillingTableManagementService) UpdateAggregatedTable(ctx context.Context, suffix string, interval string, allPartitions bool) error {
	ret := _m.Called(ctx, suffix, interval, allPartitions)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool) error); ok {
		r0 = rf(ctx, suffix, interval, allPartitions)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateAllAggregatedTables provides a mock function with given fields: ctx, suffix, allPartitions
func (_m *BillingTableManagementService) UpdateAllAggregatedTables(ctx context.Context, suffix string, allPartitions bool) []error {
	ret := _m.Called(ctx, suffix, allPartitions)

	var r0 []error
	if rf, ok := ret.Get(0).(func(context.Context, string, bool) []error); ok {
		r0 = rf(ctx, suffix, allPartitions)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	return r0
}

// UpdateAllAggregatedTablesAllCustomers provides a mock function with given fields: ctx, allPartitions
func (_m *BillingTableManagementService) UpdateAllAggregatedTablesAllCustomers(ctx context.Context, allPartitions bool) []error {
	ret := _m.Called(ctx, allPartitions)

	var r0 []error
	if rf, ok := ret.Get(0).(func(context.Context, bool) []error); ok {
		r0 = rf(ctx, allPartitions)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	return r0
}

// UpdateCSPAggregatedTable provides a mock function with given fields: ctx, allPartitions, startDate, numPartitions
func (_m *BillingTableManagementService) UpdateCSPAggregatedTable(ctx context.Context, allPartitions bool, startDate string, numPartitions int) error {
	ret := _m.Called(ctx, allPartitions, startDate, numPartitions)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, bool, string, int) error); ok {
		r0 = rf(ctx, allPartitions, startDate, numPartitions)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateCSPTable provides a mock function with given fields: ctx, startDate, endDate
func (_m *BillingTableManagementService) UpdateCSPTable(ctx context.Context, startDate string, endDate string) error {
	ret := _m.Called(ctx, startDate, endDate)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, startDate, endDate)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewBillingTableManagementService creates a new instance of BillingTableManagementService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBillingTableManagementService(t interface {
	mock.TestingT
	Cleanup(func())
}) *BillingTableManagementService {
	mock := &BillingTableManagementService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

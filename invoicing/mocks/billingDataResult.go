// Code generated by mockery v2.13.1. DO NOT EDIT.

package mocks

import (
	bigquery "cloud.google.com/go/bigquery"

	mock "github.com/stretchr/testify/mock"

	pkg "github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"

	time "time"
)

// BillingDataTransformer is an autogenerated mock type for the billingDataResult type
type BillingDataTransformer struct {
	mock.Mock
}

// TransformToDaysToAccountsToCostAndAccountIDs provides a mock function with given fields: rows
func (_m *BillingDataTransformer) TransformToDaysToAccountsToCostAndAccountIDs(rows [][]bigquery.Value) (map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem, []string, error) {
	ret := _m.Called(rows)

	var r0 map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem
	if rf, ok := ret.Get(0).(func([][]bigquery.Value) map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem); ok {
		r0 = rf(rows)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem)
		}
	}

	var r1 []string
	if rf, ok := ret.Get(1).(func([][]bigquery.Value) []string); ok {
		r1 = rf(rows)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).([]string)
		}
	}

	var r2 error
	if rf, ok := ret.Get(2).(func([][]bigquery.Value) error); ok {
		r2 = rf(rows)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

type mockConstructorTestingTNewBillingDataTransformer interface {
	mock.TestingT
	Cleanup(func())
}

// NewBillingDataTransformer creates a new instance of BillingDataTransformer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBillingDataTransformer(t mockConstructorTestingTNewBillingDataTransformer) *BillingDataTransformer {
	mock := &BillingDataTransformer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
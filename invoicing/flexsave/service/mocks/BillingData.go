// Code generated by mockery v2.27.1. DO NOT EDIT.

package mocks

import (
	context "context"

	bigquery "cloud.google.com/go/bigquery"
	cloudanalytics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics"

	gin "github.com/gin-gonic/gin"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// BillingData is an autogenerated mock type for the BillingData type
type BillingData struct {
	mock.Mock
}

// GetCustomerBillingRows provides a mock function with given fields: ctx, customerID, invoiceMonth, provider
func (_m *BillingData) GetCustomerBillingRows(ctx *gin.Context, customerID string, invoiceMonth time.Time, provider string) ([][]bigquery.Value, error) {
	ret := _m.Called(ctx, customerID, invoiceMonth, provider)

	var r0 [][]bigquery.Value

	var r1 error

	if rf, ok := ret.Get(0).(func(*gin.Context, string, time.Time, string) ([][]bigquery.Value, error)); ok {
		return rf(ctx, customerID, invoiceMonth, provider)
	}

	if rf, ok := ret.Get(0).(func(*gin.Context, string, time.Time, string) [][]bigquery.Value); ok {
		r0 = rf(ctx, customerID, invoiceMonth, provider)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([][]bigquery.Value)
		}
	}

	if rf, ok := ret.Get(1).(func(*gin.Context, string, time.Time, string) error); ok {
		r1 = rf(ctx, customerID, invoiceMonth, provider)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFlexsaveQuery provides a mock function with given fields: ctx, invoiceMonth, accounts, provider
func (_m *BillingData) GetFlexsaveQuery(ctx context.Context, invoiceMonth time.Time, accounts []string, provider string) (*cloudanalytics.QueryRequest, error) {
	ret := _m.Called(ctx, invoiceMonth, accounts, provider)

	var r0 *cloudanalytics.QueryRequest

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, time.Time, []string, string) (*cloudanalytics.QueryRequest, error)); ok {
		return rf(ctx, invoiceMonth, accounts, provider)
	}

	if rf, ok := ret.Get(0).(func(context.Context, time.Time, []string, string) *cloudanalytics.QueryRequest); ok {
		r0 = rf(ctx, invoiceMonth, accounts, provider)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*cloudanalytics.QueryRequest)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, time.Time, []string, string) error); ok {
		r1 = rf(ctx, invoiceMonth, accounts, provider)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewBillingData interface {
	mock.TestingT
	Cleanup(func())
}

// NewBillingData creates a new instance of BillingData. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBillingData(t mockConstructorTestingTNewBillingData) *BillingData {
	mock := &BillingData{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.43.0. DO NOT EDIT.

package mocks

import (
	context "context"

	gin "github.com/gin-gonic/gin"

	mock "github.com/stretchr/testify/mock"
)

// AnalyticsAWSInvoicing is an autogenerated mock type for the AnalyticsAWSInvoicing type
type AnalyticsAWSInvoicing struct {
	mock.Mock
}

// AmazonWebServicesInvoicingDataWorker provides a mock function with given fields: ginCtx, customerID, invoiceMonthInput, dry
func (_m *AnalyticsAWSInvoicing) AmazonWebServicesInvoicingDataWorker(ginCtx *gin.Context, customerID string, invoiceMonthInput string, dry bool) error {
	ret := _m.Called(ginCtx, customerID, invoiceMonthInput, dry)

	if len(ret) == 0 {
		panic("no return value specified for AmazonWebServicesInvoicingDataWorker")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(*gin.Context, string, string, bool) error); ok {
		r0 = rf(ginCtx, customerID, invoiceMonthInput, dry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateAmazonWebServicesInvoicingData provides a mock function with given fields: ctx, invoiceMonth, version, validateWithOld, dry
func (_m *AnalyticsAWSInvoicing) UpdateAmazonWebServicesInvoicingData(ctx context.Context, invoiceMonth string, version string, validateWithOld bool, dry bool) error {
	ret := _m.Called(ctx, invoiceMonth, version, validateWithOld, dry)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAmazonWebServicesInvoicingData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool, bool) error); ok {
		r0 = rf(ctx, invoiceMonth, version, validateWithOld, dry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewAnalyticsAWSInvoicing creates a new instance of AnalyticsAWSInvoicing. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAnalyticsAWSInvoicing(t interface {
	mock.TestingT
	Cleanup(func())
}) *AnalyticsAWSInvoicing {
	mock := &AnalyticsAWSInvoicing{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

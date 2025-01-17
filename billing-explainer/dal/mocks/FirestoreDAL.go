// Code generated by mockery v2.40.1. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"

	mock "github.com/stretchr/testify/mock"
)

// FirestoreDAL is an autogenerated mock type for the FirestoreDAL type
type FirestoreDAL struct {
	mock.Mock
}

// GetPayerAccountDoc provides a mock function with given fields: ctx, payerID
func (_m *FirestoreDAL) GetPayerAccountDoc(ctx context.Context, payerID string) (map[string]interface{}, error) {
	ret := _m.Called(ctx, payerID)

	if len(ret) == 0 {
		panic("no return value specified for GetPayerAccountDoc")
	}

	var r0 map[string]interface{}
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (map[string]interface{}, error)); ok {
		return rf(ctx, payerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) map[string]interface{}); ok {
		r0 = rf(ctx, payerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, payerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateEntityFirestoreDoc provides a mock function with given fields: ctx, isBackfill, yearMonth, entityID, invoicingMode, summaryBqResults, bucketName, serviceBreakdownResults, accountBreakdownResults
func (_m *FirestoreDAL) UpdateEntityFirestoreDoc(ctx context.Context, isBackfill bool, yearMonth string, entityID string, invoicingMode string, summaryBqResults []domain.SummaryBQ, bucketName string, serviceBreakdownResults []domain.ServiceRecord, accountBreakdownResults []domain.AccountRecord) error {
	ret := _m.Called(ctx, isBackfill, yearMonth, entityID, invoicingMode, summaryBqResults, bucketName, serviceBreakdownResults, accountBreakdownResults)

	if len(ret) == 0 {
		panic("no return value specified for UpdateEntityFirestoreDoc")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, bool, string, string, string, []domain.SummaryBQ, string, []domain.ServiceRecord, []domain.AccountRecord) error); ok {
		r0 = rf(ctx, isBackfill, yearMonth, entityID, invoicingMode, summaryBqResults, bucketName, serviceBreakdownResults, accountBreakdownResults)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewFirestoreDAL creates a new instance of FirestoreDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewFirestoreDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *FirestoreDAL {
	mock := &FirestoreDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

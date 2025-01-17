// Code generated by mockery v2.13.1. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	types "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

// SavingsPlansDAL is an autogenerated mock type for the SavingsPlansDAL type
type SavingsPlansDAL struct {
	mock.Mock
}

// CreateCustomerSavingsPlansCache provides a mock function with given fields: ctx, customerID, savingsPlans
func (_m *SavingsPlansDAL) CreateCustomerSavingsPlansCache(ctx context.Context, customerID string, savingsPlans []types.SavingsPlanData) error {
	ret := _m.Called(ctx, customerID, savingsPlans)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []types.SavingsPlanData) error); ok {
		r0 = rf(ctx, customerID, savingsPlans)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewSavingsPlansDAL interface {
	mock.TestingT
	Cleanup(func())
}

// NewSavingsPlansDAL creates a new instance of SavingsPlansDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSavingsPlansDAL(t mockConstructorTestingTNewSavingsPlansDAL) *SavingsPlansDAL {
	mock := &SavingsPlansDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.30.1. DO NOT EDIT.

package mocks

import (
	context "context"

	iface "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/iface"
	mock "github.com/stretchr/testify/mock"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// AddReasonCantEnableBasedOnRecommendation provides a mock function with given fields: ctx, customerID, savingsSummary
func (_m *Service) AddReasonCantEnableBasedOnSavingsSummary(ctx context.Context, customerID string, savingsSummary iface.FlexsaveSavingsSummary) error {
	ret := _m.Called(ctx, customerID, savingsSummary)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, iface.FlexsaveSavingsSummary) error); ok {
		r0 = rf(ctx, customerID, savingsSummary)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateSavingsSummaryBasedOnRecommendation provides a mock function with given fields: ctx, customerID
func (_m *Service) CreateSavingsSummaryBasedOnRecommendation(ctx context.Context, customerID string) (iface.FlexsaveSavingsSummary, error) {
	ret := _m.Called(ctx, customerID)

	var r0 iface.FlexsaveSavingsSummary

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (iface.FlexsaveSavingsSummary, error)); ok {
		return rf(ctx, customerID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) iface.FlexsaveSavingsSummary); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Get(0).(iface.FlexsaveSavingsSummary)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewService(t interface {
	mock.TestingT
	Cleanup(func())
}) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
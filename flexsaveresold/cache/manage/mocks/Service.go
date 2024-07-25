// Code generated by mockery v2.35.4. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	types "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// Disable provides a mock function with given fields: ctx, customerID
func (_m *Service) Disable(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DisableCustomerPayers provides a mock function with given fields: ctx, customerID
func (_m *Service) DisableCustomerPayers(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// EnableEligiblePayers provides a mock function with given fields: ctx
func (_m *Service) EnableEligiblePayers(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// HandleMPAActivation provides a mock function with given fields: ctx, mpaID
func (_m *Service) HandleMPAActivation(ctx context.Context, mpaID string) error {
	ret := _m.Called(ctx, mpaID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, mpaID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PayerStatusUpdateForEnabledCustomers provides a mock function with given fields: ctx
func (_m *Service) PayerStatusUpdateForEnabledCustomers(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdatePayerConfigs provides a mock function with given fields: ctx, configs
func (_m *Service) UpdatePayerConfigs(ctx context.Context, configs []types.PayerConfig) ([]types.PayerConfig, error) {
	ret := _m.Called(ctx, configs)

	var r0 []types.PayerConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []types.PayerConfig) ([]types.PayerConfig, error)); ok {
		return rf(ctx, configs)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []types.PayerConfig) []types.PayerConfig); ok {
		r0 = rf(ctx, configs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.PayerConfig)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, []types.PayerConfig) error); ok {
		r1 = rf(ctx, configs)
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
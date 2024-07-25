// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"

	mock "github.com/stretchr/testify/mock"
)

// AlertTierService is an autogenerated mock type for the AlertTierService type
type AlertTierService struct {
	mock.Mock
}

// CheckAccessToAlerts provides a mock function with given fields: ctx, customerID
func (_m *AlertTierService) CheckAccessToAlerts(ctx context.Context, customerID string) (*domain.AccessDeniedError, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *domain.AccessDeniedError
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.AccessDeniedError, error)); ok {
		return rf(ctx, customerID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.AccessDeniedError); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccessDeniedError)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewAlertTierService creates a new instance of AlertTierService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAlertTierService(t interface {
	mock.TestingT
	Cleanup(func())
}) *AlertTierService {
	mock := &AlertTierService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

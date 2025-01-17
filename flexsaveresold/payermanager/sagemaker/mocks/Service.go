// Code generated by mockery v2.38.0. DO NOT EDIT.

package mocksagemakerstate

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// ProcessPayerStatusTransition provides a mock function with given fields: ctx, accountID, customerID, initialStatus, targetStatus
func (_m *Service) ProcessPayerStatusTransition(ctx context.Context, accountID string, customerID string, initialStatus string, targetStatus string) error {
	ret := _m.Called(ctx, accountID, customerID, initialStatus, targetStatus)

	if len(ret) == 0 {
		panic("no return value specified for ProcessPayerStatusTransition")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, string) error); ok {
		r0 = rf(ctx, accountID, customerID, initialStatus, targetStatus)
	} else {
		r0 = ret.Error(0)
	}

	return r0
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

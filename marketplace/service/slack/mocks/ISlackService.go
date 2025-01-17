// Code generated by mockery v2.12.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// ISlackService is an autogenerated mock type for the ISlackService type
type ISlackService struct {
	mock.Mock
}

// PublishEntitlementCancelledMessage provides a mock function with given fields: ctx, domain, billingAccountID
func (_m *ISlackService) PublishEntitlementCancelledMessage(ctx context.Context, domain string, billingAccountID string) error {
	ret := _m.Called(ctx, domain, billingAccountID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, domain, billingAccountID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewISlackService creates a new instance of ISlackService. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewISlackService(t testing.TB) *ISlackService {
	mock := &ISlackService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

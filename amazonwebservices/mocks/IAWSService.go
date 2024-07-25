// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	amazonwebservices "github.com/doitintl/hello/scheduled-tasks/amazonwebservices"

	mock "github.com/stretchr/testify/mock"
)

// IAWSService is an autogenerated mock type for the IAWSService type
type IAWSService struct {
	mock.Mock
}

// CreateAccount provides a mock function with given fields: ctx, customerID, entityID, email, body
func (_m *IAWSService) CreateAccount(ctx context.Context, customerID string, entityID string, email string, body *amazonwebservices.CreateAccountBody) (string, error) {
	ret := _m.Called(ctx, customerID, entityID, email, body)

	if len(ret) == 0 {
		panic("no return value specified for CreateAccount")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *amazonwebservices.CreateAccountBody) (string, error)); ok {
		return rf(ctx, customerID, entityID, email, body)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *amazonwebservices.CreateAccountBody) string); ok {
		r0 = rf(ctx, customerID, entityID, email, body)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, *amazonwebservices.CreateAccountBody) error); ok {
		r1 = rf(ctx, customerID, entityID, email, body)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InviteAccount provides a mock function with given fields: ctx, customerID, entityID, email, body
func (_m *IAWSService) InviteAccount(ctx context.Context, customerID string, entityID string, email string, body *amazonwebservices.InviteAccountBody) (int, error) {
	ret := _m.Called(ctx, customerID, entityID, email, body)

	if len(ret) == 0 {
		panic("no return value specified for InviteAccount")
	}

	var r0 int
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *amazonwebservices.InviteAccountBody) (int, error)); ok {
		return rf(ctx, customerID, entityID, email, body)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, *amazonwebservices.InviteAccountBody) int); ok {
		r0 = rf(ctx, customerID, entityID, email, body)
	} else {
		r0 = ret.Get(0).(int)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, *amazonwebservices.InviteAccountBody) error); ok {
		r1 = rf(ctx, customerID, entityID, email, body)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateAccounts provides a mock function with given fields: ctx
func (_m *IAWSService) UpdateAccounts(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAccounts")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateHandshakes provides a mock function with given fields: ctx
func (_m *IAWSService) UpdateHandshakes(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateHandshakes")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIAWSService creates a new instance of IAWSService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewIAWSService(t interface {
	mock.TestingT
	Cleanup(func())
}) *IAWSService {
	mock := &IAWSService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

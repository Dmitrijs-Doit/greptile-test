// Code generated by mockery v2.37.1. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/doitintl/hello/scheduled-tasks/common"

	mock "github.com/stretchr/testify/mock"
)

// UserDAL is an autogenerated mock type for the UserDAL type
type UserDAL struct {
	mock.Mock
}

// GetUser provides a mock function with given fields: ctx, userID
func (_m *UserDAL) GetUser(ctx context.Context, userID string) (*common.User, error) {
	ret := _m.Called(ctx, userID)

	var r0 *common.User

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (*common.User, error)); ok {
		return rf(ctx, userID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) *common.User); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*common.User)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// HasAttributionsPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasAttributionsPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasBudgetsPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasBudgetsPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasCloudAnalyticsPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasCloudAnalyticsPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasEntitiesPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasEntitiesPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasInvoicesPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasInvoicesPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasLicenseManagePermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasLicenseManagePermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasMetricsPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasMetricsPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// HasUsersPermission provides a mock function with given fields: ctx, user
func (_m *UserDAL) HasUsersPermission(ctx context.Context, user *common.User) bool {
	ret := _m.Called(ctx, user)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context, *common.User) bool); ok {
		r0 = rf(ctx, user)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// NewUserDAL creates a new instance of UserDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewUserDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *UserDAL {
	mock := &UserDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

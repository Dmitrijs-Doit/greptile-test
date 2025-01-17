// Code generated by mockery v2.11.0. DO NOT EDIT.

package mocks

import (
	context "context"

	microsoft "github.com/doitintl/hello/scheduled-tasks/microsoft"
	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// IAccessToken is an autogenerated mock type for the IAccessToken type
type IAccessToken struct {
	mock.Mock
}

// GetAccessToken provides a mock function with given fields:
func (_m *IAccessToken) GetAccessToken() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetAuthenticatedCtx provides a mock function with given fields: ctx
func (_m *IAccessToken) GetAuthenticatedCtx(ctx context.Context) (context.Context, error) {
	ret := _m.Called(ctx)

	var r0 context.Context
	if rf, ok := ret.Get(0).(func(context.Context) context.Context); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(context.Context)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetDomain provides a mock function with given fields:
func (_m *IAccessToken) GetDomain() microsoft.CSPDomain {
	ret := _m.Called()

	var r0 microsoft.CSPDomain
	if rf, ok := ret.Get(0).(func() microsoft.CSPDomain); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(microsoft.CSPDomain)
	}

	return r0
}

// GetExpiresOn provides a mock function with given fields:
func (_m *IAccessToken) GetExpiresOn() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetNotBefore provides a mock function with given fields:
func (_m *IAccessToken) GetNotBefore() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetRefreshToken provides a mock function with given fields:
func (_m *IAccessToken) GetRefreshToken() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetResource provides a mock function with given fields:
func (_m *IAccessToken) GetResource() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetTokenType provides a mock function with given fields:
func (_m *IAccessToken) GetTokenType() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Refresh provides a mock function with given fields:
func (_m *IAccessToken) Refresh() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewIAccessToken creates a new instance of IAccessToken. It also registers a cleanup function to assert the mocks expectations.
func NewIAccessToken(t testing.TB) *IAccessToken {
	mock := &IAccessToken{}

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

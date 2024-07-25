// Code generated by mockery v2.32.0. DO NOT EDIT.

package mocks

import (
	context "context"

	doitemployees "github.com/doitintl/hello/scheduled-tasks/doitemployees"
	mock "github.com/stretchr/testify/mock"
)

// ServiceInterface is an autogenerated mock type for the ServiceInterface type
type ServiceInterface struct {
	mock.Mock
}

// CheckDoiTEmployeeRole provides a mock function with given fields: ctx, roleName, email
func (_m *ServiceInterface) CheckDoiTEmployeeRole(ctx context.Context, roleName string, email string) (bool, error) {
	ret := _m.Called(ctx, roleName, email)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (bool, error)); ok {
		return rf(ctx, roleName, email)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = rf(ctx, roleName, email)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, roleName, email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetByID provides a mock function with given fields: ctx, userID
func (_m *ServiceInterface) GetByID(ctx context.Context, userID string) (*doitemployees.UserDetails, error) {
	ret := _m.Called(ctx, userID)

	var r0 *doitemployees.UserDetails
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*doitemployees.UserDetails, error)); ok {
		return rf(ctx, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *doitemployees.UserDetails); ok {
		r0 = rf(ctx, userID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*doitemployees.UserDetails)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsDoitEmployee provides a mock function with given fields: ctx
func (_m *ServiceInterface) IsDoitEmployee(ctx context.Context) bool {
	ret := _m.Called(ctx)

	var r0 bool
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// SyncRole provides a mock function with given fields: ctx, roleName, users
func (_m *ServiceInterface) SyncRole(ctx context.Context, roleName string, users []string) error {
	ret := _m.Called(ctx, roleName, users)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []string) error); ok {
		r0 = rf(ctx, roleName, users)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewServiceInterface creates a new instance of ServiceInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewServiceInterface(t interface {
	mock.TestingT
	Cleanup(func())
}) *ServiceInterface {
	mock := &ServiceInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

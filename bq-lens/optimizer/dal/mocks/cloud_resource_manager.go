// Code generated by mockery v2.42.3. DO NOT EDIT.

package mocks

import (
	context "context"

	iface "github.com/doitintl/cloudresourcemanager/iface"

	mock "github.com/stretchr/testify/mock"
)

// CloudResourceManager is an autogenerated mock type for the CloudResourceManager type
type CloudResourceManager struct {
	mock.Mock
}

// ListCustomerProjects provides a mock function with given fields: ctx, crm, filter
func (_m *CloudResourceManager) ListCustomerProjects(ctx context.Context, crm iface.CloudResourceManager, filter string) ([]string, error) {
	ret := _m.Called(ctx, crm, filter)

	if len(ret) == 0 {
		panic("no return value specified for ListCustomerProjects")
	}

	var r0 []string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, iface.CloudResourceManager, string) ([]string, error)); ok {
		return rf(ctx, crm, filter)
	}
	if rf, ok := ret.Get(0).(func(context.Context, iface.CloudResourceManager, string) []string); ok {
		r0 = rf(ctx, crm, filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, iface.CloudResourceManager, string) error); ok {
		r1 = rf(ctx, crm, filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewCloudResourceManager creates a new instance of CloudResourceManager. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewCloudResourceManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *CloudResourceManager {
	mock := &CloudResourceManager{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
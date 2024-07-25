// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	iface "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	mock "github.com/stretchr/testify/mock"
)

// Notifier is an autogenerated mock type for the Notifier type
type Notifier struct {
	mock.Mock
}

// NotifyIfNecessary provides a mock function with given fields: ctx, move, eventType
func (_m *Notifier) NotifyIfNecessary(ctx context.Context, move iface.AccountMove, eventType iface.EventType) error {
	ret := _m.Called(ctx, move, eventType)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, iface.AccountMove, iface.EventType) error); ok {
		r0 = rf(ctx, move, eventType)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewNotifier interface {
	mock.TestingT
	Cleanup(func())
}

// NewNotifier creates a new instance of Notifier. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewNotifier(t mockConstructorTestingTNewNotifier) *Notifier {
	mock := &Notifier{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
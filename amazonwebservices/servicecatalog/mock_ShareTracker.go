// Code generated by mockery v2.35.1. DO NOT EDIT.

package servicecatalog

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockShareTracker is an autogenerated mock type for the ShareTracker type
type MockShareTracker struct {
	mock.Mock
}

// ListAccountIDs provides a mock function with given fields: ctx
func (_m *MockShareTracker) ListAccountIDs(ctx context.Context) (map[string]bool, error) {
	ret := _m.Called(ctx)

	var r0 map[string]bool

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) (map[string]bool, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) map[string]bool); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]bool)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SaveAccountID provides a mock function with given fields: ctx, accountID
func (_m *MockShareTracker) SaveAccountID(ctx context.Context, accountID string) error {
	ret := _m.Called(ctx, accountID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, accountID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMockShareTracker creates a new instance of MockShareTracker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockShareTracker(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockShareTracker {
	mock := &MockShareTracker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// PriorityFirestore is an autogenerated mock type for the PriorityFirestore type
type PriorityFirestore struct {
	mock.Mock
}

// HandleAvalaraStatus provides a mock function with given fields: ctx
func (_m *PriorityFirestore) HandleAvalaraStatus(ctx context.Context) (bool, bool, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for HandleAvalaraStatus")
	}

	var r0 bool
	var r1 bool
	var r2 error
	if rf, ok := ret.Get(0).(func(context.Context) (bool, bool, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context) bool); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Get(1).(bool)
	}

	if rf, ok := ret.Get(2).(func(context.Context) error); ok {
		r2 = rf(ctx)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}

// SetAvalaraHealthyStatus provides a mock function with given fields: ctx, healthy
func (_m *PriorityFirestore) SetAvalaraHealthyStatus(ctx context.Context, healthy bool) error {
	ret := _m.Called(ctx, healthy)

	if len(ret) == 0 {
		panic("no return value specified for SetAvalaraHealthyStatus")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, bool) error); ok {
		r0 = rf(ctx, healthy)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewPriorityFirestore creates a new instance of PriorityFirestore. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewPriorityFirestore(t interface {
	mock.TestingT
	Cleanup(func())
}) *PriorityFirestore {
	mock := &PriorityFirestore{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
// Code generated by mockery v2.37.1. DO NOT EDIT.

package mocks

import (
	context "context"

	algolia "github.com/doitintl/hello/scheduled-tasks/algolia"

	mock "github.com/stretchr/testify/mock"
)

// AlgoliaDAL is an autogenerated mock type for the AlgoliaDAL type
type AlgoliaDAL struct {
	mock.Mock
}

// GetConfigFromFirestore provides a mock function with given fields: ctx
func (_m *AlgoliaDAL) GetConfigFromFirestore(ctx context.Context) (*algolia.Config, error) {
	ret := _m.Called(ctx)

	var r0 *algolia.Config

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) (*algolia.Config, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) *algolia.Config); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*algolia.Config)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewAlgoliaDAL creates a new instance of AlgoliaDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAlgoliaDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *AlgoliaDAL {
	mock := &AlgoliaDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

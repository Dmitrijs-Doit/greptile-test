// Code generated by mockery v2.23.1. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/support/domain"
	mock "github.com/stretchr/testify/mock"
)

// SupportServiceInterface is an autogenerated mock type for the SupportServiceInterface type
type SupportServiceInterface struct {
	mock.Mock
}

// ListPlatforms provides a mock function with given fields: ctx
func (_m *SupportServiceInterface) ListPlatforms(ctx context.Context) ([]domain.Platform, error) {
	ret := _m.Called(ctx)

	var r0 []domain.Platform

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) ([]domain.Platform, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) []domain.Platform); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.Platform)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewSupportServiceInterface interface {
	mock.TestingT
	Cleanup(func())
}

// NewSupportServiceInterface creates a new instance of SupportServiceInterface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewSupportServiceInterface(t mockConstructorTestingTNewSupportServiceInterface) *SupportServiceInterface {
	mock := &SupportServiceInterface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

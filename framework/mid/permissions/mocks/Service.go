// Code generated by mockery v2.18.0. DO NOT EDIT.

package mocks

import (
	gin "github.com/gin-gonic/gin"
	mock "github.com/stretchr/testify/mock"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// AssertCacheDisableAccess provides a mock function with given fields: ctx
func (_m *Service) AssertCacheDisableAccess(ctx *gin.Context) (*gin.Context, error) {
	ret := _m.Called(ctx)

	var r0 *gin.Context
	if rf, ok := ret.Get(0).(func(*gin.Context) *gin.Context); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gin.Context)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*gin.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// AssertCacheEnableAccess provides a mock function with given fields: ctx
func (_m *Service) AssertCacheEnableAccess(ctx *gin.Context) (*gin.Context, error) {
	ret := _m.Called(ctx)

	var r0 *gin.Context
	if rf, ok := ret.Get(0).(func(*gin.Context) *gin.Context); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gin.Context)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*gin.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewService interface {
	mock.TestingT
	Cleanup(func())
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewService(t mockConstructorTestingTNewService) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

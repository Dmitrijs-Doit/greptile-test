// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
)

// ISplittingService is an autogenerated mock type for the ISplittingService type
type ISplittingService struct {
	mock.Mock
}

// Split provides a mock function with given fields: splitParams
func (_m *ISplittingService) Split(splitParams domain.BuildSplit) error {
	ret := _m.Called(splitParams)

	var r0 error
	if rf, ok := ret.Get(0).(func(domain.BuildSplit) error); ok {
		r0 = rf(splitParams)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateSplitsReq provides a mock function with given fields: splits
func (_m *ISplittingService) ValidateSplitsReq(splits *[]split.Split) []error {
	ret := _m.Called(splits)

	var r0 []error
	if rf, ok := ret.Get(0).(func(*[]split.Split) []error); ok {
		r0 = rf(splits)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	return r0
}

type mockConstructorTestingTNewISplittingService interface {
	mock.TestingT
	Cleanup(func())
}

// NewISplittingService creates a new instance of ISplittingService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewISplittingService(t mockConstructorTestingTNewISplittingService) *ISplittingService {
	mock := &ISplittingService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
	groupssettings "google.golang.org/api/groupssettings/v1"

	mpa "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa"
)

// IMPAService is an autogenerated mock type for the IMPAService type
type IMPAService struct {
	mock.Mock
}

// AdjustGoogleGroupSettings provides a mock function with given fields: email
func (_m *IMPAService) AdjustGoogleGroupSettings(email string) (*groupssettings.Groups, error) {
	ret := _m.Called(email)

	var r0 *groupssettings.Groups
	if rf, ok := ret.Get(0).(func(string) *groupssettings.Groups); ok {
		r0 = rf(email)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*groupssettings.Groups)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateGoogleGroup provides a mock function with given fields: ctx, req
func (_m *IMPAService) CreateGoogleGroup(ctx context.Context, req *mpa.MPAGoogleGroup) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mpa.MPAGoogleGroup) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreateGoogleGroupCloudTask provides a mock function with given fields: ctx, req
func (_m *IMPAService) CreateGoogleGroupCloudTask(ctx context.Context, req *mpa.MPAGoogleGroup) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mpa.MPAGoogleGroup) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteGoogleGroup provides a mock function with given fields: ctx, req
func (_m *IMPAService) DeleteGoogleGroup(ctx context.Context, req *mpa.MPAGoogleGroup) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mpa.MPAGoogleGroup) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LinkMpaToSauron provides a mock function with given fields: ctx, data
func (_m *IMPAService) LinkMpaToSauron(ctx context.Context, data *mpa.LinkMpaToSauronData) error {
	ret := _m.Called(ctx, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mpa.LinkMpaToSauronData) error); ok {
		r0 = rf(ctx, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateGoogleGroup provides a mock function with given fields: ctx, req
func (_m *IMPAService) UpdateGoogleGroup(ctx context.Context, req *mpa.MPAGoogleGroupUpdate) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mpa.MPAGoogleGroupUpdate) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateMPA provides a mock function with given fields: ctx, req
func (_m *IMPAService) ValidateMPA(ctx context.Context, req *mpa.ValidateMPARequest) error {
	ret := _m.Called(ctx, req)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *mpa.ValidateMPARequest) error); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewIMPAService interface {
	mock.TestingT
	Cleanup(func())
}

// NewIMPAService creates a new instance of IMPAService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewIMPAService(t mockConstructorTestingTNewIMPAService) *IMPAService {
	mock := &IMPAService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	event "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub/proto"

	mock "github.com/stretchr/testify/mock"
)

// DataHubMetadataGCS is an autogenerated mock type for the DataHubMetadataGCS type
type DataHubMetadataGCS struct {
	mock.Mock
}

// DeleteObject provides a mock function with given fields: ctx, objectName
func (_m *DataHubMetadataGCS) DeleteObject(ctx context.Context, objectName string) error {
	ret := _m.Called(ctx, objectName)

	if len(ret) == 0 {
		panic("no return value specified for DeleteObject")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, objectName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ReadEvents provides a mock function with given fields: ctx
func (_m *DataHubMetadataGCS) ReadEvents(ctx context.Context) (map[string][]*event.Event, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for ReadEvents")
	}

	var r0 map[string][]*event.Event
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (map[string][]*event.Event, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) map[string][]*event.Event); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]*event.Event)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewDataHubMetadataGCS creates a new instance of DataHubMetadataGCS. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDataHubMetadataGCS(t interface {
	mock.TestingT
	Cleanup(func())
}) *DataHubMetadataGCS {
	mock := &DataHubMetadataGCS{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

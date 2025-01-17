// Code generated by mockery v2.26.0. DO NOT EDIT.

package mocks

import (
	context "context"

	firestore "cloud.google.com/go/firestore"

	mock "github.com/stretchr/testify/mock"
)

// AttributionGroup is an autogenerated mock type for the AttributionGroup type
type AttributionGroup struct {
	mock.Mock
}

// GetRampPlanEligibleSpendAttributionGroup provides a mock function with given fields: ctx
func (_m *AttributionGroup) GetRampPlanEligibleSpendAttributionGroup(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	ret := _m.Called(ctx)

	var r0 []*firestore.DocumentSnapshot

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) ([]*firestore.DocumentSnapshot, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) []*firestore.DocumentSnapshot); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*firestore.DocumentSnapshot)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewAttributionGroup interface {
	mock.TestingT
	Cleanup(func())
}

// NewAttributionGroup creates a new instance of AttributionGroup. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAttributionGroup(t mockConstructorTestingTNewAttributionGroup) *AttributionGroup {
	mock := &AttributionGroup{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by mockery v2.35.2. DO NOT EDIT.

package mocks

import (
	context "context"

	cloudcommerceprocurement "google.golang.org/api/cloudcommerceprocurement/v1"

	dal "github.com/doitintl/hello/scheduled-tasks/marketplace/dal"

	domain "github.com/doitintl/hello/scheduled-tasks/marketplace/domain"

	mock "github.com/stretchr/testify/mock"
)

// ProcurementDAL is an autogenerated mock type for the ProcurementDAL type
type ProcurementDAL struct {
	mock.Mock
}

// ApproveAccount provides a mock function with given fields: ctx, accountID, reason
func (_m *ProcurementDAL) ApproveAccount(ctx context.Context, accountID string, reason string) error {
	ret := _m.Called(ctx, accountID, reason)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, accountID, reason)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ApproveEntitlement provides a mock function with given fields: ctx, entitlementID
func (_m *ProcurementDAL) ApproveEntitlement(ctx context.Context, entitlementID string) error {
	ret := _m.Called(ctx, entitlementID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, entitlementID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetEntitlement provides a mock function with given fields: ctx, entitlementID
func (_m *ProcurementDAL) GetEntitlement(ctx context.Context, entitlementID string) (*cloudcommerceprocurement.Entitlement, error) {
	ret := _m.Called(ctx, entitlementID)

	var r0 *cloudcommerceprocurement.Entitlement

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (*cloudcommerceprocurement.Entitlement, error)); ok {
		return rf(ctx, entitlementID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) *cloudcommerceprocurement.Entitlement); ok {
		r0 = rf(ctx, entitlementID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*cloudcommerceprocurement.Entitlement)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, entitlementID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListEntitlements provides a mock function with given fields: ctx, filters
func (_m *ProcurementDAL) ListEntitlements(ctx context.Context, filters ...dal.Filter) ([]*cloudcommerceprocurement.Entitlement, error) {
	_va := make([]interface{}, len(filters))
	for _i := range filters {
		_va[_i] = filters[_i]
	}

	var _ca []interface{}
	_ca = append(_ca, ctx)
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 []*cloudcommerceprocurement.Entitlement

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, ...dal.Filter) ([]*cloudcommerceprocurement.Entitlement, error)); ok {
		return rf(ctx, filters...)
	}

	if rf, ok := ret.Get(0).(func(context.Context, ...dal.Filter) []*cloudcommerceprocurement.Entitlement); ok {
		r0 = rf(ctx, filters...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*cloudcommerceprocurement.Entitlement)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, ...dal.Filter) error); ok {
		r1 = rf(ctx, filters...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PublishAccountApprovalRequestEvent provides a mock function with given fields: ctx, payload
func (_m *ProcurementDAL) PublishAccountApprovalRequestEvent(ctx context.Context, payload domain.SubscribePayload) error {
	ret := _m.Called(ctx, payload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.SubscribePayload) error); ok {
		r0 = rf(ctx, payload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RejectAccount provides a mock function with given fields: ctx, accountID, reason
func (_m *ProcurementDAL) RejectAccount(ctx context.Context, accountID string, reason string) error {
	ret := _m.Called(ctx, accountID, reason)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, accountID, reason)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RejectEntitlement provides a mock function with given fields: ctx, entitlementID, reason
func (_m *ProcurementDAL) RejectEntitlement(ctx context.Context, entitlementID string, reason string) error {
	ret := _m.Called(ctx, entitlementID, reason)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, entitlementID, reason)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewProcurementDAL creates a new instance of ProcurementDAL. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewProcurementDAL(t interface {
	mock.TestingT
	Cleanup(func())
}) *ProcurementDAL {
	mock := &ProcurementDAL{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
// Code generated by mockery v2.33.0. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/doitintl/hello/scheduled-tasks/common"

	mock "github.com/stretchr/testify/mock"
)

// GCPMetadata is an autogenerated mock type for the GCPMetadata type
type GCPMetadata struct {
	mock.Mock
}

// UpdateBillingAccountMetadata provides a mock function with given fields: ctx, assetID, billingAccountID, orgs
func (_m *GCPMetadata) UpdateBillingAccountMetadata(ctx context.Context, assetID string, billingAccountID string, orgs []*common.Organization) error {
	ret := _m.Called(ctx, assetID, billingAccountID, orgs)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []*common.Organization) error); ok {
		r0 = rf(ctx, assetID, billingAccountID, orgs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewGCPMetadata creates a new instance of GCPMetadata. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewGCPMetadata(t interface {
	mock.TestingT
	Cleanup(func())
}) *GCPMetadata {
	mock := &GCPMetadata{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

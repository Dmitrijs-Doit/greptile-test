// Code generated by mockery v2.42.1. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/doitintl/hello/scheduled-tasks/common"

	customerapi "github.com/doitintl/customerapi"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"

	iface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"

	mock "github.com/stretchr/testify/mock"
)

// MetadataIface is an autogenerated mock type for the MetadataIface type
type MetadataIface struct {
	mock.Mock
}

// AttributionGroupsMetadata provides a mock function with given fields: ctx, customerID, email
func (_m *MetadataIface) AttributionGroupsMetadata(ctx context.Context, customerID string, email string) ([]*domain.OrgMetadataModel, error) {
	ret := _m.Called(ctx, customerID, email)

	if len(ret) == 0 {
		panic("no return value specified for AttributionGroupsMetadata")
	}

	var r0 []*domain.OrgMetadataModel
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) ([]*domain.OrgMetadataModel, error)); ok {
		return rf(ctx, customerID, email)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []*domain.OrgMetadataModel); ok {
		r0 = rf(ctx, customerID, email)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.OrgMetadataModel)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, email)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExternalAPIGet provides a mock function with given fields: args
func (_m *MetadataIface) ExternalAPIGet(args iface.ExternalAPIGetArgs) (*iface.ExternalAPIGetRes, error) {
	ret := _m.Called(args)

	if len(ret) == 0 {
		panic("no return value specified for ExternalAPIGet")
	}

	var r0 *iface.ExternalAPIGetRes
	var r1 error
	if rf, ok := ret.Get(0).(func(iface.ExternalAPIGetArgs) (*iface.ExternalAPIGetRes, error)); ok {
		return rf(args)
	}
	if rf, ok := ret.Get(0).(func(iface.ExternalAPIGetArgs) *iface.ExternalAPIGetRes); ok {
		r0 = rf(args)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*iface.ExternalAPIGetRes)
		}
	}

	if rf, ok := ret.Get(1).(func(iface.ExternalAPIGetArgs) error); ok {
		r1 = rf(args)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExternalAPIList provides a mock function with given fields: args
func (_m *MetadataIface) ExternalAPIList(args iface.ExternalAPIListArgs) (iface.ExternalAPIListRes, error) {
	ret := _m.Called(args)

	if len(ret) == 0 {
		panic("no return value specified for ExternalAPIList")
	}

	var r0 iface.ExternalAPIListRes
	var r1 error
	if rf, ok := ret.Get(0).(func(iface.ExternalAPIListArgs) (iface.ExternalAPIListRes, error)); ok {
		return rf(args)
	}
	if rf, ok := ret.Get(0).(func(iface.ExternalAPIListArgs) iface.ExternalAPIListRes); ok {
		r0 = rf(args)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(iface.ExternalAPIListRes)
		}
	}

	if rf, ok := ret.Get(1).(func(iface.ExternalAPIListArgs) error); ok {
		r1 = rf(args)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ExternalAPIListWithFilters provides a mock function with given fields: args, req
func (_m *MetadataIface) ExternalAPIListWithFilters(args iface.ExternalAPIListArgs, req *customerapi.Request) (*domain.DimensionsExternalAPIList, error) {
	ret := _m.Called(args, req)

	if len(ret) == 0 {
		panic("no return value specified for ExternalAPIListWithFilters")
	}

	var r0 *domain.DimensionsExternalAPIList
	var r1 error
	if rf, ok := ret.Get(0).(func(iface.ExternalAPIListArgs, *customerapi.Request) (*domain.DimensionsExternalAPIList, error)); ok {
		return rf(args, req)
	}
	if rf, ok := ret.Get(0).(func(iface.ExternalAPIListArgs, *customerapi.Request) *domain.DimensionsExternalAPIList); ok {
		r0 = rf(args, req)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.DimensionsExternalAPIList)
		}
	}

	if rf, ok := ret.Get(1).(func(iface.ExternalAPIListArgs, *customerapi.Request) error); ok {
		r1 = rf(args, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateAWSAllCustomersMetadata provides a mock function with given fields: ctx
func (_m *MetadataIface) UpdateAWSAllCustomersMetadata(ctx context.Context) ([]error, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAWSAllCustomersMetadata")
	}

	var r0 []error
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]error, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []error); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateAWSCustomerMetadata provides a mock function with given fields: ctx, customerID, orgs
func (_m *MetadataIface) UpdateAWSCustomerMetadata(ctx context.Context, customerID string, orgs []*common.Organization) error {
	ret := _m.Called(ctx, customerID, orgs)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAWSCustomerMetadata")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []*common.Organization) error); ok {
		r0 = rf(ctx, customerID, orgs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateAzureAllCustomersMetadata provides a mock function with given fields: ctx
func (_m *MetadataIface) UpdateAzureAllCustomersMetadata(ctx context.Context) ([]error, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAzureAllCustomersMetadata")
	}

	var r0 []error
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]error, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []error); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateAzureCustomerMetadata provides a mock function with given fields: ctx, customerID
func (_m *MetadataIface) UpdateAzureCustomerMetadata(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for UpdateAzureCustomerMetadata")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateBQLensAllCustomersMetadata provides a mock function with given fields: ctx
func (_m *MetadataIface) UpdateBQLensAllCustomersMetadata(ctx context.Context) ([]error, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateBQLensAllCustomersMetadata")
	}

	var r0 []error
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]error, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []error); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateBQLensCustomerMetadata provides a mock function with given fields: ctx, customerID
func (_m *MetadataIface) UpdateBQLensCustomerMetadata(ctx context.Context, customerID string) error {
	ret := _m.Called(ctx, customerID)

	if len(ret) == 0 {
		panic("no return value specified for UpdateBQLensCustomerMetadata")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, customerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateGCPBillingAccountMetadata provides a mock function with given fields: ctx, assetID, billingAccountID, orgs
func (_m *MetadataIface) UpdateGCPBillingAccountMetadata(ctx context.Context, assetID string, billingAccountID string, orgs []*common.Organization) error {
	ret := _m.Called(ctx, assetID, billingAccountID, orgs)

	if len(ret) == 0 {
		panic("no return value specified for UpdateGCPBillingAccountMetadata")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, []*common.Organization) error); ok {
		r0 = rf(ctx, assetID, billingAccountID, orgs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateDataHubMetadata provides a mock function with given fields: ctx
func (_m *MetadataIface) UpdateDataHubMetadata(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for UpdateDataHubMetadata")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMetadataIface creates a new instance of MetadataIface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMetadataIface(t interface {
	mock.TestingT
	Cleanup(func())
}) *MetadataIface {
	mock := &MetadataIface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

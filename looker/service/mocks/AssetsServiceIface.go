// Code generated by mockery v2.16.0. DO NOT EDIT.

package mocks

import (
	context "context"

	bqutils "github.com/doitintl/hello/scheduled-tasks/bqutils"

	domain "github.com/doitintl/hello/scheduled-tasks/looker/domain"

	gin "github.com/gin-gonic/gin"

	mock "github.com/stretchr/testify/mock"

	pkg "github.com/doitintl/firestore/pkg"

	schema "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schema"

	time "time"
)

// AssetsServiceIface is an autogenerated mock type for the AssetsServiceIface type
type AssetsServiceIface struct {
	mock.Mock
}

// CreateLookerRows provides a mock function with given fields: ctx, contracts, updateTableInterval
func (_m *AssetsServiceIface) CreateLookerRows(ctx *gin.Context, contracts []*pkg.Contract, updateTableInterval []time.Time) (map[time.Time][]schema.BillingRow, error) {
	ret := _m.Called(ctx, contracts, updateTableInterval)

	var r0 map[time.Time][]schema.BillingRow
	if rf, ok := ret.Get(0).(func(*gin.Context, []*pkg.Contract, []time.Time) map[time.Time][]schema.BillingRow); ok {
		r0 = rf(ctx, contracts, updateTableInterval)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[time.Time][]schema.BillingRow)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*gin.Context, []*pkg.Contract, []time.Time) error); ok {
		r1 = rf(ctx, contracts, updateTableInterval)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTableLoaderPayload provides a mock function with given fields: ctx, billingRows
func (_m *AssetsServiceIface) GetTableLoaderPayload(ctx context.Context, billingRows []schema.BillingRow) (*bqutils.BigQueryTableLoaderParams, error) {
	ret := _m.Called(ctx, billingRows)

	var r0 *bqutils.BigQueryTableLoaderParams
	if rf, ok := ret.Get(0).(func(context.Context, []schema.BillingRow) *bqutils.BigQueryTableLoaderParams); ok {
		r0 = rf(ctx, billingRows)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*bqutils.BigQueryTableLoaderParams)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, []schema.BillingRow) error); ok {
		r1 = rf(ctx, billingRows)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// LoadLookerContractsToBQ provides a mock function with given fields: ctx, request
func (_m *AssetsServiceIface) LoadLookerContractsToBQ(ctx *gin.Context, request domain.UpdateTableInterval) error {
	ret := _m.Called(ctx, request)

	var r0 error
	if rf, ok := ret.Get(0).(func(*gin.Context, domain.UpdateTableInterval) error); ok {
		r0 = rf(ctx, request)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// LookerBigQueryTableLoader provides a mock function with given fields: ctx, loadAttributes, partition, tableExists
func (_m *AssetsServiceIface) LookerBigQueryTableLoader(ctx context.Context, loadAttributes bqutils.BigQueryTableLoaderParams, partition time.Time, tableExists bool) error {
	ret := _m.Called(ctx, loadAttributes, partition, tableExists)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, bqutils.BigQueryTableLoaderParams, time.Time, bool) error); ok {
		r0 = rf(ctx, loadAttributes, partition, tableExists)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewAssetsServiceIface interface {
	mock.TestingT
	Cleanup(func())
}

// NewAssetsServiceIface creates a new instance of AssetsServiceIface. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewAssetsServiceIface(t mockConstructorTestingTNewAssetsServiceIface) *AssetsServiceIface {
	mock := &AssetsServiceIface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
// Code generated by mockery v2.25.1. DO NOT EDIT.

package mocks

import (
	context "context"

	firestore "cloud.google.com/go/firestore"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/domain/cost_allocation"

	mock "github.com/stretchr/testify/mock"
)

// CostAllocations is an autogenerated mock type for the CostAllocations type
type CostAllocations struct {
	mock.Mock
}

// CommitCostAllocations provides a mock function with given fields: ctx, newValues
func (_m *CostAllocations) CommitCostAllocations(ctx context.Context, newValues *map[string]domain.CostAllocation) []error {
	ret := _m.Called(ctx, newValues)

	var r0 []error
	if rf, ok := ret.Get(0).(func(context.Context, *map[string]domain.CostAllocation) []error); ok {
		r0 = rf(ctx, newValues)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	return r0
}

// GetAllCostAllocationDocs provides a mock function with given fields: ctx
func (_m *CostAllocations) GetAllCostAllocationDocs(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
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

// GetAllEnabledCostAllocation provides a mock function with given fields: ctx
func (_m *CostAllocations) GetAllEnabledCostAllocation(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
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

// GetCostAllocation provides a mock function with given fields: ctx, customerID
func (_m *CostAllocations) GetCostAllocation(ctx context.Context, customerID string) (*domain.CostAllocation, error) {
	ret := _m.Called(ctx, customerID)

	var r0 *domain.CostAllocation

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.CostAllocation, error)); ok {
		return rf(ctx, customerID)
	}

	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.CostAllocation); ok {
		r0 = rf(ctx, customerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.CostAllocation)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, customerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetCostAllocationConfig provides a mock function with given fields: ctx
func (_m *CostAllocations) GetCostAllocationConfig(ctx context.Context) (*domain.CostAllocationConfig, error) {
	ret := _m.Called(ctx)

	var r0 *domain.CostAllocationConfig

	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context) (*domain.CostAllocationConfig, error)); ok {
		return rf(ctx)
	}

	if rf, ok := ret.Get(0).(func(context.Context) *domain.CostAllocationConfig); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.CostAllocationConfig)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UpdateCostAllocation provides a mock function with given fields: ctx, customerID, newValue
func (_m *CostAllocations) UpdateCostAllocation(ctx context.Context, customerID string, newValue *domain.CostAllocation) error {
	ret := _m.Called(ctx, customerID, newValue)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *domain.CostAllocation) error); ok {
		r0 = rf(ctx, customerID, newValue)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// UpdateCostAllocationConfig provides a mock function with given fields: ctx, newValue
func (_m *CostAllocations) UpdateCostAllocationConfig(ctx context.Context, newValue *domain.CostAllocationConfig) error {
	ret := _m.Called(ctx, newValue)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *domain.CostAllocationConfig) error); ok {
		r0 = rf(ctx, newValue)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewCostAllocations interface {
	mock.TestingT
	Cleanup(func())
}

// NewCostAllocations creates a new instance of CostAllocations. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewCostAllocations(t mockConstructorTestingTNewCostAllocations) *CostAllocations {
	mock := &CostAllocations{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

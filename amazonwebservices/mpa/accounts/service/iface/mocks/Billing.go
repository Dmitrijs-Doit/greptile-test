// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	iface "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	mock "github.com/stretchr/testify/mock"
)

// Billing is an autogenerated mock type for the Billing type
type Billing struct {
	mock.Mock
}

// GetCoveredUsage provides a mock function with given fields: ctx, accountID, from
func (_m *Billing) GetCoveredUsage(ctx context.Context, accountID string, from iface.Payer) (iface.CoveredUsage, error) {
	ret := _m.Called(ctx, accountID, from)

	var r0 iface.CoveredUsage
	if rf, ok := ret.Get(0).(func(context.Context, string, iface.Payer) iface.CoveredUsage); ok {
		r0 = rf(ctx, accountID, from)
	} else {
		r0 = ret.Get(0).(iface.CoveredUsage)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, iface.Payer) error); ok {
		r1 = rf(ctx, accountID, from)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTNewBilling interface {
	mock.TestingT
	Cleanup(func())
}

// NewBilling creates a new instance of Billing. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBilling(t mockConstructorTestingTNewBilling) *Billing {
	mock := &Billing{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

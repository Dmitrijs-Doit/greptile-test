// Code generated by mockery v2.12.0. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/marketplace/domain"

	mock "github.com/stretchr/testify/mock"

	testing "testing"
)

// MarketplaceIface is an autogenerated mock type for the MarketplaceIface type
type MarketplaceIface struct {
	mock.Mock
}

// ApproveAccount provides a mock function with given fields: ctx, accountID, email
func (_m *MarketplaceIface) ApproveAccount(ctx context.Context, accountID string, email string) error {
	ret := _m.Called(ctx, accountID, email)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, accountID, email)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ApproveEntitlement provides a mock function with given fields: ctx, entitlementID, email, doitEmployee, approveFlexsaveProduct
func (_m *MarketplaceIface) ApproveEntitlement(ctx context.Context, entitlementID string, email string, doitEmployee bool, approveFlexsaveProduct bool) error {
	ret := _m.Called(ctx, entitlementID, email, doitEmployee, approveFlexsaveProduct)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, bool, bool) error); ok {
		r0 = rf(ctx, entitlementID, email, doitEmployee, approveFlexsaveProduct)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CancelEntitlement provides a mock function with given fields: ctx, entitlementID
func (_m *MarketplaceIface) HandleCancelledEntitlement(ctx context.Context, entitlementID string) error {
	ret := _m.Called(ctx, entitlementID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, entitlementID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetAccount provides a mock function with given fields: ctx, accountID
func (_m *MarketplaceIface) GetAccount(ctx context.Context, accountID string) (*domain.AccountFirestore, error) {
	ret := _m.Called(ctx, accountID)

	var r0 *domain.AccountFirestore
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.AccountFirestore); ok {
		r0 = rf(ctx, accountID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.AccountFirestore)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, accountID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PopulateBillingAccounts provides a mock function with given fields: ctx, populateBillingAccounts
func (_m *MarketplaceIface) PopulateBillingAccounts(ctx context.Context, populateBillingAccounts domain.PopulateBillingAccounts) (domain.PopulateBillingAccountsResult, error) {
	ret := _m.Called(ctx, populateBillingAccounts)

	var r0 domain.PopulateBillingAccountsResult
	if rf, ok := ret.Get(0).(func(context.Context, domain.PopulateBillingAccounts) domain.PopulateBillingAccountsResult); ok {
		r0 = rf(ctx, populateBillingAccounts)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(domain.PopulateBillingAccountsResult)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, domain.PopulateBillingAccounts) error); ok {
		r1 = rf(ctx, populateBillingAccounts)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RejectAccount provides a mock function with given fields: ctx, accountID, email
func (_m *MarketplaceIface) RejectAccount(ctx context.Context, accountID string, email string) error {
	ret := _m.Called(ctx, accountID, email)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, accountID, email)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RejectEntitlement provides a mock function with given fields: ctx, entitlementID, email
func (_m *MarketplaceIface) RejectEntitlement(ctx context.Context, entitlementID string, email string) error {
	ret := _m.Called(ctx, entitlementID, email)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, entitlementID, email)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// StandaloneApprove provides a mock function with given fields: ctx, customerID, billingAccountID
func (_m *MarketplaceIface) StandaloneApprove(ctx context.Context, customerID string, billingAccountID string) error {
	ret := _m.Called(ctx, customerID, billingAccountID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, customerID, billingAccountID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Subscribe provides a mock function with given fields: ctx, subscribePayload
func (_m *MarketplaceIface) Subscribe(ctx context.Context, subscribePayload domain.SubscribePayload) error {
	ret := _m.Called(ctx, subscribePayload)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.SubscribePayload) error); ok {
		r0 = rf(ctx, subscribePayload)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewMarketplaceIface creates a new instance of MarketplaceIface. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewMarketplaceIface(t testing.TB) *MarketplaceIface {
	mock := &MarketplaceIface{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
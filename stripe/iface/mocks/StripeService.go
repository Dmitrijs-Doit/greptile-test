// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	common "github.com/doitintl/hello/scheduled-tasks/common"

	mock "github.com/stretchr/testify/mock"

	service "github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

// StripeService is an autogenerated mock type for the StripeService type
type StripeService struct {
	mock.Mock
}

// AutomaticPayments provides a mock function with given fields: ctx, input
func (_m *StripeService) AutomaticPayments(ctx context.Context, input service.AutomaticPaymentsInput) error {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for AutomaticPayments")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, service.AutomaticPaymentsInput) error); ok {
		r0 = rf(ctx, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// AutomaticPaymentsEntityWorker provides a mock function with given fields: ctx, input
func (_m *StripeService) AutomaticPaymentsEntityWorker(ctx context.Context, input service.AutomaticPaymentsEntityWorkerInput) error {
	ret := _m.Called(ctx, input)

	if len(ret) == 0 {
		panic("no return value specified for AutomaticPaymentsEntityWorker")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, service.AutomaticPaymentsEntityWorkerInput) error); ok {
		r0 = rf(ctx, input)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// CreatePMSetupIntentForEntity provides a mock function with given fields: ctx, entity, newEntity
func (_m *StripeService) CreatePMSetupIntentForEntity(ctx context.Context, entity *common.Entity, newEntity bool) (service.SetupIntentClientSecret, error) {
	ret := _m.Called(ctx, entity, newEntity)

	if len(ret) == 0 {
		panic("no return value specified for CreatePMSetupIntentForEntity")
	}

	var r0 service.SetupIntentClientSecret
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity, bool) (service.SetupIntentClientSecret, error)); ok {
		return rf(ctx, entity, newEntity)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity, bool) service.SetupIntentClientSecret); ok {
		r0 = rf(ctx, entity, newEntity)
	} else {
		r0 = ret.Get(0).(service.SetupIntentClientSecret)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *common.Entity, bool) error); ok {
		r1 = rf(ctx, entity, newEntity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateSetupSessionForEntity provides a mock function with given fields: ctx, entity, urls
func (_m *StripeService) CreateSetupSessionForEntity(ctx context.Context, entity *common.Entity, urls service.SetupSessionURLs) (service.SetupSession, error) {
	ret := _m.Called(ctx, entity, urls)

	if len(ret) == 0 {
		panic("no return value specified for CreateSetupSessionForEntity")
	}

	var r0 service.SetupSession
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity, service.SetupSessionURLs) (service.SetupSession, error)); ok {
		return rf(ctx, entity, urls)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity, service.SetupSessionURLs) service.SetupSession); ok {
		r0 = rf(ctx, entity, urls)
	} else {
		r0 = ret.Get(0).(service.SetupSession)
	}

	if rf, ok := ret.Get(1).(func(context.Context, *common.Entity, service.SetupSessionURLs) error); ok {
		r1 = rf(ctx, entity, urls)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DetachPaymentMethod provides a mock function with given fields: ctx, input, entity
func (_m *StripeService) DetachPaymentMethod(ctx context.Context, input service.PaymentMethodBody, entity *common.Entity) error {
	ret := _m.Called(ctx, input, entity)

	if len(ret) == 0 {
		panic("no return value specified for DetachPaymentMethod")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, service.PaymentMethodBody, *common.Entity) error); ok {
		r0 = rf(ctx, input, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCreditCardProcessingFee provides a mock function with given fields: ctx, customerID, entity, amount
func (_m *StripeService) GetCreditCardProcessingFee(ctx context.Context, customerID string, entity *common.Entity, amount int64) (*service.ProcessingFee, error) {
	ret := _m.Called(ctx, customerID, entity, amount)

	if len(ret) == 0 {
		panic("no return value specified for GetCreditCardProcessingFee")
	}

	var r0 *service.ProcessingFee
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, *common.Entity, int64) (*service.ProcessingFee, error)); ok {
		return rf(ctx, customerID, entity, amount)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, *common.Entity, int64) *service.ProcessingFee); ok {
		r0 = rf(ctx, customerID, entity, amount)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*service.ProcessingFee)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, *common.Entity, int64) error); ok {
		r1 = rf(ctx, customerID, entity, amount)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPaymentMethods provides a mock function with given fields: ctx, entity
func (_m *StripeService) GetPaymentMethods(ctx context.Context, entity *common.Entity) ([]*service.PaymentMethod, error) {
	ret := _m.Called(ctx, entity)

	if len(ret) == 0 {
		panic("no return value specified for GetPaymentMethods")
	}

	var r0 []*service.PaymentMethod
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity) ([]*service.PaymentMethod, error)); ok {
		return rf(ctx, entity)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity) []*service.PaymentMethod); ok {
		r0 = rf(ctx, entity)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*service.PaymentMethod)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *common.Entity) error); ok {
		r1 = rf(ctx, entity)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PatchPaymentMethod provides a mock function with given fields: ctx, input, entity
func (_m *StripeService) PatchPaymentMethod(ctx context.Context, input service.PaymentMethodBody, entity *common.Entity) error {
	ret := _m.Called(ctx, input, entity)

	if len(ret) == 0 {
		panic("no return value specified for PatchPaymentMethod")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, service.PaymentMethodBody, *common.Entity) error); ok {
		r0 = rf(ctx, input, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PayInvoice provides a mock function with given fields: ctx, input, entity
func (_m *StripeService) PayInvoice(ctx context.Context, input service.PayInvoiceInput, entity *common.Entity) error {
	ret := _m.Called(ctx, input, entity)

	if len(ret) == 0 {
		panic("no return value specified for PayInvoice")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, service.PayInvoiceInput, *common.Entity) error); ok {
		r0 = rf(ctx, input, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// PaymentsDigest provides a mock function with given fields: ctx
func (_m *StripeService) PaymentsDigest(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for PaymentsDigest")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SyncCustomerData provides a mock function with given fields: ctx, entity
func (_m *StripeService) SyncCustomerData(ctx context.Context, entity *common.Entity) error {
	ret := _m.Called(ctx, entity)

	if len(ret) == 0 {
		panic("no return value specified for SyncCustomerData")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *common.Entity) error); ok {
		r0 = rf(ctx, entity)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ValidateUserPermissions provides a mock function with given fields: ctx, customerID, userID
func (_m *StripeService) ValidateUserPermissions(ctx context.Context, customerID string, userID string) (bool, error) {
	ret := _m.Called(ctx, customerID, userID)

	if len(ret) == 0 {
		panic("no return value specified for ValidateUserPermissions")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (bool, error)); ok {
		return rf(ctx, customerID, userID)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) bool); ok {
		r0 = rf(ctx, customerID, userID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, customerID, userID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewStripeService creates a new instance of StripeService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewStripeService(t interface {
	mock.TestingT
	Cleanup(func())
}) *StripeService {
	mock := &StripeService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
// Code generated by mockery v2.40.3. DO NOT EDIT.

package mocks

import (
	context "context"

	domain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	mock "github.com/stretchr/testify/mock"

	priority "github.com/doitintl/hello/scheduled-tasks/priority"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// ApproveInvoice provides a mock function with given fields: ctx, pid
func (_m *Service) ApproveInvoice(ctx context.Context, pid domain.PriorityInvoiceIdentifier) (string, error) {
	ret := _m.Called(ctx, pid)

	if len(ret) == 0 {
		panic("no return value specified for ApproveInvoice")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.PriorityInvoiceIdentifier) (string, error)); ok {
		return rf(ctx, pid)
	}
	if rf, ok := ret.Get(0).(func(context.Context, domain.PriorityInvoiceIdentifier) string); ok {
		r0 = rf(ctx, pid)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, domain.PriorityInvoiceIdentifier) error); ok {
		r1 = rf(ctx, pid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CloseInvoice provides a mock function with given fields: ctx, pid
func (_m *Service) CloseInvoice(ctx context.Context, pid domain.PriorityInvoiceIdentifier) (string, error) {
	ret := _m.Called(ctx, pid)

	if len(ret) == 0 {
		panic("no return value specified for CloseInvoice")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.PriorityInvoiceIdentifier) (string, error)); ok {
		return rf(ctx, pid)
	}
	if rf, ok := ret.Get(0).(func(context.Context, domain.PriorityInvoiceIdentifier) string); ok {
		r0 = rf(ctx, pid)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, domain.PriorityInvoiceIdentifier) error); ok {
		r1 = rf(ctx, pid)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// CreateInvoice provides a mock function with given fields: ctx, req
func (_m *Service) CreateInvoice(ctx context.Context, req domain.Invoice) (domain.Invoice, error) {
	ret := _m.Called(ctx, req)

	if len(ret) == 0 {
		panic("no return value specified for CreateInvoice")
	}

	var r0 domain.Invoice
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.Invoice) (domain.Invoice, error)); ok {
		return rf(ctx, req)
	}
	if rf, ok := ret.Get(0).(func(context.Context, domain.Invoice) domain.Invoice); ok {
		r0 = rf(ctx, req)
	} else {
		r0 = ret.Get(0).(domain.Invoice)
	}

	if rf, ok := ret.Get(1).(func(context.Context, domain.Invoice) error); ok {
		r1 = rf(ctx, req)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// DeleteInvoice provides a mock function with given fields: ctx, pid
func (_m *Service) DeleteInvoice(ctx context.Context, pid domain.PriorityInvoiceIdentifier) error {
	ret := _m.Called(ctx, pid)

	if len(ret) == 0 {
		panic("no return value specified for DeleteInvoice")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.PriorityInvoiceIdentifier) error); ok {
		r0 = rf(ctx, pid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DeleteReceipt provides a mock function with given fields: ctx, priorityCompany, receiptID
func (_m *Service) DeleteReceipt(ctx context.Context, priorityCompany string, receiptID string) error {
	ret := _m.Called(ctx, priorityCompany, receiptID)

	if len(ret) == 0 {
		panic("no return value specified for DeleteReceipt")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, priorityCompany, receiptID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ListCustomerReceipts provides a mock function with given fields: ctx, priorityCompany, customerName
func (_m *Service) ListCustomerReceipts(ctx context.Context, priorityCompany priority.CompanyCode, customerName string) (domain.TInvoices, error) {
	ret := _m.Called(ctx, priorityCompany, customerName)

	if len(ret) == 0 {
		panic("no return value specified for ListCustomerReceipts")
	}

	var r0 domain.TInvoices
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, priority.CompanyCode, string) (domain.TInvoices, error)); ok {
		return rf(ctx, priorityCompany, customerName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, priority.CompanyCode, string) domain.TInvoices); ok {
		r0 = rf(ctx, priorityCompany, customerName)
	} else {
		r0 = ret.Get(0).(domain.TInvoices)
	}

	if rf, ok := ret.Get(1).(func(context.Context, priority.CompanyCode, string) error); ok {
		r1 = rf(ctx, priorityCompany, customerName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PrintInvoice provides a mock function with given fields: ctx, pid
func (_m *Service) PrintInvoice(ctx context.Context, pid domain.PriorityInvoiceIdentifier) error {
	ret := _m.Called(ctx, pid)

	if len(ret) == 0 {
		panic("no return value specified for PrintInvoice")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.PriorityInvoiceIdentifier) error); ok {
		r0 = rf(ctx, pid)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// SyncCustomers provides a mock function with given fields: ctx
func (_m *Service) SyncCustomers(ctx context.Context) error {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for SyncCustomers")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewService(t interface {
	mock.TestingT
	Cleanup(func())
}) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

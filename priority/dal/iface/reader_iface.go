//go:generate mockery --output=../mocks --name=Reader --filename=reader_iface.go
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

type Reader interface {
	GetCustomers(ctx context.Context, priorityCompany priority.CompanyCode) (priorityDomain.Customers, error)
	GetAccountsReceivables(ctx context.Context, priorityCompany priority.CompanyCode) (priorityDomain.AccountsReceivable, error)
	GetInvoiceItems(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string) ([]priorityDomain.InvoiceItem, error)
	GetCustomerDetails(ctx context.Context, priorityCompany, customerName string) (priorityDomain.CustomerDetails, error)
	GetCustomerCountryName(ctx context.Context, priorityCompany, customerName string) (string, error)
	ListCustomerReceipts(ctx context.Context, priorityCompany priority.CompanyCode, customerName string) (priorityDomain.TInvoices, error)
	GetInvoiceID(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string) (uint64, error)
	GetInvoice(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string) (priorityDomain.Invoice, error)
	FilterInvoices(ctx context.Context, priorityCompany, invoicesType, filter string) ([]priorityDomain.Invoice, error)
	PingAvalaraTax(ctx context.Context) bool
}

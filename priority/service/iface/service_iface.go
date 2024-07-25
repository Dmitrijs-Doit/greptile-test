//go:generate mockery --output=../mocks --name Service --filename service_iface.go
package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

type Service interface {
	// SyncCustomers syncs all Priority companies to firestore entities collection.
	SyncCustomers(ctx context.Context) error

	// ListCustomerReceipts lists all final receipts for a customer.
	ListCustomerReceipts(ctx context.Context, priorityCompany priority.CompanyCode, customerName string) (priorityDomain.TInvoices, error)

	// CreateInvoice creates an invoice and returns a new invoice. If the invoice requires an Avalara processing - it executes it as well.
	CreateInvoice(ctx context.Context, req priorityDomain.Invoice) (priorityDomain.Invoice, error)

	// ApproveInvoice approves an invoice.
	ApproveInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) (string, error)

	// CloseInvoice closes an invoice.
	CloseInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) (string, error)

	// DeleteInvoice deletes an invoice.
	DeleteInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) error

	// PrintInvoice prints an invoice in the required format. In this method, the final invoice number should be injected.
	PrintInvoice(ctx context.Context, pid priorityDomain.PriorityInvoiceIdentifier) error

	// DeleteReceipt deletes an receipt.
	DeleteReceipt(ctx context.Context, priorityCompany, receiptID string) error
}

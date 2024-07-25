//go:generate mockery --output=../mocks --name=Writer --filename=writer_iface.go
package iface

import (
	"context"

	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

type Writer interface {
	CreateInvoice(ctx context.Context, invoiceType string, invoice priorityDomain.Invoice) (string, error)
	UpdateInvoiceStatus(ctx context.Context, priorityCompany, invoiceType, invoiceNumber, status string) (priorityDomain.UpdateInvoiceStatusResponse, error)
	UpdateReceiptStatus(ctx context.Context, priorityCompany, receiptID, status string) error
	UpdateAvalaraTax(ctx context.Context, priorityCompany string, invoiceID uint64) error
	CloseInvoice(ctx context.Context, priorityCompany string, invoiceID uint64) error
	DeleteInvoice(ctx context.Context, priorityCompany string, invoiceID uint64) error
	NullifyInvoiceItems(ctx context.Context, priorityCompany, invoiceType, invoiceNumber string, invoiceItems []priorityDomain.InvoiceItem) error
	PrintInvoice(ctx context.Context, priorityCompany, customerCountryName, invoiceType string, invoiceID uint64) error
}

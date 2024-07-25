package invoices

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/invoicing"
)

//go:generate mockery --output=./mocks --all
type InvoicesDAL interface {
	ListInvoices(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		limit int,
	) ([]*invoicing.Invoice, error)
}

package invoices

import (
	"context"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
)

const (
	invoicesCollection = "invoices"
)

type InvoicesFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewInvoicesFirestore(ctx context.Context, projectID string) (*InvoicesFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewInvoicesFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewInvoicesFirestoreWithClient(fun connection.FirestoreFromContextFun) *InvoicesFirestore {
	return &InvoicesFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *InvoicesFirestore) ListInvoices(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	limit int,
) ([]*invoicing.Invoice, error) {
	query := d.
		firestoreClientFun(ctx).
		Collection(invoicesCollection).
		Where("customer", "==", customerRef)

	if limit > 0 {
		query = query.Limit(limit)
	}

	iter := query.Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	invoices := make([]*invoicing.Invoice, len(snaps))

	for i, snap := range snaps {
		var invoice invoicing.Invoice
		if err := snap.DataTo(&invoice); err != nil {
			return nil, err
		}

		invoices[i] = &invoice
	}

	return invoices, nil
}

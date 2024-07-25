package invoicing

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

type InvoiceAdjustment struct {
	Description   string                      `firestore:"description"`
	Details       string                      `firestore:"details"`
	Type          string                      `firestore:"type"`
	Customer      *firestore.DocumentRef      `firestore:"customer"`
	Entity        *firestore.DocumentRef      `firestore:"entity"`
	InvoiceMonths []time.Time                 `firestore:"invoiceMonths"`
	Currency      string                      `firestore:"currency"`
	Amount        float64                     `firestore:"amount"`
	Metadata      map[string]interface{}      `firestore:"metadata"`
	Snapshot      *firestore.DocumentSnapshot `firestore:"-"`
}

func getCustomerInvoiceAdjustments(ctx context.Context, customerRef *firestore.DocumentRef, invoiceType string, invoiceMonth time.Time) ([]*InvoiceAdjustment, error) {
	invoiceAdjustments := make([]*InvoiceAdjustment, 0)

	docSnaps, err := customerRef.Collection("customerInvoiceAdjustments").
		Where("type", "==", invoiceType).
		Where("invoiceMonths", "array-contains", time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var invoiceAdjustment InvoiceAdjustment
		if err := docSnap.DataTo(&invoiceAdjustment); err != nil {
			return nil, err
		}

		invoiceAdjustment.Snapshot = docSnap
		invoiceAdjustments = append(invoiceAdjustments, &invoiceAdjustment)
	}

	return invoiceAdjustments, nil
}

package dal

import (
	"context"
	"fmt"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/utils"
)

const (
	fieldInvoiceStripePaymentIntents = "stripePaymentIntents"
)

// StripeFirestore is used to interact with stripe data stored on Firestore.
type StripeFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	integrationDocID   string
}

// NewStripeFirestore returns a new StripeFirestore instance with given project id.
func NewStripeFirestore(ctx context.Context, projectID string, integrationDocID string) (*StripeFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	stripeFirestore := NewStripeFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		}, integrationDocID)

	return stripeFirestore, nil
}

// NewStripeFirestoreWithClient returns a new StripeFirestore using given client.
func NewStripeFirestoreWithClient(fun connection.FirestoreFromContextFun, integrationDocID string) *StripeFirestore {
	return &StripeFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		integrationDocID:   integrationDocID,
	}
}

func (d *StripeFirestore) getStripeIntegrationsRef(ctx context.Context) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection("integrations").Doc(d.integrationDocID)
}

func (d *StripeFirestore) getInvoiceRef(ctx context.Context, invoiceDocID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection("invoices").Doc(invoiceDocID)
}

// GetPaymentRef returns the firestore document reference for the given payment ID.
func (d *StripeFirestore) GetPaymentRef(ctx context.Context, paymentID string) *firestore.DocumentRef {
	return d.getStripeIntegrationsRef(ctx).Collection("stripePayments").Doc(paymentID)
}

// IsCreditCardPaymentDisabled returns true if credit card payments are disabled.
func (d *StripeFirestore) IsPaymentTypeDisabled(ctx context.Context, paymentMethodType stripe.PaymentMethodType) (bool, error) {
	docSnap, err := d.firestoreClientFun(ctx).Collection("app").Doc("stripe").Get(ctx)
	if err != nil {
		return true, err
	}

	var lockField string

	switch paymentMethodType {
	case stripe.PaymentMethodTypeCard:
		lockField = "disableCreditCardPayments"
	case stripe.PaymentMethodTypeUSBankAccount:
		lockField = "disableACHPayments"
	case stripe.PaymentMethodTypeSEPADebit:
		lockField = "disableSEPAPayments"
	default:
		return false, nil
	}

	if v, err := docSnap.DataAt(lockField); err != nil {
		return true, err
	} else if disabled, ok := v.(bool); ok {
		return disabled, nil
	} else {
		return true, nil
	}
}

// GetCustomerInfo returns the customer info for the given customer ID.
func (d *StripeFirestore) GetCustomerInfo(ctx context.Context, EntityID string) (*domain.Customer, error) {
	sciDocSnap, err := d.getStripeIntegrationsRef(ctx).Collection("stripeCustomers").Doc(EntityID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var sci domain.Customer

	if err := sciDocSnap.DataTo(&sci); err != nil {
		return nil, err
	}

	return &sci, nil
}

// TODO(yoni): use this function in all the places where needed
// PersistPaymentIntent saves the stripe payment intent information in firestore
func (d *StripeFirestore) PersistPaymentIntentDetails(
	ctx context.Context,
	pi *stripe.PaymentIntent,
	amount *int64,
	openInvoiceDebit int64,
	customerID, invoiceDocID string,
) error {
	paymentRef := d.GetPaymentRef(ctx, pi.ID)
	invoiceDocRef := d.getInvoiceRef(ctx, invoiceDocID)

	err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		invoiceDocSnap, err := tx.Get(invoiceDocRef)
		if err != nil {
			return err
		}

		var invoice invoices.FullInvoice

		if err := invoiceDocSnap.DataTo(&invoice); err != nil {
			return err
		}

		var invoiceBalance = int64(math.Round(invoice.Debit * 100))
		if !utils.IsValidAmount(&invoice, *amount, invoiceBalance, openInvoiceDebit) {
			err := fmt.Errorf("%s: invalid payment intent amount %d, possible duplicate payment", invoice.ID, amount)
			return err
		}

		if err := tx.Update(invoiceDocRef, []firestore.Update{
			{
				FieldPath: []string{fieldInvoiceStripePaymentIntents},
				Value: firestore.ArrayUnion(invoices.StripePaymentIntent{
					ID:                 pi.ID,
					Ref:                paymentRef,
					Amount:             *amount,
					AmountWithFees:     pi.Amount,
					AmountReceived:     pi.AmountReceived,
					Currency:           pi.Currency,
					Status:             pi.Status,
					PaymentMethodTypes: pi.PaymentMethodTypes,
					Debit:              invoice.Debit,
					Timestamp:          time.Now().UTC(),
				}),
			},
		}); err != nil {
			return err
		}

		if err := tx.Create(paymentRef, domain.PaymentIntent{
			Refs: domain.PaymentRefs{
				Customer: invoice.Customer,
				Entity:   invoice.Entity,
				Invoice:  invoiceDocRef,
			},
			Customer:                  customerID,
			ID:                        pi.ID,
			Amount:                    pi.Amount,
			AmountCapturable:          pi.AmountCapturable,
			AmountReceived:            pi.AmountReceived,
			Created:                   pi.Created,
			Currency:                  pi.Currency,
			Description:               pi.Description,
			Livemode:                  pi.Livemode,
			Metadata:                  pi.Metadata,
			Status:                    pi.Status,
			StatementDescriptorSuffix: pi.StatementDescriptorSuffix,
			PaymentMethodTypes:        pi.PaymentMethodTypes,
		}); err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(10))
	if err != nil {
		return err
	}

	return nil
}

// UpdatePaymentIntentDetails updates the payment intent details of invoice in firestore using
// the given update function.
func (d *StripeFirestore) UpdatePaymentIntentDetails(
	ctx context.Context,
	paymentIntent *stripe.PaymentIntent,
	invoiceDocID string,
	updateFn updatePaymentIntentDetailsFn,
) error {
	invoiceDocRef := d.getInvoiceRef(ctx, invoiceDocID)

	err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		invoiceDocSnap, err := tx.Get(invoiceDocRef)
		if err != nil {
			return err
		}

		var invoice invoices.FullInvoice

		if err := invoiceDocSnap.DataTo(&invoice); err != nil {
			return err
		}

		for _, pi := range invoice.StripePaymentIntents {
			if pi.ID != paymentIntent.ID {
				continue
			}

			if err := updateFn(pi); err != nil {
				return err
			}

			return tx.Update(invoiceDocSnap.Ref, []firestore.Update{
				{
					FieldPath: []string{fieldInvoiceStripePaymentIntents},
					Value:     invoice.StripePaymentIntents,
				},
			})
		}

		return ErrInvoicePaymentIntentNotFound
	}, firestore.MaxAttempts(10))
	if err != nil {
		return err
	}

	return nil
}

// TODO(yoni): use this function in all the places where needed
// LockInvoice locks the invoice for payments.
func (d *StripeFirestore) LockInvoice(ctx context.Context, invoiceDocID string) error {
	invoiceDocRef := d.getInvoiceRef(ctx, invoiceDocID)

	err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceDocRef)
		if err != nil {
			return err
		}

		if !docSnap.Exists() {
			return ErrPaymentInvoiceNotFound
		}

		var invoice invoices.FullInvoice
		if err := docSnap.DataTo(&invoice); err != nil {
			return err
		}

		// If invoice is already locked, return error to stop the payment
		if invoice.StripeLocked {
			return ErrPaymentInvoiceLocked
		}

		// Lock the invoice for payments
		return tx.Update(invoiceDocRef, []firestore.Update{
			{
				FieldPath: []string{"stripeLocked"},
				Value:     true,
			},
		})
	})

	return err
}

// TODO(yoni): use this function in all the places where needed
// UnlockInvoice unlocks the invoice for payments.
func (d *StripeFirestore) UnlockInvoice(ctx context.Context, invoiceDocID string) error {
	invoiceDocRef := d.getInvoiceRef(ctx, invoiceDocID)

	err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceDocRef)
		if err != nil {
			return err
		}

		if !docSnap.Exists() {
			// Invoice doc does not exist
			return nil
		}

		return tx.Update(invoiceDocRef, []firestore.Update{
			{
				FieldPath: []string{"stripeLocked"},
				Value:     false,
			},
		})
	})
	if err != nil {
		return ErrFailedToUnlockInvoice
	}

	return err
}

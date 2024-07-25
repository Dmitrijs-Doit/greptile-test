package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/utils"
)

func (s *StripeService) makeACSSDebitPayment(ctx context.Context, email string, entity *common.Entity, invoiceDocSnap *firestore.DocumentSnapshot, today time.Time, pmID string, amount *int64) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	if invoiceDocSnap == nil {
		return ErrInvalidInvoice
	}

	invoiceDocID := invoiceDocSnap.Ref.ID

	if paymentsDisabled, err := s.stripeDAL.IsPaymentTypeDisabled(ctx, stripe.PaymentMethodTypeBACSDebit); err != nil {
		return err
	} else if paymentsDisabled {
		return ErrOnlinePaymentsUnavailable
	}

	// Lock invoice for payments
	if err := lock(ctx, fs, invoiceDocSnap); err != nil {
		return fmt.Errorf("%s : %s", invoiceDocSnap.Ref.ID, err)
	}

	var shouldUnlock = true
	defer func(shouldUnlock *bool) {
		if *shouldUnlock {
			unlock(ctx, l, fs, invoiceDocSnap)
		}
	}(&shouldUnlock)

	if email == "" {
		email = paymentsEmail
	}

	var invoice invoices.FullInvoice
	if err := invoiceDocSnap.DataTo(&invoice); err != nil {
		return err
	}

	if invoice.Symbol != string(fixer.USD) && invoice.Symbol != string(fixer.CAD) {
		return ErrInvalidCurrency
	}

	if pmID == "" {
		err := fmt.Errorf("%s : invalid payment method", invoiceDocID)
		return err
	}

	// Check invoice properties
	if invoice.Canceled || invoice.Paid || invoice.Debit <= 0 {
		l.Warning(fmt.Sprintf("%s : invoice is canceled or already paid", invoiceDocID))
		return nil
	}

	// Check invoice properties
	if today.Before(invoice.Date) {
		l.Warning(fmt.Sprintf("%s : invalid invoice date", invoiceDocID))
		return nil
	}

	openInvoice, err := getOpenInvoice(&invoice, entity)
	if err != nil {
		return err
	}

	if openInvoice == nil {
		err := fmt.Errorf("%s : payment failed - already paid", invoiceDocID)
		return err
	}

	// Get payment intent amount
	var openInvoiceDebit = int64(math.Round(openInvoice.Debit * 100))

	var invoiceBalance = int64(math.Round(invoice.Debit * 100))

	if amount == nil {
		amount = stripe.Int64(openInvoiceDebit)
	}

	// Verify min amount
	if *amount < 50 {
		err := fmt.Errorf("%s: invalid payment intent amount (min) %d", invoice.ID, *amount)
		return err
	}

	if !utils.IsValidAmount(&invoice, *amount, invoiceBalance, openInvoiceDebit) {
		err := fmt.Errorf("%s: invalid payment intent amount %d, possible duplicate payment", invoice.ID, *amount)
		return err
	}

	// Get stripe currency from invoice
	currency, err := toStripeCurrency(invoice.Symbol)
	if err != nil {
		err := fmt.Errorf("%s: invalid currency code %s", invoice.ID, invoice.Currency)
		return err
	}

	l.Debug(map[string]interface{}{"id": invoiceDocID, "amount": amount, "currency": currency})

	sciDocSnap, err := fs.Collection("integrations").Doc(s.integrationDocID).Collection("stripeCustomers").Doc(invoice.Entity.ID).Get(ctx)
	if err != nil {
		return err
	}

	var sci domain.Customer
	if err := sciDocSnap.DataTo(&sci); err != nil {
		return err
	}

	if sci.Metadata.PriorityID != entity.PriorityID {
		err := fmt.Errorf("entity %s and stripe customer integration %s mismatch", entity.PriorityID, sci.ID)
		return err
	}

	customer, err := s.stripeClient.Customers.Get(sci.ID, &stripe.CustomerParams{})
	if err != nil {
		return err
	}

	if customer.Metadata["priority_id"] != entity.PriorityID {
		err := fmt.Errorf("entity %s and stripe customer metadata mismatch", entity.PriorityID)
		return err
	}

	pm, err := s.stripeClient.PaymentMethods.Get(pmID, nil)
	if err != nil {
		return err
	}

	if pm.Customer == nil || pm.Customer.ID != customer.ID {
		err := fmt.Errorf("invalid payment method %s for customer %s", pm.ID, customer.ID)
		return err
	}

	mandateID, ok := pm.Metadata["mandate_id"]
	if !ok {
		err := fmt.Errorf("payment method %s does not have mandate_id", pm.ID)
		return err
	}

	params := &stripe.PaymentIntentParams{
		Customer:                  stripe.String(pm.Customer.ID),
		PaymentMethod:             stripe.String(pm.ID),
		PaymentMethodTypes:        []*string{stripe.String(string(stripe.PaymentMethodTypeACSSDebit))},
		Amount:                    amount,
		Currency:                  stripe.String(string(currency)),
		Confirm:                   stripe.Bool(true),
		CaptureMethod:             stripe.String(string(stripe.PaymentIntentCaptureMethodAutomatic)),
		Description:               stripe.String(fmt.Sprintf("%s-%s", invoice.PriorityID, invoice.ID)),
		StatementDescriptorSuffix: stripe.String(invoice.ID),
		OffSession:                stripe.Bool(true),
		Mandate:                   &mandateID,
	}
	params.AddMetadata("email", email)
	params.AddMetadata("customer_id", invoice.Customer.ID)
	params.AddMetadata("entity_id", invoice.Entity.ID)
	params.AddMetadata("priority_id", entity.PriorityID)
	params.AddMetadata("invoice_id", invoice.ID)
	params.AddMetadata("draft_receipt_id", "")
	params.AddMetadata("receipt_id", "")
	params.AddMetadata("invoice_details", invoice.Details)

	now := time.Now().UTC()

	pi, err := s.stripeClient.PaymentIntents.New(params)
	if err != nil {
		return err
	}

	// payment-intent was created at this point, unlock invoice only if function ended successfully
	shouldUnlock = false

	paymentRef := fs.Collection("integrations").Doc(s.integrationDocID).Collection("stripePayments").Doc(pi.ID)

	err = fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceDocSnap.Ref)
		if err != nil {
			return err
		}

		var invoice invoices.FullInvoice
		if err := docSnap.DataTo(&invoice); err != nil {
			return err
		}

		var invoiceBalance = int64(math.Round(invoice.Debit * 100))
		if !utils.IsValidAmount(&invoice, *amount, invoiceBalance, openInvoiceDebit) {
			err := fmt.Errorf("%s: invalid payment intent amount %d, possible duplicate payment", invoice.ID, amount)
			return err
		}

		if err := tx.Update(invoiceDocSnap.Ref, []firestore.Update{
			{
				FieldPath: []string{"stripePaymentIntents"},
				Value: firestore.ArrayUnion(invoices.StripePaymentIntent{
					ID:                 pi.ID,
					Ref:                paymentRef,
					Amount:             pi.Amount,
					AmountReceived:     pi.AmountReceived,
					Currency:           pi.Currency,
					Status:             pi.Status,
					PaymentMethodTypes: pi.PaymentMethodTypes,
					Debit:              invoice.Debit,
					Timestamp:          now,
				}),
			},
		}); err != nil {
			return err
		}

		if err := tx.Create(paymentRef, PaymentIntent{
			Refs: PaymentRefs{
				Customer: invoice.Customer,
				Entity:   invoice.Entity,
				Invoice:  invoiceDocSnap.Ref,
			},
			Customer:                  customer.ID,
			ID:                        pi.ID,
			AccountID:                 s.stripeClient.accountID,
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

	shouldUnlock = true

	return nil
}

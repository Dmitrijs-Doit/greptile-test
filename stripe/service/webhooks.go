package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/webhook"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/utils"
)

var (
	ErrPaymentIntentAlreadyProcessed = errors.New("payment intent event already processed")
)

func (s *StripeWebhookService) constructWebhookEvent(body []byte, signature string, apiVersion string) (*stripe.Event, error) {
	if apiVersion == stripe.APIVersion {
		// if apiVersion is provided and matches the current api version the SDK uses,
		// then we can use the default ConstructEvent method
		event, err := webhook.ConstructEvent(body, signature, s.stripeClient.webhookSignKey)
		if err != nil {
			return nil, err
		}

		return &event, nil
	}

	// If no apiVersion is provided, then set ignore api version mismatch flag
	event, err := webhook.ConstructEventWithOptions(body, signature, s.stripeClient.webhookSignKey, webhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func (s *StripeWebhookService) HandleEvent(ctx context.Context, body []byte, signature string, apiVersion string) error {
	l := s.loggerProvider(ctx)

	l.Println(string(body))

	l.Infof("webhook api version: %s", apiVersion)

	// Pass the request body and Stripe-Signature header to ConstructEvent,
	// along with the webhook signing key.
	event, err := s.constructWebhookEvent(body, signature, apiVersion)
	if err != nil {
		return err
	}

	l.SetLabels(map[string]string{
		"eventType":       event.Type,
		"eventApiVersion": event.APIVersion,
	})

	l.Infof("event type: %s", event.Type)
	l.Infof("event api version: %s", event.APIVersion)

	// Unmarshal the event data into an appropriate struct depending on its Type
	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return err
		}

		return s.handlePaymentIntentSucceededEvent(ctx, &paymentIntent)
	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
			return err
		}

		return s.handlePaymentIntentFailedEvent(ctx, &paymentIntent)
	case "charge.dispute.created":
		var dispute stripe.Dispute
		if err := json.Unmarshal(event.Data.Raw, &dispute); err != nil {
			return err
		}

		return s.handleChargeDisputeCreated(ctx, &dispute)
	case "setup_intent.succeeded":
		var setupIntent stripe.SetupIntent
		if err := json.Unmarshal(event.Data.Raw, &setupIntent); err != nil {
			return err
		}

		return s.handleSetupIntentSucceededEvent(ctx, &setupIntent)
	case "mandate.updated":
		var mandate stripe.Mandate
		if err := json.Unmarshal(event.Data.Raw, &mandate); err != nil {
			return err
		}

		return s.handleMandateUpdatedEvent(ctx, &mandate)
	default:
		l.Warningf("Unhandled Stripe webhook event type: %s", event.Type)
		return nil
	}
}

func newCustomerField(ctx context.Context, fs *firestore.Client, pi *stripe.PaymentIntent) (map[string]interface{}, error) {
	if customerID, prs := pi.Metadata["customer_id"]; prs && customerID != "" {
		customerRef := fs.Collection("customers").Doc(customerID)

		customer, err := common.GetCustomer(ctx, customerRef)
		if err != nil {
			return nil, err
		}

		return map[string]interface{}{
			"title": "Customer",
			"value": fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", customerID, customer.Name),
			"short": true,
		}, nil
	}

	return map[string]interface{}{
		"title": "Customer",
		"value": "N/A",
		"short": true,
	}, nil
}

func newInvoiceField(pi *stripe.PaymentIntent) map[string]interface{} {
	if pi.Metadata["customer_id"] != "" && pi.Metadata["entity_id"] != "" && pi.Metadata["invoice_id"] != "" {
		return map[string]interface{}{
			"title": "Invoice",
			"value": fmt.Sprintf("<https://console.doit.com/customers/%s/invoices/%s/%s|%s>",
				pi.Metadata["customer_id"],
				pi.Metadata["entity_id"],
				pi.Metadata["invoice_id"],
				pi.Metadata["invoice_id"],
			),
			"short": true,
		}
	}

	return map[string]interface{}{
		"title": "Invoice",
		"value": pi.Metadata["invoice_id"],
		"short": true,
	}
}

func newReceiptField(pi *stripe.PaymentIntent) map[string]interface{} {
	if receiptID, prs := pi.Metadata["receipt_id"]; prs && receiptID != "" {
		return map[string]interface{}{
			"title": "Receipt",
			"value": receiptID,
			"short": true,
		}
	}

	return map[string]interface{}{
		"title": "Draft Receipt",
		"value": pi.Metadata["draft_receipt_id"],
		"short": true,
	}
}

func paymentMethodType(pi *stripe.PaymentIntent) (stripe.PaymentMethodType, error) {
	if len(pi.PaymentMethodTypes) != 1 {
		return "", fmt.Errorf("expected only one payment method type, got %d", len(pi.PaymentMethodTypes))
	}

	return stripe.PaymentMethodType(pi.PaymentMethodTypes[0]), nil
}

func (s *StripeWebhookService) handlePaymentIntentSucceededEvent(ctx context.Context, pi *stripe.PaymentIntent) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	time.Sleep(15 * time.Second) // TODO: create cloud task instead of sleep

	pmType, err := paymentMethodType(pi)
	if err != nil {
		return err
	}

	if pmType == stripe.PaymentMethodTypeUSBankAccount || pmType == stripe.PaymentMethodTypeSEPADebit || pmType == stripe.PaymentMethodTypeBACSDebit || pmType == stripe.PaymentMethodTypeACSSDebit {
		if err := s.handleBankDebitPaymentSucceeded(ctx, pi); err != nil {
			l.Error(err)

			if err == ErrPaymentIntentAlreadyProcessed {
				return nil
			}
		}
	}

	pi, err = s.stripeClient.PaymentIntents.Get(pi.ID, &stripe.PaymentIntentParams{})
	if err != nil {
		return err
	}

	customerField, err := newCustomerField(ctx, fs, pi)
	if err != nil {
		return err
	}

	invoiceField := newInvoiceField(pi)
	receiptField := newReceiptField(pi)

	var email = pi.Metadata["email"]

	var message = map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#4CAF50",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", email, email),
				"title":       "Payment Intent Succeeded",
				"title_link":  fmt.Sprintf("https://dashboard.stripe.com/payments/%s", pi.ID),
				"fields": []map[string]interface{}{
					{
						"title": "Stripe Account",
						"value": domain.StripeAccountNames[s.stripeClient.accountID],
						"short": true,
					},
					{
						"title": "Payment Method",
						"value": pmType,
						"short": true,
					},
					{
						"title": "Status",
						"value": pi.Status,
						"short": false,
					},
					customerField,
					{
						"title": "Priority ID",
						"value": pi.Metadata["priority_id"],
						"short": true,
					},
					invoiceField,
					receiptField,
					{
						"title": "Currency",
						"value": CurrencyToUpperString(pi.Currency),
						"short": true,
					},
					{
						"title": "Amount",
						"value": float64(pi.AmountReceived) / 100,
						"short": true,
					},
				},
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, collectionOpsSlackChannel); err != nil {
		return err
	}

	return nil
}

func (s *StripeWebhookService) handlePaymentIntentFailedEvent(ctx context.Context, pi *stripe.PaymentIntent) error {
	fs := s.Firestore(ctx)

	pmType, err := paymentMethodType(pi)
	if err != nil {
		return err
	}

	// failed credit card payments are not saved in the DB
	if pmType != stripe.PaymentMethodTypeCard {
		s.updateDBPaymentStatus(ctx, pi)
	}

	customerField, err := newCustomerField(ctx, fs, pi)
	if err != nil {
		return err
	}

	invoiceField := newInvoiceField(pi)

	var email = pi.Metadata["email"]

	var message = map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#F44336",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", email, email),
				"title":       "Payment Intent Failed",
				"title_link":  fmt.Sprintf("https://dashboard.stripe.com/payments/%s", pi.ID),
				"fields": []map[string]interface{}{
					{
						"title": "Stripe Account",
						"value": domain.StripeAccountNames[s.stripeClient.accountID],
						"short": true,
					},
					{
						"title": "Payment Method",
						"value": pmType,
						"short": true,
					},
					{
						"title": "Error Code",
						"value": pi.LastPaymentError.Code,
						"short": true,
					},
					{
						"title": "Decline Code",
						"value": pi.LastPaymentError.DeclineCode,
						"short": true,
					},
					{
						"title": "Error Message",
						"value": pi.LastPaymentError.Msg,
						"short": true,
					},
					{
						"title": "Status",
						"value": pi.Status,
						"short": true,
					},
					customerField,
					{
						"title": "Priority ID",
						"value": pi.Metadata["priority_id"],
						"short": true,
					},
					invoiceField,
					{
						"title": "Draft Receipt",
						"value": pi.Metadata["draft_receipt_id"],
						"short": true,
					},
					{
						"title": "Currency",
						"value": CurrencyToUpperString(pi.Currency),
						"short": true,
					},
					{
						"title": "Amount",
						"value": float64(pi.Amount) / 100,
						"short": true,
					},
				},
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, collectionOpsSlackChannel); err != nil {
		return err
	}

	return nil
}

func (s *StripeWebhookService) handleBankDebitPaymentSucceeded(ctx context.Context, evtPaymentIntent *stripe.PaymentIntent) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	paymentRef := fs.Collection("integrations").
		Doc(s.integrationDocID).
		Collection("stripePayments").
		Doc(evtPaymentIntent.ID)

	paymentDocSnap, err := paymentRef.Get(ctx)
	if err != nil {
		return err
	}

	var payment PaymentIntent

	if err := paymentDocSnap.DataTo(&payment); err != nil {
		return err
	}

	invoiceDocSnap, err := payment.Refs.Invoice.Get(ctx)
	if err != nil {
		return err
	}

	var invoice invoices.FullInvoice

	if err := invoiceDocSnap.DataTo(&invoice); err != nil {
		return err
	}

	// Lock invoice for payments
	if err := lock(ctx, fs, invoiceDocSnap); err != nil {
		return err
	}

	var shouldUnlock = true
	defer func(shouldUnlock *bool) {
		if *shouldUnlock {
			unlock(ctx, l, fs, invoiceDocSnap)
		}
	}(&shouldUnlock)

	pi, err := s.stripeClient.PaymentIntents.Get(evtPaymentIntent.ID, &stripe.PaymentIntentParams{})
	if err != nil {
		return err
	}

	if receiptID, ok := pi.Metadata["receipt_id"]; ok && receiptID != "" {
		// Payment intent already processed and a receipt was created
		return ErrPaymentIntentAlreadyProcessed
	}

	invoiceBalance := utils.ToCents(invoice.Debit)
	isPartialPayment := pi.AmountReceived < invoiceBalance

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)

	// Create temporary receipt
	receipt, err := createReceipt(&invoice, pi.AmountReceived, today)
	if err != nil {
		return err
	}

	// patch receipt fields
	if err := patchReceipt(invoice.Company, receipt, invoice.ID, "", isPartialPayment); err != nil {
		return err
	}

	// payment was successful at this point, unlock invoice only if function ended successfully
	shouldUnlock = false

	// approve receipt
	approvedReceipt, err := approveReceipt(invoice.Company, receipt)
	if err != nil {
		return nil
	}

	// update payment intent metadata with draft receipt ID
	params := &stripe.PaymentIntentParams{}
	params.AddMetadata("draft_receipt_id", receipt.ID)
	params.AddMetadata("receipt_id", approvedReceipt.ID)

	pi, err = s.stripeClient.PaymentIntents.Update(pi.ID, params)
	if err != nil {
		return err
	}

	// Update payment in DB
	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceDocSnap.Ref)
		if err != nil {
			return err
		}

		var invoice invoices.FullInvoice
		if err := docSnap.DataTo(&invoice); err != nil {
			return err
		}

		for _, _pi := range invoice.StripePaymentIntents {
			if _pi.ID == pi.ID {
				_pi.Status = pi.Status
				_pi.AmountReceived = pi.AmountReceived
				_pi.Debit = invoice.Debit
			}
		}

		if err := tx.Update(invoiceDocSnap.Ref, []firestore.Update{
			{
				FieldPath: []string{"stripePaymentIntents"},
				Value:     invoice.StripePaymentIntents,
			},
		}); err != nil {
			return err
		}

		if err := tx.Update(paymentRef, []firestore.Update{
			{FieldPath: []string{"status"}, Value: pi.Status},
			{FieldPath: []string{"amount_received"}, Value: pi.AmountReceived},
			{FieldPath: []string{"metadata", "draft_receipt_id"}, Value: receipt.ID},
			{FieldPath: []string{"metadata", "receipt_id"}, Value: approvedReceipt.ID},
		}); err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(10)); err != nil {
		l.Warningf("failed to update invoice %s after captured pi %s: %s", invoiceDocSnap.Ref.ID, pi.ID, err)
		return nil
	}

	shouldUnlock = true

	return nil
}

// updateDBPaymentStatus updates the payment status in the DB. Logs error if it fails.
func (s *StripeWebhookService) updateDBPaymentStatus(ctx context.Context, pi *stripe.PaymentIntent) {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		paymentRef := fs.Collection("integrations").Doc(s.integrationDocID).Collection("stripePayments").Doc(pi.ID)
		paymentDocSnap, err := tx.Get(paymentRef)
		if err != nil {
			return err
		}
		var payment PaymentIntent
		if err := paymentDocSnap.DataTo(&payment); err != nil {
			return err
		}

		invoiceRef := payment.Refs.Invoice
		invoiceDocSnap, err := tx.Get(invoiceRef)
		if err != nil {
			return err
		}
		var invoice invoices.FullInvoice
		if err := invoiceDocSnap.DataTo(&invoice); err != nil {
			return err
		}

		for _, _pi := range invoice.StripePaymentIntents {
			if _pi.ID == pi.ID {
				_pi.Status = pi.Status
			}
		}

		if err := tx.Update(invoiceRef, []firestore.Update{
			{
				FieldPath: []string{"stripePaymentIntents"},
				Value:     invoice.StripePaymentIntents,
			},
		}); err != nil {
			return err
		}

		if err := tx.Update(paymentRef, []firestore.Update{
			{FieldPath: []string{"status"}, Value: pi.Status},
		}); err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(10)); err != nil {
		l.Warningf("failed to update payment status on pi %s: %s", pi.ID, err)
	}
}

func (s *StripeWebhookService) handleChargeDisputeCreated(ctx context.Context, dispute *stripe.Dispute) error {
	fs := s.Firestore(ctx)

	pi, err := s.stripeClient.PaymentIntents.Get(dispute.PaymentIntent.ID, nil)
	if err != nil {
		return err
	}

	pmType, err := paymentMethodType(pi)
	if err != nil {
		return err
	}

	customerField, err := newCustomerField(ctx, fs, pi)
	if err != nil {
		return err
	}

	invoiceField := newInvoiceField(pi)
	receiptField := newReceiptField(pi)

	var email = pi.Metadata["email"]

	var message = map[string]interface{}{
		"text": "<!here>",
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#F44336",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", email, email),
				"title":       "Dispute Created",
				"title_link":  fmt.Sprintf("https://dashboard.stripe.com/payments/%s", pi.ID),
				"fields": []map[string]interface{}{
					{
						"title": "Stripe Account",
						"value": domain.StripeAccountNames[s.stripeClient.accountID],
						"short": true,
					},
					{
						"title": "Payment Method",
						"value": pmType,
						"short": true,
					},
					{
						"title": "Status",
						"value": dispute.Status,
						"short": false,
					},
					customerField,
					{
						"title": "Priority ID",
						"value": pi.Metadata["priority_id"],
						"short": true,
					},
					invoiceField,
					receiptField,
					{
						"title": "Currency",
						"value": CurrencyToUpperString(pi.Currency),
						"short": true,
					},
					{
						"title": "Amount",
						"value": float64(pi.AmountReceived) / 100,
						"short": true,
					},
				},
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, collectionOpsDisputesSlackChannel); err != nil {
		return err
	}

	return nil
}

func (s *StripeWebhookService) handleSetupIntentSucceededEvent(ctx context.Context, setupIntent *stripe.SetupIntent) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	if setupIntent.PaymentMethod == nil {
		l.Error("no payment method on setup intent")
		return nil
	}

	entityID, ok := setupIntent.Metadata["entity_id"]
	if !ok {
		customer, err := s.stripeClient.Customers.Get(setupIntent.Customer.ID, nil)
		if err != nil {
			return err
		}

		entityID = customer.Metadata["entity_id"]
	}

	pm, err := s.stripeClient.PaymentMethods.Get(setupIntent.PaymentMethod.ID, nil)
	if err != nil {
		return err
	}

	var entityPayment common.EntityPayment

	switch pm.Type {
	case stripe.PaymentMethodTypeCard:
		entityPayment = common.EntityPayment{
			Type:      common.EntityPaymentTypeCard,
			AccountID: string(s.stripeClient.accountID),
			Card: &common.PaymentMethodCard{
				ID:       pm.ID,
				Last4:    pm.Card.Last4,
				ExpYear:  pm.Card.ExpYear,
				ExpMonth: pm.Card.ExpMonth,
			},
		}
	case stripe.PaymentMethodTypeUSBankAccount:
		entityPayment = common.EntityPayment{
			Type:      common.EntityPaymentTypeUSBankAccount,
			AccountID: string(s.stripeClient.accountID),
			Card:      nil,
			BankAccount: &common.PaymentMethodUSBankAccount{
				ID:       pm.ID,
				Last4:    pm.USBankAccount.Last4,
				BankName: pm.USBankAccount.BankName,
			},
		}
	case stripe.PaymentMethodTypeSEPADebit:
		entityPayment = common.EntityPayment{
			Type:      common.EntityPaymentTypeSEPADebit,
			AccountID: string(s.stripeClient.accountID),
			SEPADebit: &common.PaymentMethodSEPADebit{
				ID:       pm.ID,
				Last4:    pm.SEPADebit.Last4,
				Name:     pm.BillingDetails.Name,
				Email:    pm.BillingDetails.Email,
				BankCode: pm.SEPADebit.BankCode,
			},
		}
	case stripe.PaymentMethodTypeBACSDebit:
		entityPayment = common.EntityPayment{
			Type:      common.EntityPaymentTypeBACSDebit,
			AccountID: string(s.stripeClient.accountID),
			BACSDebit: &common.PaymentMethodBACSDebit{
				ID:    pm.ID,
				Last4: pm.BACSDebit.Last4,
				Name:  pm.BillingDetails.Name,
				Email: pm.BillingDetails.Email,
			},
		}
	case stripe.PaymentMethodTypeACSSDebit:
		// update the payment method with the mandate ID, this is required for future payments
		params := &stripe.PaymentMethodParams{}
		params.AddMetadata("mandate_id", setupIntent.Mandate.ID)

		pm, err = s.stripeClient.PaymentMethods.Update(pm.ID, params)
		if err != nil {
			return err
		}

		entityPayment = common.EntityPayment{
			Type:      common.EntityPaymentTypeACSSDebit,
			AccountID: string(s.stripeClient.accountID),
			ACSSDebit: &common.PaymentMethodACSSDebit{
				ID:    pm.ID,
				Last4: pm.ACSSDebit.Last4,
				Name:  pm.BillingDetails.Name,
				Email: pm.BillingDetails.Email,
			},
		}
	default:
		l.Error("unknown payment method type")
		return nil
	}

	newEntityStr, ok := setupIntent.Metadata["new_entity"]
	if !ok {
		newEntityStr = "false"
	}

	newEntity, err := strconv.ParseBool(newEntityStr)
	if err != nil || !newEntity {
		// if the entity is an old one, then we don't want to assign a default payment method.
		return nil
	}

	entityRef := fs.Collection("entities").Doc(entityID)
	if _, err := entityRef.Update(ctx, []firestore.Update{
		{Path: "payment", Value: entityPayment},
	}); err != nil {
		if status.Code(err) == codes.Unavailable {
			return err
		}

		l.Errorf("failed to update entity payment: %s", err)

		return nil
	}

	return nil
}

// handleMandateUpdatedEvent sets the mandate status on the payment method metadata
func (s *StripeWebhookService) handleMandateUpdatedEvent(ctx context.Context, mandate *stripe.Mandate) error {
	l := s.loggerProvider(ctx)

	pm, err := s.stripeClient.PaymentMethods.Get(mandate.PaymentMethod.ID, nil)
	if err != nil {
		l.Errorf("failed to get payment method %s: %s", mandate.PaymentMethod.ID, err)
		return err
	}

	params := &stripe.PaymentMethodParams{}
	params.AddMetadata("mandate_status", string(mandate.Status))

	_, err = s.stripeClient.PaymentMethods.Update(pm.ID, params)
	if err != nil {
		l.Errorf("failed to update payment method metadata%s: %s", pm.ID, err)
		return err
	}

	return nil
}

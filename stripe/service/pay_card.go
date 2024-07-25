package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/utils"
)

// TODO(yoni): instead of passing the invoiceDocSnap, pass the invoiceDocID and get the invoiceDocSnap inside this function
func (s *StripeService) makeCreditCardPayment(ctx context.Context, email string, entity *common.Entity, invoiceDocSnap *firestore.DocumentSnapshot, today time.Time, pmID string, amount *int64) error {
	l := s.loggerProvider(ctx)

	if invoiceDocSnap == nil {
		return ErrInvalidInvoice
	}

	invoiceDocID := invoiceDocSnap.Ref.ID

	if paymentsDisabled, err := s.stripeDAL.IsPaymentTypeDisabled(ctx, stripe.PaymentMethodTypeCard); err != nil {
		return err
	} else if paymentsDisabled {
		return ErrOnlinePaymentsUnavailable
	}

	// Lock invoice for payments
	if err := s.stripeDAL.LockInvoice(ctx, invoiceDocID); err != nil {
		return fmt.Errorf("%s : %s", invoiceDocSnap.Ref.ID, err)
	}

	var shouldUnlock = true
	defer func(shouldUnlock *bool) {
		if *shouldUnlock {
			err := s.stripeDAL.UnlockInvoice(ctx, invoiceDocID)
			if err != nil {
				l.Errorf("error unlocking invoice %s: %v", invoiceDocID, err)
			}
		}
	}(&shouldUnlock)

	if email == "" {
		email = paymentsEmail
	}

	var invoice invoices.FullInvoice
	if err := invoiceDocSnap.DataTo(&invoice); err != nil {
		return err
	}

	priorityCompany := invoice.Company
	priorityCustomerID := invoice.PriorityID

	if pmID == "" {
		err := fmt.Errorf("%s : invalid payment method", invoiceDocID)
		return err
	}

	// Check invoice properties
	if invoice.Canceled || invoice.Paid || invoice.Debit <= 0 {
		l.Warningf("%s : invoice is canceled or already paid", invoiceDocID)
		return nil
	}

	// Check invoice properties
	if today.Before(invoice.Date) {
		l.Warning(fmt.Sprintf("%s : invalid invoice date", invoiceDocID))
		return nil
	}

	customer, err := s.customersDAL.GetCustomer(ctx, entity.Customer.ID)
	if err != nil {
		l.Errorf("error getting customer %s. error [%s]", entity.Customer.ID, err)
		return err
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
	openInvoiceDebit := utils.ToCents(openInvoice.Debit)
	invoiceBalance := utils.ToCents(invoice.Debit)
	invoiceTaxAmount := utils.ToCents(invoice.Vat)
	invoiceTotalAmount := utils.ToCents(invoice.TotalTax)

	if amount == nil {
		amount = stripe.Int64(openInvoiceDebit)
	}

	// Verify min amount
	if *amount < 50 {
		err := fmt.Errorf("%s : invalid payment intent amount (min) %d", invoiceDocID, *amount)
		return err
	}

	if !utils.IsValidAmount(&invoice, *amount, invoiceBalance, openInvoiceDebit) {
		err := fmt.Errorf("%s : invalid payment intent amount %d, possible duplicate payment", invoiceDocID, *amount)
		return err
	}

	isPartialPayment := *amount < invoiceBalance
	isSinglePayment := *amount == invoiceTotalAmount

	l.Infof("single payment [%t]", isSinglePayment)
	l.Infof("partial payment [%t]", isPartialPayment)

	// Get stripe currency from invoice
	currency, err := toStripeCurrency(invoice.Symbol)
	if err != nil {
		err := fmt.Errorf("%s : invalid currency code %s", invoiceDocID, invoice.Currency)
		return err
	}

	l.Infof("invoice document id [%s]", invoiceDocID)
	l.Infof("invoice balance [%d]", invoiceBalance)
	l.Infof("payment amount [%d]", *amount)
	l.Infof("payment currency [%s]", currency)

	sci, err := s.stripeDAL.GetCustomerInfo(ctx, invoice.Entity.ID)
	if err != nil {
		return err
	}

	if sci.Metadata.PriorityID != entity.PriorityID {
		err := fmt.Errorf("entity %s and stripe customer integration %s mismatch", entity.PriorityID, sci.ID)
		return err
	}

	stripeCustomer, err := s.stripeClient.Customers.Get(sci.ID, &stripe.CustomerParams{})
	if err != nil {
		return err
	}

	if stripeCustomer.Metadata["priority_id"] != entity.PriorityID {
		err := fmt.Errorf("entity %s and stripe customer metadata mismatch", entity.PriorityID)
		return err
	}

	pm, err := s.stripeClient.PaymentMethods.Get(pmID, nil)
	if err != nil {
		return err
	}

	if pm.Customer != nil && pm.Customer.ID != stripeCustomer.ID {
		err := fmt.Errorf("invalid payment method %s for customer %s", pm.ID, stripeCustomer.ID)
		return err
	}

	if disabled, reason := PMDisabled(pm); disabled {
		return fmt.Errorf("%w: %s", ErrPaymentMethodDisabled, reason)
	}

	var (
		shouldDeleteFeesDraftInvoice = false
		shouldDeleteDraftReceipt     = false
		feesInvoiceIdentifier        *priorityDomain.PriorityInvoiceIdentifier
		feesAmount                   int64
		feesInvoiceTaxAmount         int64
		feesInvoiceTotalAfterTax     int64
	)

	isExemptFromFees, feePct := isExemptFromCreditCardFees(customer, entity, *amount)
	shouldChargeFees := !isExemptFromFees

	l.Infof("should charge fees [%t]", shouldChargeFees)
	l.Infof("fee percentage [%f]", feePct)

	if shouldChargeFees {
		feesAmount = s.CalculateTotalFees(*amount, feePct)

		feesInvoice, err := s.createDraftFeesInvoice(ctx, &invoice, feesAmount, today)
		if err != nil {
			l.Errorf("error creating draft fees invoice. error [%s]", err)
			return err
		}

		l.Infof("created draft fees invoice [%s]", feesInvoice.InvoiceNumber)

		shouldDeleteFeesDraftInvoice = true
		feesInvoiceTotalAfterTax = utils.ToCents(feesInvoice.TotalAfterTax)
		feesInvoiceTaxAmount = utils.ToCents(feesInvoice.Vat)

		feesInvoiceIdentifier = &priorityDomain.PriorityInvoiceIdentifier{
			PriorityCompany:    priorityCompany,
			PriorityCustomerID: priorityCustomerID,
			InvoiceNumber:      feesInvoice.InvoiceNumber,
		}

		defer func(shouldDelete *bool, pid *priorityDomain.PriorityInvoiceIdentifier) {
			if *shouldDelete && pid != nil {
				if err := s.priorityService.DeleteInvoice(ctx, *pid); err != nil {
					l.Errorf("failed to delete draft fees invoice: %s", err)
				}
			}
		}(&shouldDeleteFeesDraftInvoice, feesInvoiceIdentifier)

		if feesInvoiceTotalAfterTax < 0 || feesInvoiceTotalAfterTax > *amount {
			err := fmt.Errorf("%s : invalid fees amount %d", invoiceDocID, feesInvoiceTotalAfterTax)
			return err
		}
	}

	paymentIntentAmount := *amount + feesInvoiceTotalAfterTax
	totalTaxAmount := invoiceTaxAmount + feesInvoiceTaxAmount

	l.Infof("fees invoice amount [%d]", feesAmount)
	l.Infof("fees invoice tax amount [%d]", feesInvoiceTaxAmount)
	l.Infof("fees invoice total after tax [%d]", feesInvoiceTotalAfterTax)

	l.Infof("total tax amount [%d]", totalTaxAmount)
	l.Infof("payment intent amount [%d]", paymentIntentAmount)

	// Create temporary receipt
	receipt, err := createReceipt(&invoice, paymentIntentAmount, today)
	if err != nil {
		return err
	}

	shouldDeleteDraftReceipt = true

	defer func(shouldDelete *bool) {
		if *shouldDelete && strings.HasPrefix(receipt.ID, "T") {
			if err := s.priorityService.DeleteReceipt(ctx, priorityCompany, receipt.ID); err != nil {
				l.Errorf("failed to delete draft receipt: %s", err)
			}
		}
	}(&shouldDeleteDraftReceipt)

	params := &stripe.PaymentIntentParams{
		PaymentMethod:             stripe.String(pm.ID),
		PaymentMethodTypes:        stripe.StringSlice([]string{string(stripe.PaymentMethodTypeCard)}),
		Amount:                    stripe.Int64(paymentIntentAmount),
		Currency:                  stripe.String(string(currency)),
		Confirm:                   stripe.Bool(true),
		CaptureMethod:             stripe.String(string(stripe.PaymentIntentCaptureMethodManual)),
		Description:               stripe.String(fmt.Sprintf("%s-%s", priorityCustomerID, invoice.ID)),
		StatementDescriptorSuffix: stripe.String(invoice.ID),
	}
	params.AddMetadata("email", email)
	params.AddMetadata("customer_id", invoice.Customer.ID)
	params.AddMetadata("entity_id", invoice.Entity.ID)
	params.AddMetadata("priority_id", entity.PriorityID)
	params.AddMetadata("invoice_id", invoice.ID)
	params.AddMetadata("draft_receipt_id", receipt.ID)
	params.AddMetadata("receipt_id", "")
	params.AddMetadata("invoice_details", invoice.Details)

	if feesInvoiceIdentifier != nil {
		params.AddMetadata("draft_fees_invoice_id", feesInvoiceIdentifier.InvoiceNumber)
		params.AddMetadata("fees_invoice_id", "")
	} else {
		params.AddMetadata("draft_fees_invoice_id", "N/A")
		params.AddMetadata("fees_invoice_id", "N/A")
	}

	if pm.Customer == nil {
		// New credit card added in the Pay dialog, setup for future use
		params.Customer = stripe.String(sci.ID)
		params.SetupFutureUsage = stripe.String(string(stripe.PaymentIntentSetupFutureUsageOffSession))
	} else {
		params.Customer = stripe.String(pm.Customer.ID)
		params.OffSession = stripe.Bool(true)
	}

	// Add L3 charge data if invoice is not partial payment or invoice has no tax.
	// L3 charge data is only relevant for US: https://stripe.com/docs/level3#sending-level-iii-data
	if s.stripeClient.accountID == domain.StripeAccountUS && (isSinglePayment || totalTaxAmount == 0) {
		if err := s.AddLevel3ChargeData(params, &invoice, sci.ID, paymentIntentAmount, totalTaxAmount); err != nil {
			l.Errorf("failed to add level3 charge data error: %s", err)
		}
	}

	pi, err := s.stripeClient.PaymentIntents.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			if err := s.sendPaymentFailedNotification(ctx, entity, &invoice, *amount, stripeErr); err != nil {
				l.Errorf("failed to send payment failed notification: %s", err)
			}
		}

		return err
	}

	if pi.Status != stripe.PaymentIntentStatusRequiresCapture {
		return fmt.Errorf("status %s for payment intent %s", pi.Status, pi.ID)
	}

	if err := s.stripeDAL.PersistPaymentIntentDetails(ctx,
		pi,
		amount,
		openInvoiceDebit,
		stripeCustomer.ID,
		invoiceDocID,
	); err != nil {
		return err
	}

	paymentRef := s.stripeDAL.GetPaymentRef(ctx, pi.ID)

	// Capture payment
	pi, err = s.stripeClient.PaymentIntents.Capture(pi.ID, &stripe.PaymentIntentCaptureParams{})
	if err != nil {
		if err := s.cancelPaymentIntent(ctx, pi.ID, invoiceDocSnap.Ref, paymentRef); err != nil {
			l.Error(err)
		}

		return err
	}

	// Payment was successful at this point, unlock invoice only if function ended successfully
	// and do not cancel the draft fees invoice and receipt
	shouldUnlock = false
	shouldDeleteFeesDraftInvoice = false
	shouldDeleteDraftReceipt = false

	// Close and print fees invoice
	if feesInvoiceIdentifier != nil {
		ivNum, err := s.priorityService.ApproveInvoice(ctx, *feesInvoiceIdentifier)
		if err != nil {
			return err
		}

		feesInvoiceIdentifier.InvoiceNumber = ivNum

		l.Infof("fees invoice approved: %s", ivNum)
	}

	// Update invoice payment intent with the linked CC fees invoice details
	updateFn := func(v *invoices.StripePaymentIntent) error {
		v.Status = pi.Status
		v.AmountReceived = pi.AmountReceived

		if feesInvoiceIdentifier != nil {
			feesInvoiceDocID := fmt.Sprintf("%s-%s-%s",
				feesInvoiceIdentifier.PriorityCompany,
				feesInvoiceIdentifier.PriorityCustomerID,
				feesInvoiceIdentifier.InvoiceNumber,
			)
			feesInvoiceDocRef := invoiceDocSnap.Ref.Parent.Doc(feesInvoiceDocID)

			v.LinkedInvoice = &invoices.StripePaymentIntentLinkedInvoice{
				AmountFees: feesInvoiceTotalAfterTax,
				ID:         feesInvoiceIdentifier.InvoiceNumber,
				Ref:        feesInvoiceDocRef,
			}
		}

		return nil
	}

	if err := s.stripeDAL.UpdatePaymentIntentDetails(ctx, pi, invoiceDocID, updateFn); err != nil {
		l.Errorf("failed to update payment intent %s details for invoice %s: %s", pi.ID, invoiceDocSnap.Ref.ID, err)
		return err
	}

	// Set receipt details
	var feesInvoiceNumber string
	if feesInvoiceIdentifier != nil {
		feesInvoiceNumber = feesInvoiceIdentifier.InvoiceNumber
	}

	if err := patchReceipt(priorityCompany, receipt, invoice.ID, feesInvoiceNumber, isPartialPayment); err != nil {
		errMsg := fmt.Sprintf("failed to patch receipt %s for invoice %s (payment intent %s): %s", receipt.ID, invoiceDocSnap.Ref.ID, pi.ID, err)
		l.Error(errMsg)

		if e := s.sendReceiptCreationSlackAlert(ctx, priorityCustomerID, receipt.ID, invoice.ID, pi.ID, errMsg); e != nil {
			l.Errorf("failed to send receipt creation slack alert; %s", e)
		}

		return err
	}

	// Approve receipt
	approvedReceipt, err := approveReceipt(priorityCompany, receipt)
	if err != nil {
		errMsg := fmt.Sprintf("failed to approve receipt %s for invoice %s (payment intent %s): %s", receipt.ID, invoiceDocSnap.Ref.ID, pi.ID, err)
		l.Error(errMsg)

		if e := s.sendReceiptCreationSlackAlert(ctx, priorityCustomerID, receipt.ID, invoice.ID, pi.ID, errMsg); e != nil {
			l.Errorf("failed to send receipt creation slack alert; %s", e)
		}

		return err
	}

	// update payment intent metadata with approved receipt ID
	params = &stripe.PaymentIntentParams{}
	params.AddMetadata("receipt_id", approvedReceipt.ID)

	if feesInvoiceIdentifier != nil {
		params.AddMetadata("fees_invoice_id", feesInvoiceIdentifier.InvoiceNumber)
	}

	if _, err = s.stripeClient.PaymentIntents.Update(pi.ID, params); err != nil {
		l.Errorf("failed to update payment intent %s metadata. error [%s]", pi.ID, err)
	}

	// Update payment in DB
	if _, err := paymentRef.Update(ctx, []firestore.Update{
		{FieldPath: []string{"status"}, Value: pi.Status},
		{FieldPath: []string{"amount_received"}, Value: pi.AmountReceived},
		{FieldPath: []string{"metadata", "draft_receipt_id"}, Value: receipt.ID},
		{FieldPath: []string{"metadata", "receipt_id"}, Value: approvedReceipt.ID},
	}); err != nil {
		l.Errorf("failed to update payment intent %s details. error [%s]", pi.ID, err)
	}

	shouldUnlock = true

	return nil
}

func (s *StripeService) sendReceiptCreationSlackAlert(
	ctx context.Context,
	priorityID string,
	receiptID string,
	invoiceID string,
	paymentIntent string,
	errMessage string,
) error {
	if !common.Production {
		return nil
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":         time.Now().Unix(),
				"color":      "#F44336",
				"title":      "Receipt creation error",
				"title_link": fmt.Sprintf("https://dashboard.stripe.com/payments/%s", paymentIntent),
				"fields": []map[string]interface{}{
					{
						"title": "Priority Company Id",
						"value": priorityID,
						"short": true,
					},
					{
						"title": "Invoice Id",
						"value": invoiceID,
						"short": true,
					},
					{
						"title": "Receipt Id",
						"value": receiptID,
						"short": true,
					},
					{
						"title": "Error Message",
						"value": errMessage,
						"short": true,
					},
				},
			},
		},
	}

	_, err := common.PublishToSlack(ctx, message, payCardAlertsSlackChannel)
	if err != nil {
		return err
	}

	return nil
}

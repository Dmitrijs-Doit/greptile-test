package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

const (
	cashName      = "105"
	paymentCode   = "21"
	paymentsEmail = "payments@doit.com"
)

func (s *StripeService) cancelPaymentIntent(ctx context.Context, piID string, invoiceRef, paymentRef *firestore.DocumentRef) error {
	fs := s.Firestore(ctx)

	pi, err := s.stripeClient.PaymentIntents.Cancel(piID, &stripe.PaymentIntentCancelParams{})
	if err != nil {
		return fmt.Errorf("failed to cancel payment intent %s for invoice %s: %s", pi.ID, invoiceRef.ID, err)
	}

	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceRef)
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
			{FieldPath: []string{"canceled_at"}, Value: stripe.Int64(pi.CanceledAt)},
		}); err != nil {
			return err
		}

		return nil
	}, firestore.MaxAttempts(10)); err != nil {
		return fmt.Errorf("failed to update invoice %s after canceled payment intent %s: %s", invoiceRef.ID, pi.ID, err)
	}

	return nil
}

func approveReceipt(company string, receipt *priorityDomain.TInvoice) (*priorityDomain.TInvoice, error) {
	form := fmt.Sprintf("TINVOICES(IVNUM='%s',DEBIT='D',IVTYPE='T')", receipt.ID)

	data, _ := json.Marshal(priorityDomain.TInvoice{
		Status: invoices.StatusApproved,
	})

	resp, err := priority.Client.Patch(company, form, nil, data)
	if err != nil {
		return nil, fmt.Errorf("approve receipt failed: %s", err)
	}

	var approved priorityDomain.TInvoice

	if err := json.Unmarshal(resp, &approved); err != nil {
		return nil, err
	}

	return &approved, nil
}

func patchReceipt(company string, receipt *priorityDomain.TInvoice, invoiceID, feesInvoiceID string, isPartialPayment bool) error {
	tfncItems2Filter := fmt.Sprintf("ROTL_FNCIREF1 eq '%s'", invoiceID)
	expectedItems := 1

	if feesInvoiceID != "" {
		tfncItems2Filter += fmt.Sprintf(" or ROTL_FNCIREF1 eq '%s'", feesInvoiceID)
		expectedItems++
	}

	receiptForm := fmt.Sprintf("TINVOICES(IVNUM='%s',DEBIT='D',IVTYPE='T')", receipt.ID)
	params := map[string][]string{
		"$select": {"IVNUM,DEBIT,IVTYPE,CUSTNAME,CASHNAME,IVDATE"},
		"$expand": {fmt.Sprintf("TPAYMENT2_SUBFORM($select=QPRICE),TFNCITEMS2_SUBFORM($filter=(%s);$select=IVNUM,FNCTRANS,ROTL_FNCIREF1,KLINE,PAYFLAG)", tfncItems2Filter)},
	}

	if res, err := priority.Client.Get(company, receiptForm, params); err != nil {
		return fmt.Errorf("patch receipt failed: %s", err)
	} else if err := json.Unmarshal(res, receipt); err != nil {
		return err
	}

	if len(receipt.TFNCItems2Subform) != expectedItems {
		return ErrPaymentReceiptCreateFailed
	}

	// Mark invoices to be paid
	for _, item := range receipt.TFNCItems2Subform {
		data, err := json.Marshal(priorityDomain.TFNCItem2{
			PayFlag: priority.String("Y"),
		})
		if err != nil {
			return err
		}

		form := fmt.Sprintf("TINVOICES(IVNUM='%s',DEBIT='D',IVTYPE='T')/TFNCITEMS2_SUBFORM(FNCTRANS=%d,KLINE=%d)",
			receipt.ID, item.FNCTrans, item.KLine)

		if _, err := priority.Client.Patch(company, form, nil, data); err != nil {
			return fmt.Errorf("patch receipt failed: %s", err)
		}
	}

	// If receipt covers a partial payment, we need to update the original invoice credit
	if !isPartialPayment {
		return nil
	}

	var tfncItems priorityDomain.TFNCItems

	form := fmt.Sprintf("TINVOICES(IVNUM='%s',DEBIT='D',IVTYPE='T')/TFNCITEMS_SUBFORM", receipt.ID)
	params = map[string][]string{
		"$select": {"FNCIREF1,FNCTRANS,KLINE,CREDIT"},
	}

	if res, err := priority.Client.Get(company, form, params); err != nil {
		return fmt.Errorf("patch receipt failed: %s", err)
	} else if err := json.Unmarshal(res, &tfncItems); err != nil {
		return err
	}

	if len(tfncItems.Value) != expectedItems {
		return ErrPaymentReceiptUpdatePartial
	}

	// Update the amount paid for the original invoice

	var (
		tfncItem *priorityDomain.TFNCItem
		credit   = receipt.TPayment2Subform[0].Price
	)

	for _, v := range tfncItems.Value {
		switch v.FNCIREF1 {
		case invoiceID:
			tfncItem = v
		case feesInvoiceID:
			// If there are fees, we need to subtract them from the credit used for the original invoice
			credit -= v.Credit
		default:
			return ErrPaymentReceiptUpdatePartial
		}
	}

	if tfncItem == nil {
		return ErrPaymentReceiptUpdatePartial
	}

	data, err := json.Marshal(priorityDomain.TFNCItem{
		Credit: credit,
	})
	if err != nil {
		return err
	}

	form = fmt.Sprintf("TINVOICES(IVNUM='%s',DEBIT='D',IVTYPE='T')/TFNCITEMS_SUBFORM(FNCTRANS=%d,KLINE=%d)",
		receipt.ID, tfncItem.FNCTrans, tfncItem.KLine)

	if _, err := priority.Client.Patch(company, form, nil, data); err != nil {
		return fmt.Errorf("patch receipt failed: %s", err)
	}

	return nil
}

func createReceipt(invoice *invoices.FullInvoice, amount int64, receiptDate time.Time) (*priorityDomain.TInvoice, error) {
	receiptDateString := receiptDate.Format(time.RFC3339)

	data, _ := json.Marshal(priorityDomain.TInvoice{
		PriorityID: invoice.PriorityID,
		CashName:   cashName,
		Currency:   invoice.Currency,
		DateString: receiptDateString,
		TPayment2Subform: []*priorityDomain.TPayment2{
			{
				PaymentCode: paymentCode,
				Price:       float64(amount) / 100,
				Details:     priority.String(invoice.Details),
				PayDate:     receiptDateString,
			},
		},
	})

	resp, err := priority.Client.Post(invoice.Company, "TINVOICES", nil, data)
	if err != nil {
		return nil, err
	}

	var receipt priorityDomain.TInvoice
	if err := json.Unmarshal(resp, &receipt); err != nil {
		return nil, err
	}

	return &receipt, nil
}

func getOpenInvoice(invoice *invoices.FullInvoice, entity *common.Entity) (*priorityDomain.OpenInvoice, error) {
	params := map[string][]string{
		"$filter": {fmt.Sprintf("CUSTNAME eq '%s' and IVNUM eq '%s'", entity.PriorityID, invoice.ID)},
	}

	res, err := priority.Client.Get(entity.PriorityCompany, "ROTL_OPENIVS", params)
	if err != nil {
		return nil, err
	}

	var result priorityDomain.OpenInvoicesResponse

	if err := json.Unmarshal(res, &result); err != nil {
		return nil, err
	}

	if len(result.OpenInvoices) == 0 {
		// Invoice is marked as paid
		return nil, nil
	}

	var openInvoice *priorityDomain.OpenInvoice

	for _, v := range result.OpenInvoices {
		if openInvoice == nil {
			openInvoice = v
		} else {
			openInvoice.Debit += v.Debit
		}
	}

	return openInvoice, nil
}

func toStripeCurrency(symbol string) (stripe.Currency, error) {
	switch symbol {
	case "USD":
		return stripe.CurrencyUSD, nil
	case "ILS":
		return stripe.CurrencyILS, nil
	case "EUR":
		return stripe.CurrencyEUR, nil
	case "GBP":
		return stripe.CurrencyGBP, nil
	case "AUD":
		return stripe.CurrencyAUD, nil
	case "CAD":
		return stripe.CurrencyCAD, nil
	case "DKK":
		return stripe.CurrencyDKK, nil
	case "NOK":
		return stripe.CurrencyNOK, nil
	case "SEK":
		return stripe.CurrencySEK, nil
	case "BRL":
		return stripe.CurrencyBRL, nil
	case "SGD":
		return stripe.CurrencySGD, nil
	case "MXN":
		return stripe.CurrencyMXN, nil
	case "CHF":
		return stripe.CurrencyCHF, nil
	case "MYR":
		return stripe.CurrencyMYR, nil
	case "TWD":
		return stripe.CurrencyTWD, nil
	case "EGP":
		return stripe.CurrencyEGP, nil
	case "ZAR":
		return stripe.CurrencyZAR, nil
	case "JPY":
		return stripe.CurrencyJPY, nil
	case "IDR":
		return stripe.CurrencyIDR, nil
	default:
		return "", fmt.Errorf("invalid currency symbol %s", symbol)
	}
}

func lock(ctx context.Context, fs *firestore.Client, invoiceDocSnap *firestore.DocumentSnapshot) error {
	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceDocSnap.Ref)
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
		return tx.Update(invoiceDocSnap.Ref, []firestore.Update{
			{
				FieldPath: []string{"stripeLocked"},
				Value:     true,
			},
		})
	})

	return err
}

func unlock(ctx context.Context, l logger.ILogger, fs *firestore.Client, invoiceDocSnap *firestore.DocumentSnapshot) {
	err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(invoiceDocSnap.Ref)
		if err != nil {
			return err
		}

		if !docSnap.Exists() {
			// Invoice doc does not exist
			return nil
		}

		return tx.Update(invoiceDocSnap.Ref, []firestore.Update{
			{
				FieldPath: []string{"stripeLocked"},
				Value:     false,
			},
		})
	})
	if err != nil {
		l.Infof("%s : invoice unlock %s", invoiceDocSnap.Ref.ID, err)
	}
}

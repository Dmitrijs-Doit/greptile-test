package utils

import (
	"math"

	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
)

func ToCents(v float64) int64 {
	return int64(math.Round(v * 100))
}

func IsValidAmount(invoice *invoices.FullInvoice, amount, balance, openInvoiceDebit int64) bool {
	var processingAmount int64

	if invoice.StripePaymentIntents != nil {
		for _, pi := range invoice.StripePaymentIntents {
			if pi.Status == stripe.PaymentIntentStatusSucceeded || pi.Status == stripe.PaymentIntentStatusRequiresCapture {
				if pi.Debit == invoice.Debit {
					balance -= pi.AmountReceived
				}
			} else if pi.Status == stripe.PaymentIntentStatusProcessing {
				processingAmount += pi.Amount
			}
		}
	}

	// Check that the invoice is completely paid or at least 5.00 (of currency) remaining
	// stripe has a minimum charge amount
	var balanceAfterPayment = balance - processingAmount - amount

	return balance == openInvoiceDebit && (balanceAfterPayment == 0 || balanceAfterPayment >= 500)
}

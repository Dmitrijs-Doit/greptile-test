package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type PayInvoiceInput struct {
	Type            common.EntityPaymentType `json:"type"`
	Amount          float64                  `json:"amount"`
	PaymentMethodID string                   `json:"payment_method_id"`

	UserID     string `json:"-"`
	Email      string `json:"-"`
	CustomerID string `json:"-"`
	EntityID   string `json:"-"`
	InvoiceID  string `json:"-"`
}

func (s *StripeService) PayInvoice(ctx context.Context, input PayInvoiceInput, entity *common.Entity) error {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	docID := fmt.Sprintf("%s-%s-%s", entity.PriorityCompany, entity.PriorityID, input.InvoiceID)
	l.Infof("invoice id: %s", docID)

	invoiceDocSnap, err := fs.Collection("invoices").Doc(docID).Get(ctx)
	if err != nil {
		return err
	}

	amount := stripe.Int64(int64(math.Round(input.Amount * 100)))
	today := time.Now().UTC().Truncate(24 * time.Hour)

	switch input.Type {
	case common.EntityPaymentTypeCard:
		if err := s.makeCreditCardPayment(ctx, input.Email, entity, invoiceDocSnap, today, input.PaymentMethodID, amount); err != nil {
			return err
		}
	case common.EntityPaymentTypeUSBankAccount, common.EntityPaymentTypeBankAccount:
		if err := s.makeACHPayment(ctx, input.Email, entity, invoiceDocSnap, today, input.PaymentMethodID, amount); err != nil {
			return err
		}
	case common.EntityPaymentTypeSEPADebit:
		if err := s.makeSEPADebitPayment(ctx, input.Email, entity, invoiceDocSnap, today, input.PaymentMethodID, amount); err != nil {
			return err
		}
	case common.EntityPaymentTypeBACSDebit:
		if err := s.makeBACSDebitPayment(ctx, input.Email, entity, invoiceDocSnap, today, input.PaymentMethodID, amount); err != nil {
			return err
		}
	case common.EntityPaymentTypeACSSDebit:
		if err := s.makeACSSDebitPayment(ctx, input.Email, entity, invoiceDocSnap, today, input.PaymentMethodID, amount); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid payment type: %s", input.Type)
	}

	return nil
}

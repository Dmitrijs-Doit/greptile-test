package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type AutomaticPaymentsInput struct {
	PaymentMethodTypes   []common.EntityPaymentType `json:"payment_method_types"`
	OverdueDaysRemainder int                        `json:"overdue_days_remainder"`
}

type AutomaticPaymentsEntityWorkerInput struct {
	AutomaticPaymentsInput
	StripeAccountID domain.StripeAccountID `json:"stripe_account_id"`
	EntityID        string                 `json:"entity_id"`
}

const paymentRateLimit = time.Millisecond * 250

// AutomaticPayments charges invoices on their due date or when overdue days remainder is reached
func (s *StripeService) AutomaticPayments(ctx context.Context, input AutomaticPaymentsInput) error {
	l := s.loggerProvider(ctx)

	// If no payment method types are specified for the job, run for all available types
	if len(input.PaymentMethodTypes) == 0 {
		input.PaymentMethodTypes = []common.EntityPaymentType{
			common.EntityPaymentTypeCard,
			common.EntityPaymentTypeSEPADebit,
			common.EntityPaymentTypeBankAccount,
			common.EntityPaymentTypeUSBankAccount,
			common.EntityPaymentTypeBACSDebit,
			common.EntityPaymentTypeACSSDebit,
		}
	}

	// If overdue days remainder is not specified, charge for overdue invoices every week
	if input.OverdueDaysRemainder < 0 {
		input.OverdueDaysRemainder = 7
	}

	entities, err := s.entitiesDAL.ListActiveEntitiesForPayments(ctx, s.stripeClient.accountID, input.PaymentMethodTypes)
	if err != nil {
		return err
	}

	for _, entity := range entities {
		// If entity has no default payment method of the specified type, skip
		switch entity.Payment.Type {
		case common.EntityPaymentTypeCard:
			if entity.Payment.Card == nil {
				l.Infof("entity %s has no default CC payment method", entity.PriorityID)
				continue
			}
		case common.EntityPaymentTypeSEPADebit:
			if entity.Payment.SEPADebit == nil {
				l.Infof("entity %s has no default SEPA debit payment method", entity.PriorityID)
				continue
			}
		case common.EntityPaymentTypeBACSDebit:
			if entity.Payment.BACSDebit == nil {
				l.Infof("entity %s has no default BACS debit payment method", entity.PriorityID)
				continue
			}
		case common.EntityPaymentTypeACSSDebit:
			if entity.Payment.ACSSDebit == nil {
				l.Infof("entity %s has no default ACSS debit payment method", entity.PriorityID)
				continue
			}
		case common.EntityPaymentTypeBankAccount, common.EntityPaymentTypeUSBankAccount:
			if entity.Payment.BankAccount == nil {
				l.Infof("entity %s has no default ACH debit payment method", entity.PriorityID)
				continue
			}
		}

		entityID := entity.Snapshot.Ref.ID

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/stripe/payments/entities/%s", entityID),
			Queue:  common.TaskQueueStripePayments,
		}

		body := AutomaticPaymentsEntityWorkerInput{
			AutomaticPaymentsInput: input,
			StripeAccountID:        s.stripeClient.accountID,
			EntityID:               entityID,
		}

		_, err := s.Connection.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(body))
		if err != nil {
			l.Errorf("failed to create task for entity %s with error: %s", entity.PriorityID, err)
			continue
		}
	}

	return nil
}

// AutomaticPaymentsEntityWorker charges an entity's invoices on their due date or when overdue days remainder is reached
func (s *StripeService) AutomaticPaymentsEntityWorker(ctx context.Context, input AutomaticPaymentsEntityWorkerInput) error {
	l := s.loggerProvider(ctx)
	fs := s.Connection.Firestore(ctx)

	today := times.CurrentDayUTC()
	entityID, overdueDaysRemainder := input.EntityID, input.OverdueDaysRemainder
	entityRef := s.entitiesDAL.GetRef(ctx, entityID)

	entity, err := s.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		l.Errorf("failed to get entity %s", entityID)
		return err
	}

	l.Infof("processing payments for entity %s priority ID %s", entityID, entity.PriorityID)

	invoiceDocSnaps, err := fs.Collection("invoices").
		Where("entity", "==", entityRef).
		Where("PAID", "==", false).
		Where("CANCELED", "==", false).
		Where("DEBIT", ">", 0).
		Where("PAYDATE", "<=", today).
		OrderBy("DEBIT", firestore.Asc).
		OrderBy("PAYDATE", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		l.Errorf("failed to get open invoices for billing profile %s", entity.PriorityID)
		return err
	}

	if len(invoiceDocSnaps) <= 0 {
		l.Infof("no unpaid invoices for %s", entity.PriorityID)
		return nil
	}

	l.Infof("fetched %d unpaid invoices for %s", len(invoiceDocSnaps), entity.PriorityID)

	for _, invoiceDocSnap := range invoiceDocSnaps {
		var invoice invoices.FullInvoice

		if err := invoiceDocSnap.DataTo(&invoice); err != nil {
			l.Errorf("failed to populate invoice %s data with error: %s", invoiceDocSnap.Ref.ID, err)
			continue
		}

		if invoice.StripeLocked {
			l.Warningf("invoice %s is locked for payments", invoiceDocSnap.Ref.ID)
			continue
		}

		// Proceed with payment if today is the pay date or every 7 days after pay date
		payDate := invoice.PayDate.UTC().Truncate(common.DayDuration)
		daysSincePayDate := int(today.Sub(payDate).Hours() / 24)
		shouldCharge := today.Equal(payDate) || (today.After(payDate) && daysSincePayDate%overdueDaysRemainder == 0)

		if !shouldCharge {
			continue
		}

		switch entity.Payment.Type {
		case common.EntityPaymentTypeCard:
			if err := s.makeCreditCardPayment(ctx, paymentsEmail, entity, invoiceDocSnap, today, entity.Payment.Card.ID, nil); err != nil {
				if stripeErr, ok := err.(*stripe.Error); ok {
					// Insufficient funds error can still try to charge other invoices for this entity
					if stripeErr.Type == stripe.ErrorTypeCard && stripeErr.Code == stripe.ErrorCodeCardDeclined {
						if err, ok := stripeErr.Err.(*stripe.CardError); ok && err.DeclineCode == stripe.DeclineCodeInsufficientFunds {
							l.Warningf("CC payment for invoice %s failed with insufficient funds", invoiceDocSnap.Ref.ID)
							continue
						}
					}

					l.Errorf("CC payment for invoice %s failed with stripe error [code: %s]: %s", invoiceDocSnap.Ref.ID, stripeErr.Code, stripeErr.Msg)

					// If payment failed because of stripe error other than insufficient funds,
					// stop trying to process payments for this entity
					return nil
				}

				l.Errorf("CC payment for invoice %s failed with error: %s", invoiceDocSnap.Ref.ID, err)

				continue
			}

			l.Infof("CC payment for invoice %s succeeded", invoiceDocSnap.Ref.ID)
		case common.EntityPaymentTypeSEPADebit:
			if err := s.makeSEPADebitPayment(ctx, paymentsEmail, entity, invoiceDocSnap, today, entity.Payment.SEPADebit.ID, nil); err != nil {
				l.Errorf("SEPA payment for invoice %s failed with error: %s", invoiceDocSnap.Ref.ID, err)
				break
			}

			l.Infof("SEPA payment for invoice %s succeeded", invoiceDocSnap.Ref.ID)
		case common.EntityPaymentTypeBACSDebit:
			if err := s.makeBACSDebitPayment(ctx, paymentsEmail, entity, invoiceDocSnap, today, entity.Payment.BACSDebit.ID, nil); err != nil {
				l.Errorf("BACS payment for invoice %s failed with error: %s", invoiceDocSnap.Ref.ID, err)
				break
			}

			l.Infof("BACS payment for invoice %s succeeded", invoiceDocSnap.Ref.ID)
		case common.EntityPaymentTypeACSSDebit:
			if err := s.makeACSSDebitPayment(ctx, paymentsEmail, entity, invoiceDocSnap, today, entity.Payment.ACSSDebit.ID, nil); err != nil {
				l.Errorf("ACSS payment for invoice %s failed with error: %s", invoiceDocSnap.Ref.ID, err)
				break
			}

			l.Infof("ACSS payment for invoice %s succeeded", invoiceDocSnap.Ref.ID)
		case common.EntityPaymentTypeBankAccount, common.EntityPaymentTypeUSBankAccount:
			if err := s.makeACHPayment(ctx, paymentsEmail, entity, invoiceDocSnap, today, entity.Payment.BankAccount.ID, nil); err != nil {
				l.Errorf("ACH payment for invoice %s failed with error: %s", invoiceDocSnap.Ref.ID, err)
				break
			}

			l.Infof("ACH payment for invoice %s succeeded", invoiceDocSnap.Ref.ID)

		default:
			return fmt.Errorf("invalid payment method type %s for entity %s", entity.Payment.Type, entity.PriorityID)
		}

		// Rate limit payments to avoid hitting priority rate limits
		time.Sleep(paymentRateLimit)
	}

	return nil
}

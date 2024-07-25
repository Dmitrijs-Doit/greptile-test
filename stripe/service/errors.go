package service

import (
	"errors"
)

var (
	ErrStripeSettings              = errors.New("failed to get stripe settings")
	ErrOnlinePaymentsUnavailable   = errors.New("online payments are not available at this time")
	ErrPaymentMethodDisabled       = errors.New("payment method is disabled")
	ErrPaymentInvoiceNotFound      = errors.New("invoice not found")
	ErrPaymentInvoiceLocked        = errors.New("invoice is locked")
	ErrPaymentReceiptCreateFailed  = errors.New("invoice not found in receipt subform")
	ErrPaymentReceiptUpdatePartial = errors.New("failed to update partial receipt")
	ErrInvalidPaymentMethodType    = errors.New("invalid payment method type")
	ErrBadRequestCard              = errors.New("invalid payment method request (card)")
	ErrBadRequestBankAccount       = errors.New("invalid payment method request (bank account)")
	ErrBadRequestSEPADebit         = errors.New("invalid payment method request (SEPA debit)")
	ErrBadRequestBACSDebit         = errors.New("invalid payment method request (BACS debit)")
	ErrInvalidInvoice              = errors.New("invalid invoice")
	ErrInvalidCurrency             = errors.New("invalid currency")
	ErrInvalidCard                 = errors.New("invalid credit card")
	ErrCustomerNotFound            = errors.New("customer not found")
	ErrInvalidUser                 = errors.New("invalid user")
	ErrCreateSetupIntent           = errors.New("failed to create setup intent")
	ErrCreateSession               = errors.New("failed to create session")
	ErrCreateStripeCustomer        = errors.New("failed to create stripe customer")
	ErrEntityNotFound              = errors.New("entity not found")
	ErrDoitCustomerNotFound        = errors.New("doit customer not found")
	ErrDetachPaymentMethod         = errors.New("failed to detach payment method")
)

package dal

import (
	"errors"
)

var (
	// TODO(yoni): move all the errors that are not used in the service anymore to this file, once we have the dal completed
	ErrPaymentInvoiceNotFound       = errors.New("invoice not found")
	ErrPaymentInvoiceLocked         = errors.New("invoice is locked")
	ErrFailedToUnlockInvoice        = errors.New("failed to unlock invoice")
	ErrInvoicePaymentIntentNotFound = errors.New("invoice payment not found")
)

package handlers

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidAccountIDFormat = func(row int, accountID string) error {
		return fmt.Errorf(`row %d: account ID "%s" must be a 12 digit number`, row, accountID)
	}
	ErrInvalidSupportLevel = func(row int, supportLevel string) error {
		return fmt.Errorf(`row %d: invalid support level "%s". It must be one of: basic, developer, business, enterprise`, row, supportLevel)
	}
	ErrInvalidPayerIDFormat = func(row int, payerID string) error {
		return fmt.Errorf(`row %d: payer ID "%s" must be a 12 digit number`, row, payerID)
	}
	ErrInvalidMonthFormat = func(invoiceMonth string) error {
		return fmt.Errorf(`invalid month "%s". Invoice month must use format YYYY-MM`, invoiceMonth)
	}
	ErrMonthNotCurrentOrPrevious = func(invoiceMonth string) error {
		return fmt.Errorf(`invalid month "%s". Invoice month must be the current or previous month`, invoiceMonth)
	}
	ErrInvalidAccountName     = func(row int) error { return fmt.Errorf("row %d: account name must be a non-empty string", row) }
	ErrInvalidNumberOfColumns = func(row int) error { return fmt.Errorf("row %d: invalid number of columns, wanted 4", row) }
	ErrInvalidHeaders         = errors.New("invalid CSV file: expected headers to be 'account_name', 'account_id', 'support_level', 'payer_id'")
	ErrInvoiceAlreadyIssued   = errors.New("an invoice has already been issued for the given month")
)

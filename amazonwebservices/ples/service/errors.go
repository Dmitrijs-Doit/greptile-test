package service

import (
	"fmt"
)

var (
	ErrPayerIDDoesNotExist = func(row int, payerID string) error {
		return fmt.Errorf(`row %d: payer_id "%s" does not exist`, row, payerID)
	}
	ErrInvalidFlexsaveAccountName = func(row int, accountName string) error {
		return fmt.Errorf(`row %d: "%s" flexsave account name must start with "fs"`, row, accountName)
	}
	ErrInvalidAccountName    = func(row int) error { return fmt.Errorf("row %d: account name must be a non-empty string", row) }
	ErrAccountIDDoesNotExist = func(row int, accountID string) error {
		return fmt.Errorf(`row %d: account ID "%s" does not exist`, row, accountID)
	}
	ErrPayerIDNotInRequest = func(payerID string) error {
		return fmt.Errorf(`payer account "%s" does not appear in the request`, payerID)
	}
)

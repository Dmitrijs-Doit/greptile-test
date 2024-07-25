package domain

import "errors"

var (
	ErrAccountUserMissing           = errors.New("procurement account has no user")
	ErrAccountCustomerMissing       = errors.New("procurement account has no customer")
	ErrAccountBillingAccountMissing = errors.New("procurement account has no billing account")
)

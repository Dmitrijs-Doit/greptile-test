package dal

import (
	"errors"
)

var (
	ErrInvalidEntityID             = errors.New("invalid entity id")
	ErrInvalidCustomerRef          = errors.New("invalid customer ref")
	ErrInvalidEmptyStripeAccountID = errors.New("invalid empty stripe account id")
	ErrInvalidEmptyPaymentTypes    = errors.New("invalid empty payment types")
)

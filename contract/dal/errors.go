package dal

import "errors"

var (
	ErrUndefinedCustomerRef  = errors.New("customer ref cannot be empty")
	ErrUndefinedContractType = errors.New("contract type cannot be empty")
	ErrUndefinedPaymentTerm  = errors.New("payment term cannot be empty")
)

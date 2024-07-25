package service

import "errors"

var (
	ErrCustomerHasBillingProfiles        = errors.New("customer has billing profiles")
	ErrCustomerHasContracts              = errors.New("customer has contracts")
	ErrCustomerHasAssets                 = errors.New("customer has assets")
	ErrCustomerHasInvoices               = errors.New("customer has invoices")
	ErrCustomerHasUsers                  = errors.New("customer has users")
	ErrCustomerHasGCPMarketplaceAccounts = errors.New("customer has gcp marketplace accounts")
	ErrCustomerHasAWSMarketplaceAccounts = errors.New("customer has aws marketplace accounts")
	ErrCustomerIsNotEmpty                = errors.New("customer is not empty")
)

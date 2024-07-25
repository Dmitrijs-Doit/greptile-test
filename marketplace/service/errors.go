package service

import "errors"

var (
	ErrEntitlementStatusIsNotPending   = errors.New("entitlement status is not pending")
	ErrProcurementAccountIsNotApproved = errors.New("procurement account is not approved")
	ErrGetAccountFirestore             = errors.New("failed to get account from firestore")
	ErrBillingAccountNotFound          = errors.New("billing account not found in marketplace")
	ErrBillingAccountTypeUnknown       = errors.New("billing account type unknown")

	ErrCustomerAlreadySubscribed        = errors.New("customer is already subscribed")
	ErrCustomerIsNotEligibleFlexsave    = errors.New("customer is not eligible for flexsave on marketplace")
	ErrCustomerIsNotEligibleCostAnomaly = errors.New("customer is not eligible for cost-anomaly on marketplace")
	ErrCustomerNotStandalone            = errors.New("customer is not a standalone customer")
	ErrBillingAccountMismatch           = errors.New("billing account id mismatch")
	ErrEntitlementNotFound              = errors.New("entitlement not found")
	ErrFlexsaveProductIsDisabled        = errors.New("flexsave product is disabled")
)

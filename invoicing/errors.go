package invoicing

import "errors"

var (
	ErrPLPSRowButNoPLPSChargesTpl     = "encountered a PLPS row but don't have any PLPS charges data for the billing account: %s, in the period from: %s to: %s"
	ErrNoSuitablePLPSContractFoundTpl = "no suitable PLPS contract found for the billing account: %s, in the period from: %s to %s"

	ErrUnknownBillingDataItemFieldID = errors.New("unknown billing data item field id")
	ErrNoSuitablePLPSContractFound   = errors.New("no suitable PLPS contract found")
)

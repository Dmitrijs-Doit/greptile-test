package service

import "errors"

var (
	ErrLogIsNil                          = errors.New("log is nil")
	ErrPriorityReaderWriterIsNil         = errors.New("priority reader writer is nil")
	ErrPriorityInvoiceIdentifierNotValid = errors.New("priority invoice identifier is not valid")

	ErrInvoiceCurrencyDoesNotMatch = errors.New("invoice does not match the customer's currency")
	ErrAvalaraNotAvailable         = errors.New("avalara is not available try again later")
	ErrAvalaraTaxNotCalculated     = errors.New("avalara tax was not calculated")
)

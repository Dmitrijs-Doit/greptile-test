package service

import "errors"

var (
	ErrInvalidReportType  = errors.New("invalid report type")
	ErrInvalidReportID    = errors.New("invalid report id")
	ErrInvalidCustomerID  = errors.New("invalid customer id")
	ErrUnauthorizedDelete = errors.New("user does not have required permissions to delete this report")
	ErrInternalToExternal = errors.New(ErrExternalFromInternalValidationErrorsMsg)
)

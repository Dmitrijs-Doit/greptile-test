package handlers

import "errors"

var (
	ErrUnmarshalCmpEntitlementApproveRequestedEvent = errors.New("error unmarshalling cmp entitlement approve requested event")
	ErrUnmarshalCmpEntitlementCancelledEvent        = errors.New("error unmarshalling cmp entitlement cancelled event")
	ErrInvalidPayload                               = errors.New("invalid payload")
)

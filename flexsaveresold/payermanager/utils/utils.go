package utils

const (
	ActiveState   = "active"
	PendingState  = "pending"
	DisabledState = "disabled"

	ActiveToPending   = "activeToPending"
	DisabledToPending = "disabledToPending"
	DisabledToActive  = "disabledToActive"
	PendingToActive   = "pendingToActive"
	PendingToDisabled = "pendingToDisabled"
	ActiveToDisabled  = "activeToDisabled"
	InvalidTrigger    = "invalidTrigger"
	StayWithinState   = "stayWithinState"
)

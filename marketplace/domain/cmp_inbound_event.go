package domain

type CmpInboundEventType string

const (
	CmpInboundEventTypeEntitlementApproveRequested CmpInboundEventType = "CMP_ENTITLEMENT_APPROVE_REQUESTED"
	CmpInboundEventTypeEntitlementCancelled        CmpInboundEventType = "CMP_ENTITLEMENT_CANCELLED"
)

type CmpInboundBaseEvent struct {
	EventID   string              `json:"eventId" validate:"required"`
	EventType CmpInboundEventType `json:"eventType" validate:"required"`
}

type CmpEntitlementApproveRequestedEvent struct {
	CmpInboundBaseEvent
	Entitlement struct {
		ID string `json:"id" validate:"required"`
	} `json:"entitlement" validate:"required"`
}

type CmpEntitlementCancelledEvent struct {
	CmpInboundBaseEvent
	Entitlement struct {
		ID string `json:"id" validate:"required"`
	} `json:"entitlement" validate:"required"`
}

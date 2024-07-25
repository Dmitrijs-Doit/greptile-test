package domain

import (
	"fmt"

	"github.com/google/uuid"
)

type CmpOutboundEventType string

const (
	CmpOutboundEventTypeAccountApproveRequested CmpOutboundEventType = "CMP_ACCOUNT_APPROVE_REQUESTED"
)

type CmpOutboundBaseEvent struct {
	EventID   string               `json:"eventId"`
	EventType CmpOutboundEventType `json:"eventType"`
}

type ID struct {
	ID string `json:"id"`
}

type CmpOutboundAccountApprovalRequestEvent struct {
	CmpOutboundBaseEvent
	Account ID `json:"account"`
}

func NewOutboundBaseEvent(eventType CmpOutboundEventType) (*CmpOutboundBaseEvent, error) {
	eventID, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid for event: %s, with error: %s", eventType, err)
	}

	return &CmpOutboundBaseEvent{
		EventID:   eventID.String(),
		EventType: eventType,
	}, nil
}

func NewAccountApproveRequestEvent(procurementAccountID string) (*CmpOutboundAccountApprovalRequestEvent, error) {
	baseEvent, err := NewOutboundBaseEvent(CmpOutboundEventTypeAccountApproveRequested)
	if err != nil {
		return nil, err
	}

	event := CmpOutboundAccountApprovalRequestEvent{
		CmpOutboundBaseEvent: *baseEvent,
		Account: ID{
			ID: procurementAccountID,
		},
	}

	return &event, nil
}

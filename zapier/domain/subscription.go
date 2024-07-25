package domain

import (
	"time"

	"cloud.google.com/go/firestore"
)

type EventType string

const (
	AlertConditionSatisfied EventType = "alertConditionSatisfied"
	BudgetThresholdAchieved EventType = "budgetThresholdAchieved"
)

type WebhookSubscription struct {
	ID           string                 `firestore:"-"`
	Customer     *firestore.DocumentRef `firestore:"customer"`
	UserEmail    string                 `firestore:"userEmail"`
	EventType    EventType              `firestore:"eventType"`
	ItemID       string                 `firestore:"itemId"`
	TargetURL    string                 `firestore:"targetUrl"`
	TimeCreated  time.Time              `firestore:"timeCreated,serverTimestamp"`
	TimeModified time.Time              `firestore:"timeModified,serverTimestamp"`
}

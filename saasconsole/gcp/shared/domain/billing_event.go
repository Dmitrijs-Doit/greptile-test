package domain

import (
	"time"

	"cloud.google.com/go/firestore"
)

type BillingUpdateEvent string

const (
	BillingUpdateEventOnboarding  BillingUpdateEvent = "onboarding"
	BillingUpdateEventOffboarding BillingUpdateEvent = "offboarding"
	BillingUpdateEventBackfill    BillingUpdateEvent = "backfill"
)

const (
	BillingUpdateCollection = "cloudAnalytics/standalone/cloudAnalyticsGcpStandaloneBillingCopy"
)

type BillingEvent struct {
	BillingAccountID string             `firestore:"billingAccountId"`
	TimeCreated      *time.Time         `firestore:"timeCreated"`
	TimeCompleted    *time.Time         `firestore:"timeCompleted"`
	EventType        BillingUpdateEvent `firestore:"eventType"`
	EventRange       Range              `firestore:"range"`

	Snapshot *firestore.DocumentSnapshot `firestore:"-"`
}

type Range struct {
	StartTime *time.Time `firestore:"startTime"`
	EndTime   *time.Time `firestore:"endTime"`
}

func (e *BillingEvent) ID() string {
	return e.Snapshot.Ref.ID
}

package aws

import (
	"time"

	"cloud.google.com/go/firestore"
)

type CustomerCreditAmazonWebServices struct {
	Name          string                        `firestore:"name"`
	Type          string                        `firestore:"type"`
	Customer      *firestore.DocumentRef        `firestore:"customer"`
	Entity        *firestore.DocumentRef        `firestore:"entity"`
	Assets        []*firestore.DocumentRef      `firestore:"assets"`
	Currency      string                        `firestore:"currency"`
	Amount        float64                       `firestore:"amount"`
	StartDate     time.Time                     `firestore:"startDate"`
	EndDate       time.Time                     `firestore:"endDate"`
	DepletionDate *time.Time                    `firestore:"depletionDate"`
	Utilization   map[string]map[string]float64 `firestore:"utilization"`
	Metadata      map[string]interface{}        `firestore:"metadata"`
	Alerts        map[string]interface{}        `firestore:"alerts"`
	UpdatedBy     map[string]interface{}        `firestore:"updatedBy"`
	Timestamp     time.Time                     `firestore:"timestamp,serverTimestamp"`

	// Utility fields
	Remaining              float64                     `firestore:"-"`
	RemainingPreviousMonth float64                     `firestore:"-"`
	Touched                bool                        `firestore:"-"`
	Snapshot               *firestore.DocumentSnapshot `firestore:"-"`
}

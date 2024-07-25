package invoicing

import (
	"time"

	"cloud.google.com/go/firestore"
)

type CustomerCreditGoogleCloud struct {
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
	Scope         []string                      `firestore:"scope"`

	// Utility fields
	Remaining              float64                     `firestore:"-"`
	RemainingPreviousMonth float64                     `firestore:"-"`
	Touched                bool                        `firestore:"-"`
	Snapshot               *firestore.DocumentSnapshot `firestore:"-"`
}

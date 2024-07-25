package domain

import "time"

type CustomerMetadata struct {
	CustomerID string `firestore:"customer_id"`
	Name       string `firestore:"customer_name"`
	PriorityID string `firestore:"priority_id"`
	EntityID   string `firestore:"entity_id"`
}

type Customer struct {
	ID        string           `firestore:"id"`
	AccountID StripeAccountID  `firestore:"account_id"`
	Email     string           `firestore:"email"`
	LiveMode  bool             `firestore:"livemode"`
	Metadata  CustomerMetadata `firestore:"metadata"`
	Timestamp time.Time        `firestore:"timestamp,serverTimestamp"`
}

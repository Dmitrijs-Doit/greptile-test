package domain

import "time"

type PriorityStatus string

const (
	ApprovedStatus PriorityStatus = "Approved"
	DraftStatus    PriorityStatus = "Draft"
	FinalStatus    PriorityStatus = "Final"
	DeleteStatus   PriorityStatus = "Delete"
)

func (as PriorityStatus) String() string {
	return string(as)
}

type AvalaraStatus struct {
	Healthy   bool      `firestore:"healthy"`
	Timestamp time.Time `firestore:"timestamp"`
	Locked    bool      `firestore:"locked"`
}

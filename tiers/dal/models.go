package dal

import (
	"time"
)

type CustomerTrialNotifications struct {
	CustomerID string               `firestore:"-"`
	LastSent   map[string]time.Time `firestore:"lastSent"`
	UsageSent  map[string][]string  `firestore:"usageSent"`
}

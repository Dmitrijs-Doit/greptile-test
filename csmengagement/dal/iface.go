package dal

import "time"

type EngagementDetails struct {
	CustomerID    string      `firestore:"CustomerID"`
	NotifiedDates []time.Time `firestore:"NotifiedDates"`
}

func (e EngagementDetails) WasNotifiedAboutWithinLastMonth() bool {
	oneMonthAgo := time.Now().AddDate(0, -1, 0)

	for _, notifiedDate := range e.NotifiedDates {
		if notifiedDate.After(oneMonthAgo) {
			return true
		}
	}

	return false
}

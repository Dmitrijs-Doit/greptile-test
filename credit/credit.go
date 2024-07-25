package credit

import "cloud.google.com/go/firestore"

type BaseCredit struct {
	Customer    *firestore.DocumentRef        `firestore:"customer"`
	Name        string                        `firestore:"name"`
	Type        string                        `firestore:"type"`
	Utilization map[string]map[string]float64 `firestore:"utilization"`
}

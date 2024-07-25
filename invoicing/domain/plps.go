package domain

import "time"

const (
	// If these can change over time we should consider storing the data in firestore.
	PLPSSkuID                  string  = "213C-8958-B386"
	GooglePLPSChargePercentage float64 = 3
)

type PLPSCharge struct {
	PLPSPercent float64
	StartDate   time.Time
	EndDate     time.Time
}

type SortablePLPSCharges []*PLPSCharge

func (sp SortablePLPSCharges) Len() int {
	return len(sp)
}

func (sp SortablePLPSCharges) Less(i, j int) bool {
	return sp[i].StartDate.Before(sp[j].StartDate)
}

func (sp SortablePLPSCharges) Swap(i, j int) {
	sp[i], sp[j] = sp[j], sp[i]
}

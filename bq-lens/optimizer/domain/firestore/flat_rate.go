package firestoremodels

import "time"

type FieldDetail struct {
	Order       int    `firestore:"order"`
	Sign        string `firestore:"sign"`
	Title       string `firestore:"title"`
	Visible     bool   `firestore:"visible"`
	IsPartition bool   `firestore:"isPartition"`
}

type TimeSeriesData struct {
	XAxis []string  `firestore:"xAxis"`
	Bar   []float64 `firestore:"bar"`
	Line  []float64 `firestore:"line"`
}

type ScheduledQueriesMovement struct {
	DetailedTable              []ScheduledQueriesDetailTable `firestore:"detailedTable"`
	DetailedTableFieldsMapping map[string]FieldDetail        `firestore:"detailedTableFieldsMapping"`
	Recommendation             string                        `firestore:"recommendation"`
	SavingsPercentage          float64                       `firestore:"savingsPercentage"`
	SavingsPrice               float64                       `firestore:"savingsPrice"`
}

type ScheduledQueriesDetailTable struct {
	AllJobs          int64   `firestore:"allJobs"`
	BillingProjectID string  `firestore:"billingProjectId"`
	JobID            string  `firestore:"jobId"`
	Location         string  `firestore:"location"`
	ScheduledTime    string  `firestore:"scheduledTime"`
	Slots            float64 `firestore:"slots"`
}

type UserSlots struct {
	UserID     string                  `firestore:"userId,omitempty"`
	Slots      float64                 `firestore:"slots,omitempty"`
	TopQueries map[string]UserTopQuery `firestore:"topQueries,omitempty"`
	LastUpdate time.Time               `firestore:"lastUpdate,omitempty"`
}

type UserTopQuery struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	Location    string  `firestore:"location"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}

type BillingProject struct {
	BillingProjectID string                                 `firestore:"billingProjectId,omitempty"`
	Slots            float64                                `firestore:"slots,omitempty"`
	TopUsers         map[string]float64                     `firestore:"topUsers,omitempty"`
	TopQuery         map[string]BillingProjectSlotsTopQuery `firestore:"topQueries,omitempty"`
	LastUpdate       time.Time                              `firestore:"lastUpdate,omitempty"`
}

type BillingProjectSlotsTopQuery struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	Location    string  `firestore:"location"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}

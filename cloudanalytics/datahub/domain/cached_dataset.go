package domain

import "time"

type CachedDataset struct {
	Dataset     string    `json:"dataset"     bigquery:"cloud"       firestore:"cloud"`
	UpdatedBy   string    `json:"updatedBy"   bigquery:"updatedBy"   firestore:"updatedBy"`
	Records     int64     `json:"records"     bigquery:"records"     firestore:"records"`
	LastUpdated time.Time `json:"lastUpdated" bigquery:"lastUpdated" firestore:"lastUpdated"`
	// enriched fields
	Description string `json:"description" firestore:"-"`
}

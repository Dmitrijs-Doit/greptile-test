package domain

import "time"

type DatasetBatch struct {
	Batch       string    `json:"batch"        bigquery:"batch"       firestore:"batch"`
	Origin      string    `json:"origin"       bigquery:"source"      firestore:"source"`
	Records     int64     `json:"records"      bigquery:"records"     firestore:"records"`
	SubmittedBy string    `json:"submittedBy"  bigquery:"updatedBy"   firestore:"updatedBy"`
	SubmittedAt time.Time `json:"submittedAt"  bigquery:"lastUpdated" firestore:"lastUpdated"`
}

type DatasetBatchesRes struct {
	Items    []DatasetBatch `json:"items"    firestore:"items"`
	CachedAt time.Time      `json:"cachedAt" firestore:"cachedAt"`
}

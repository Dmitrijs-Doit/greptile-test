package domain

import "time"

type CachedDatasetsRes struct {
	Items    []CachedDataset `json:"items"    firestore:"items"`
	CachedAt time.Time       `json:"cachedAt" firestore:"cachedAt"`
}

package bqmodels

import (
	"time"

	"cloud.google.com/go/bigquery"
)

type CheckCompleteDaysResult struct {
	Min bigquery.NullTimestamp `bigquery:"min"`
	Max bigquery.NullTimestamp `bigquery:"max"`
}

type StorageRecommendationsResult struct {
	ProjectID        string               `bigquery:"projectId"`
	DatasetID        string               `bigquery:"datasetId"`
	TableID          string               `bigquery:"tableId"`
	TableIDBaseName  string               `bigquery:"tableIdBaseName"`
	TableCreateDate  time.Time            `bigquery:"tableCreateDate"`
	StorageSizeTB    bigquery.NullFloat64 `bigquery:"storageSizeTB"`
	Cost             bigquery.NullFloat64 `bigquery:"cost"`
	TotalStorageCost bigquery.NullFloat64 `bigquery:"totalStorageCost"`
}

type ScanPricePerPeriod struct {
	TotalUpTo30DaysAgo bigquery.NullFloat64 `bigquery:"total_up_to_30_days_ago"`
	TotalUpTo7DaysAgo  bigquery.NullFloat64 `bigquery:"total_up_to_7_days_ago"`
	TotalUpTo1DayAgo   bigquery.NullFloat64 `bigquery:"total_up_to_1_day_ago"`
}

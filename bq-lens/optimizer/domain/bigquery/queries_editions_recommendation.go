package bqmodels

type AggregatedJobStatistic struct {
	Location         string `bigquery:"location"`
	ProjectID        string `bigquery:"projectId"`
	Reservation      string `bigquery:"reservation"`
	TotalSlotsMS     int    `bigquery:"totalSlotMs"`
	TotalBilledBytes int    `bigquery:"totalBilledBytes"`
}

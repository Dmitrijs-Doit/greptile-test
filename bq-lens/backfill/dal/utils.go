package dal

import (
	"encoding/json"
	"time"

	"cloud.google.com/go/bigquery"
)

func convertJobToHistoricalJobsRow(job *bigquery.Job) (*HistoricalJobsRow, error) {
	jobStatus := job.LastStatus()
	jobStatistics := jobStatus.Statistics

	jobConfiguration, err := job.Config()
	if err != nil {
		return nil, err
	}

	var (
		query            string
		jobType          string
		property         string
		totalBytesBilled int64
		billingTier      int64
		cacheHit         bool
		referencedTables []*bigquery.Table
		timeline         string
		queryPlan        []bigquery.NullString
		totalSlotMs      int64
		statistics       string
		configuration    string
		status           string
	)

	switch jobConfiguration.(type) {
	case *bigquery.QueryConfig:
		query = jobConfiguration.(*bigquery.QueryConfig).Q
		jobType = "QUERY"
		property = "query"

		statDetails := jobStatistics.Details.(*bigquery.QueryStatistics)
		totalBytesBilled = statDetails.TotalBytesBilled
		totalSlotMs = statDetails.SlotMillis
		billingTier = statDetails.BillingTier
		cacheHit = statDetails.CacheHit

		referencedTables = statDetails.ReferencedTables

		if js, err := json.Marshal(statDetails.Timeline); err == nil {
			timeline = string(js)
		}

		for _, qp := range statDetails.QueryPlan {
			if js, err := json.Marshal(qp); err == nil {
				queryPlan = append(queryPlan, string2NullString(string(js)))
			}
		}

	case *bigquery.LoadConfig:
		jobType = "LOAD"
		property = "load"
	case *bigquery.ExtractConfig:
		jobType = "EXTRACT"
		property = "extract"
	case *bigquery.CopyConfig:
		jobType = "COPY"
		property = "copy"
	}

	ts := time.Now().UTC()

	referencedTablesData := make([]ReferencedTable, 0, len(referencedTables))
	for i, table := range referencedTables {
		referencedTablesData[i] = ReferencedTable{
			ProjectID: string2NullString(table.ProjectID),
			DatasetID: string2NullString(table.DatasetID),
			TableID:   string2NullString(table.TableID),
		}
	}

	reservationUsageData := make([]ReservationUsage, 0, len(jobStatistics.ReservationUsage))
	for i, ru := range jobStatistics.ReservationUsage {
		reservationUsageData[i] = ReservationUsage{
			Name:   string2NullString(ru.Name),
			SlotMs: int2NullInt64(ru.SlotMillis),
		}
	}

	if js, err := json.Marshal(jobStatistics); err == nil {
		statistics = string(js)
	}

	if js, err := json.Marshal(jobConfiguration); err == nil {
		configuration = string(js)
	}

	if js, err := json.Marshal(jobStatus); err == nil {
		status = string(js)
	}

	// ErrorResult
	errorResult := ErrorResult{}

	for _, err := range jobStatus.Errors {
		errorResult = ErrorResult{
			Reason:   string2NullString(err.Reason),
			Message:  string2NullString(err.Message),
			Location: string2NullString(err.Location),
		}
	}

	row := HistoricalJobsRow{
		Ts:                       time2NullTimestamp(ts),
		JobID:                    string2NullString(job.ID()),
		ProjectID:                string2NullString(job.ProjectID()),
		Location:                 string2NullString(job.Location()),
		UserEmail:                string2NullString(job.Email()),
		State:                    string2NullString(jobState2String(jobStatus.State)),
		CreationTime:             time2NullTimestamp(jobStatistics.CreationTime),
		EndTime:                  time2NullTimestamp(jobStatistics.EndTime),
		StartTime:                time2NullTimestamp(jobStatistics.StartTime),
		Query:                    string2NullString(query),
		TotalBytesProcessed:      bigquery.NullInt64{Int64: jobStatistics.TotalBytesProcessed, Valid: true},
		TotalBytesBilled:         int2NullInt64(totalBytesBilled),
		TotalSlotMs:              int2NullInt64(totalSlotMs),
		Property:                 bigquery.NullString{StringVal: property, Valid: true},
		JobType:                  bigquery.NullString{StringVal: jobType, Valid: true},
		BillingTier:              int2NullInt64(billingTier),
		CacheHit:                 bool2NullBool(cacheHit),
		ReferencedTables:         referencedTablesData,
		TotalPartitionsProcessed: bigquery.NullInt64{Valid: false}, // missing in the jobStatistics
		QueryPlan:                queryPlan,
		Timeline:                 string2NullString(timeline),
		ReservationUsage:         reservationUsageData,
		ReservationID:            bigquery.NullString{Valid: false}, // missing
		Statistics:               string2NullString(statistics),
		Configuration:            string2NullString(configuration),
		Status:                   string2NullString(status),
		ErrorResult:              errorResult,
	}

	return &row, nil
}

func jobState2String(state bigquery.State) string {
	var st string

	switch state {
	case bigquery.StateUnspecified:
		st = ""
	case bigquery.Pending:
		st = "pending"
	case bigquery.Running:
		st = "running"
	case bigquery.Done:
		st = "done"
	}

	return st
}

func string2NullString(s string) bigquery.NullString {
	if s == "" {
		return bigquery.NullString{
			Valid: false,
		}
	}

	return bigquery.NullString{
		StringVal: s,
		Valid:     true,
	}
}

func bool2NullBool(b bool) bigquery.NullBool {
	return bigquery.NullBool{
		Bool:  b,
		Valid: true,
	}
}

func time2NullTimestamp(t time.Time) bigquery.NullTimestamp {
	if t.IsZero() {
		return bigquery.NullTimestamp{
			Valid: false,
		}
	}

	return bigquery.NullTimestamp{
		Timestamp: t,
		Valid:     true,
	}
}

func int2NullInt64(i int64) bigquery.NullInt64 {
	if i == 0 {
		return bigquery.NullInt64{
			Valid: false,
		}
	}

	return bigquery.NullInt64{
		Int64: i,
		Valid: true,
	}
}

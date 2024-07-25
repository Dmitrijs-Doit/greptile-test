package domain

import (
	"cloud.google.com/go/bigquery"
)

const (
	DoitCmpDatasetID           = "doitintl_cmp_bq"
	DoitCmpTablesTable         = "tables_discovery"
	DoitCmpHistoricalJobsTable = "historicalJobs"
)

var (
	HistoricalJobsSchema = bigquery.Schema{
		{Name: "ts", Type: bigquery.TimestampFieldType, Required: false},
		{Name: "jobId", Type: bigquery.StringFieldType, Required: false},
		{Name: "projectId", Type: bigquery.StringFieldType, Required: false},
		{Name: "location", Type: bigquery.StringFieldType, Required: false},
		{Name: "user_email", Type: bigquery.StringFieldType, Required: false},
		{Name: "state", Type: bigquery.StringFieldType, Required: false},
		{Name: "creationTime", Type: bigquery.TimestampFieldType, Required: false},
		{Name: "endTime", Type: bigquery.TimestampFieldType, Required: false},
		{Name: "startTime", Type: bigquery.TimestampFieldType, Required: false},
		{Name: "query", Type: bigquery.StringFieldType, Required: false},
		{Name: "totalBytesProcessed", Type: bigquery.IntegerFieldType, Required: false},
		{Name: "totalBytesBilled", Type: bigquery.IntegerFieldType, Required: false},
		{Name: "totalSlotMs", Type: bigquery.IntegerFieldType, Required: false},
		{Name: "property", Type: bigquery.StringFieldType, Required: false},
		{Name: "jobType", Type: bigquery.StringFieldType, Required: false},
		{Name: "billingTier", Type: bigquery.IntegerFieldType, Required: false},
		{Name: "cacheHit", Type: bigquery.BooleanFieldType, Required: false},
		{Name: "referencedTables", Type: bigquery.RecordFieldType, Repeated: true, Required: false,
			Schema: bigquery.Schema{
				{Name: "projectId", Type: bigquery.StringFieldType, Required: false},
				{Name: "datasetId", Type: bigquery.StringFieldType, Required: false},
				{Name: "tableId", Type: bigquery.StringFieldType, Required: false},
			},
		},
		{Name: "totalPartitionsProcessed", Type: bigquery.IntegerFieldType, Required: false},
		{Name: "queryPlan", Type: bigquery.StringFieldType, Repeated: true, Required: false},
		{Name: "timeline", Type: bigquery.StringFieldType, Repeated: true, Required: false},
		{Name: "reservationUsage", Type: bigquery.RecordFieldType, Repeated: true, Required: false,
			Schema: bigquery.Schema{
				{Name: "name", Type: bigquery.StringFieldType, Required: false},
				{Name: "slotMs", Type: bigquery.IntegerFieldType, Required: false},
			},
		},
		{Name: "reservation_id", Type: bigquery.StringFieldType, Required: false},
		{Name: "statistics", Type: bigquery.StringFieldType, Required: false},
		{Name: "configuration", Type: bigquery.StringFieldType, Required: false},
		{Name: "status", Type: bigquery.StringFieldType, Required: false},
		{Name: "errorResult", Type: bigquery.RecordFieldType, Required: false,
			Schema: bigquery.Schema{
				{Name: "reason", Type: bigquery.StringFieldType, Required: false},
				{Name: "message", Type: bigquery.StringFieldType, Required: false},
				{Name: "location", Type: bigquery.StringFieldType, Required: false},
				{Name: "debugInfo", Type: bigquery.StringFieldType, Required: false},
			},
		},
	}

	HistoricalJobsClustering = []string{"projectId", "user_email"}

	HistoricalJobsPartitioning = &bigquery.TimePartitioning{
		Field: "startTime",
		Type:  "DAY",
	}

	DoitCmpHistoricalJobsTableMetadata = &bigquery.TableMetadata{
		Schema:           HistoricalJobsSchema,
		TimePartitioning: HistoricalJobsPartitioning,
		Clustering:       &bigquery.Clustering{Fields: HistoricalJobsClustering},
	}
)

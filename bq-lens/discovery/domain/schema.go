package domain

import (
	"cloud.google.com/go/bigquery"
)

const (
	ProjectIDColumn           = 1
	DatasetIDColumn           = 2
	StorageBillingModelColumn = 30
)

var TablesTableClustering = []string{"project_id", "dataset_id", "table_id"}

var TablesSchema = bigquery.Schema{
	{Name: "ts", Type: bigquery.TimestampFieldType},
	{Name: "project_id", Type: bigquery.StringFieldType},
	{Name: "dataset_id", Type: bigquery.StringFieldType},
	{Name: "table_id", Type: bigquery.StringFieldType},
	{Name: "table_base_name", Type: bigquery.StringFieldType},
	{Name: "creation_time", Type: bigquery.TimestampFieldType},
	{Name: "labels", Type: bigquery.RecordFieldType, Repeated: true,
		Schema: bigquery.Schema{
			{Name: "key", Type: bigquery.StringFieldType},
			{Name: "value", Type: bigquery.StringFieldType},
		},
	},
	{Name: "ddl", Type: bigquery.StringFieldType},
	{Name: "partition_info", Type: bigquery.StringFieldType},
	{Name: "clustering", Type: bigquery.StringFieldType},
	{Name: "type", Type: bigquery.StringFieldType},
	{Name: "location", Type: bigquery.StringFieldType},
	{Name: "is_insertable_into", Type: bigquery.StringFieldType},
	{Name: "is_typed", Type: bigquery.StringFieldType},
	{Name: "base_project_id", Type: bigquery.StringFieldType},
	{Name: "base_dataset_id", Type: bigquery.StringFieldType},
	{Name: "base_table_id", Type: bigquery.StringFieldType},
	{Name: "snapshot_time_ms", Type: bigquery.TimestampFieldType},
	{Name: "default_collation_name", Type: bigquery.StringFieldType},
	{Name: "upsert_stream_apply_watermark", Type: bigquery.TimestampFieldType},
	{Name: "total_rows", Type: bigquery.IntegerFieldType},
	{Name: "total_partitions", Type: bigquery.IntegerFieldType},
	{Name: "total_logical_bytes", Type: bigquery.IntegerFieldType},
	{Name: activeLogicalBytes, Type: bigquery.IntegerFieldType},
	{Name: longTermLogicalBytes, Type: bigquery.IntegerFieldType},
	{Name: "total_physical_bytes", Type: bigquery.IntegerFieldType},
	{Name: activePhysicalBytes, Type: bigquery.IntegerFieldType},
	{Name: longTermPhysicalBytes, Type: bigquery.IntegerFieldType},
	{Name: "time_travel_physical_bytes", Type: bigquery.IntegerFieldType},
	{Name: "storage_last_modified_time", Type: bigquery.TimestampFieldType},
	{Name: "storage_model", Type: bigquery.StringFieldType},
	{Name: "fail_safe_physical_bytes", Type: bigquery.IntegerFieldType},
	{Name: "deleted", Type: bigquery.BooleanFieldType},
	// actual cost based on chosen storage model
	{Name: "cost", Type: bigquery.FloatFieldType},
	// cost if storage model is PHYSICAL
	{Name: "physical_cost", Type: bigquery.FloatFieldType},
	// cost if storage model is LOGICAL
	{Name: "logical_cost", Type: bigquery.FloatFieldType},
}

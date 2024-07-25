package domain

const (
	activeLogicalBytes    = "active_logical_bytes"
	longTermLogicalBytes  = "long_term_logical_bytes"
	activePhysicalBytes   = "active_physical_bytes"
	longTermPhysicalBytes = "long_term_physical_bytes"
)

const udfParsePartitionInfo = `CREATE TEMP FUNCTION parse_partition_info(ddl STRING)
RETURNS STRING
LANGUAGE js AS """
  if (!ddl) {
    return null;
  }
  // Regular expression pattern to match the partition field
  var pattern = /PARTITION BY\\s+([A-Za-z0-9_]+\\(([^)]+)\\)|[A-Za-z0-9_]+)/;

  // Search for the pattern in the DDL
  var match = ddl.match(pattern);

  // If a match is found, return the partition field
  if (match) {
    var partitionField = match[1];

    // Check if a function is applied to the partition field
    var functionMatch = partitionField.match(/([A-Za-z0-9_]+)\\(([^)]+)\\)/);
    if (functionMatch) {
      return functionMatch[2].split(',')[0];
    }

    return partitionField;
  }

  // If no match is found, return null
  return null;
""";
`

const udfParseClusteringInfo = `CREATE TEMP FUNCTION parse_clustering_info(ddl STRING)
RETURNS STRING
LANGUAGE js AS """
  if (!ddl) {
    return null;
  }
  var match = ddl.match(/CLUSTER BY ([^;\\n]+)/);
  if (match) {
    var clustering = match[1].split(',').map(function(s) { return s.trim(); })
    return JSON.stringify(clustering);
  } else {
    return null;
  }
""";
`

const udfParseLabels = `CREATE TEMP FUNCTION parse_labels(ddl STRING)
RETURNS ARRAY<STRUCT<key STRING, value STRING>>
LANGUAGE js AS """
  var labels = [];
  if (!ddl) {
    return labels;
  }
  var match = ddl.match(/labels=\\[(.*?)\\]/s);
  if (match && match[1]) {
    var labelsStr = match[1];
    var labelsPairs = labelsStr ? labelsStr.match(/\\(\"(.*?)\", \"(.*?)\"\\)/g) : null;
    if (labelsPairs) {
      for (var i = 0; i < labelsPairs.length; i++) {
        var pair = labelsPairs[i];
        var keyAndValue = pair.slice(2, -2).split('", "');
        labels.push({key: keyAndValue[0], value: keyAndValue[1]});
      }
    }
  }
  return labels;
""";
`

const udfGetTableIDBaseName = `CREATE TEMP FUNCTION get_table_base_name(tableId STRING)
  RETURNS STRING
  LANGUAGE js AS """
    if (tableId == null || tableId == undefined) {
        return
    }
    // Sharded tables suffix is '_YYYYMMDD'
    let suffix = tableId.substr(tableId.length - 9, tableId.length)
    if (suffix.startsWith("_") && !isNaN(suffix.substr(1, suffix.length)*1)) {
        //If table name ends with 8 digits, then extract them and
        //check if they represent a date in YYYYMMDD format:
        let date = suffix.match(/([0-9]{4})([0-9]{2})([0-9]{2})$/)
        if (date) {
            let possibleDate = new Date(` + "`${date[1]}-${date[2]}-${date[3]}`" + `);
            if (!possibleDate.toString().toLowerCase().startsWith("invalid")) {
                return tableId.substr(0, tableId.length - 9);
            }
        }
    }
    return tableId
""";
`

const SingleProjectQueryTpl = `
SELECT
	table_catalog,
	table_schema,
	table_name,
	table_type,
	is_insertable_into,
	is_typed,
	creation_time,
	base_table_catalog,
	base_table_schema,
	base_table_name,
	snapshot_time_ms,
	ddl,
	default_collation_name,
	upsert_stream_apply_watermark,
	'LOGICAL' AS storage_model,
FROM
` + "`{project}.region-{region}`" + `.INFORMATION_SCHEMA.TABLES
`

const RegionalQueryTpl = udfParsePartitionInfo + udfParseClusteringInfo + udfParseLabels + udfGetTableIDBaseName +
	`WITH
  tables AS (
    {tablesQuery}
  ),
  res AS (
  SELECT
    *,
    '{location}' AS location
  FROM
    tables
  LEFT JOIN (
    SELECT
      * EXCEPT(creation_time,
        table_type),
    FROM
      ` + "`region-{location}`" + `.INFORMATION_SCHEMA.TABLE_STORAGE_BY_ORGANIZATION)
  USING
    ( table_catalog,
      table_schema,
      table_name ))
SELECT
  CURRENT_TIMESTAMP() as ts,
  table_catalog AS project_id,
  table_schema AS dataset_id,
  table_name AS table_id,
  get_table_base_name(table_name) AS table_base_name,
  creation_time,
  parse_labels(ddl) AS labels,
  ddl,
  parse_partition_info(ddl) AS partition_info,
  parse_clustering_info(ddl) AS clustering,
  table_type AS type,
  location,
  is_insertable_into,
  is_typed,
  base_table_catalog AS base_project_id,
  base_table_schema AS base_dataset_id,
  base_table_name AS base_table_id,
  snapshot_time_ms,
  default_collation_name,
  upsert_stream_apply_watermark,
  total_rows,
  total_partitions,
  total_logical_bytes,
  ` + activeLogicalBytes + `,
  ` + longTermLogicalBytes + `,
  total_physical_bytes,
  ` + activePhysicalBytes + `,
  ` + longTermPhysicalBytes + `,
  time_travel_physical_bytes,
  storage_last_modified_time,
  storage_model,
  fail_safe_physical_bytes,
  deleted,
  IF ( storage_model = 'PHYSICAL', {storagePricing} ) / POW(1024, 3) AS cost,  -- Convert bytes to GiB and apply regional pricing
  ({physicalStoragePricing}) / POW(1024, 3) AS physical_cost,  -- Convert bytes to GiB and apply regional pricing
  ({logicalStoragePricing}) / POW(1024, 3) AS logical_cost  -- Convert bytes to GiB and apply regional pricing
  FROM
  res
`

const (
	storagePricingV1 = activePhysicalBytes + " * 0.046 + " + longTermPhysicalBytes + " * 0.023, " + activeLogicalBytes + " * 0.02 + " + longTermLogicalBytes + " * 0.01"
	storagePricingV2 = activePhysicalBytes + " * 0.052 + " + longTermPhysicalBytes + " * 0.026, " + activeLogicalBytes + "* 0.023 + " + longTermLogicalBytes + " * 0.016"
	storagePricingV3 = activePhysicalBytes + " * 0.044 + " + longTermPhysicalBytes + " * 0.022, " + activeLogicalBytes + "* 0.02 + " + longTermLogicalBytes + " * 0.01"
	storagePricingV4 = activePhysicalBytes + " * 0.05 + " + longTermPhysicalBytes + " * 0.025, " + activeLogicalBytes + " * 0.023 + " + longTermLogicalBytes + " * 0.016"
	storagePricingV5 = activePhysicalBytes + " * 0.04 + " + longTermPhysicalBytes + " * 0.02, " + activeLogicalBytes + " * 0.023 + " + longTermLogicalBytes + " * 0.016"
)

var StorageCostPricesPerRegion = map[string]string{
	"asia-east1":              storagePricingV1,
	"asia-southeast1":         storagePricingV1,
	"europe-north1":           storagePricingV1,
	"asia-east2":              storagePricingV2,
	"asia-northeast1":         storagePricingV2,
	"asia-northeast2":         storagePricingV2,
	"asia-northeast3":         storagePricingV2,
	"asia-south1":             storagePricingV2,
	"asia-south2":             storagePricingV2,
	"asia-southeast2":         storagePricingV2,
	"australia-southeast1":    storagePricingV2,
	"australia-southeast2":    storagePricingV2,
	"europe-central2":         storagePricingV2,
	"europe-west2":            storagePricingV2,
	"europe-west3":            storagePricingV2,
	"europe-west8":            storagePricingV2,
	"europe-west9":            storagePricingV2,
	"europe-west1":            storagePricingV3,
	"europe-west4":            storagePricingV3,
	"EU":                      storagePricingV3,
	"northamerica-northeast1": storagePricingV4,
	"northamerica-northeast2": storagePricingV4,
	"us-east4":                storagePricingV4,
	"us-east5":                storagePricingV4,
	"us-west2":                storagePricingV4,
	"us-west3":                storagePricingV4,
	"us-west4":                storagePricingV4,
	"us-central1":             storagePricingV5,
	"us-west1":                storagePricingV5,
	"europe-southwest1":       activePhysicalBytes + " * 0.05 + " + longTermPhysicalBytes + " * 0.025, " + activeLogicalBytes + " * 0.029 + " + longTermLogicalBytes + " * 0.02",
	"europe-west6":            activePhysicalBytes + " * 0.056 + " + longTermPhysicalBytes + " * 0.028, " + activeLogicalBytes + " * 0.025 + " + longTermLogicalBytes + " * 0.017",
	"southamerica-east1":      activePhysicalBytes + " * 0.07 + " + longTermPhysicalBytes + " * 0.035, " + activeLogicalBytes + " * 0.023 + " + longTermLogicalBytes + " * 0.016",
	"southamerica-west1":      activePhysicalBytes + " * 0.06 + " + longTermPhysicalBytes + " * 0.03, " + activeLogicalBytes + " * 0.033 + " + longTermLogicalBytes + " * 0.023",
	"us-east1":                activePhysicalBytes + " * 0.044 + " + longTermPhysicalBytes + " * 0.022, " + activeLogicalBytes + " * 0.023 + " + longTermLogicalBytes + " * 0.016",
	"us-south1":               activePhysicalBytes + " * 0.05 + " + longTermPhysicalBytes + " * 0.025, " + activeLogicalBytes + " * 0.028 + " + longTermLogicalBytes + " * 0.019",
	"US":                      activePhysicalBytes + " * 0.04 + " + longTermPhysicalBytes + " * 0.02, " + activeLogicalBytes + " * 0.02 + " + longTermLogicalBytes + " * 0.01",
}

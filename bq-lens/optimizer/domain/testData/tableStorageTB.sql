WITH
  tables AS (
SELECT
  project_id AS projectId,
  dataset_id AS datasetId,
  CASE WHEN partition_info IS NULL THEN table_base_name ELSE table_id END AS tableId, -- sharded tables have NULL partition_info
  CASE WHEN storage_model="PHYSICAL" THEN active_physical_bytes ELSE active_logical_bytes END AS active_bytes,
  CASE WHEN storage_model="PHYSICAL" THEN long_term_physical_bytes ELSE long_term_logical_bytes END AS long_term_bytes,
  ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
FROM
  `mock-project-id.mock-dataset-id.mock-table-discovery`
WHERE
  DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY))
  AND project_id IS NOT NULL
  AND dataset_id IS NOT NULL
  AND table_id IS NOT NULL ),
tablesDedup AS (
SELECT
  * EXCEPT(_rnk)
FROM
  tables
WHERE
  _rnk = 1 )
SELECT
  projectId,
  datasetId,
  tableId,
  ROUND(SUM((active_bytes+long_term_bytes)/ POW(1024,4)), 4) AS storageTB,
  ROUND(SUM(active_bytes / POW(1024,4)), 4) AS shortTermStorageTB,
  ROUND(SUM(long_term_bytes / POW(1024,4)), 4) AS longTermStorageTB
FROM
  tablesDedup
GROUP BY
  projectId,
  datasetId,
  tableId
ORDER BY
  storageTB DESC
LIMIT
  10

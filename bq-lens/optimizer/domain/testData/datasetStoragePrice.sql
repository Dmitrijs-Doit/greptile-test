WITH
 tables AS (
  SELECT
  project_id AS projectId,
  dataset_id AS datasetId,
  table_id AS tableId,
  CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(active_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(active_logical_bytes,total_logical_bytes),0)*logical_cost END AS shortTermPrice,
  CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(long_term_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(long_term_logical_bytes,total_logical_bytes),0)*logical_cost END AS longTermPrice,
  DATE(ts) AS runDate,
  ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id, DATE(ts) ORDER BY ts DESC) AS _rnk
FROM
  `mock-project-id.mock-dataset-id.mock-table-discovery`
WHERE
  DATE(_PARTITIONTIME) >= '0000-12-02'
  AND DATE(ts) >= '0000-12-02'
  AND project_id IS NOT NULL
  AND dataset_id IS NOT NULL
  AND table_id IS NOT NULL ),
tablesDedup AS (
SELECT
  * EXCEPT(_rnk)
FROM
  tables
WHERE _rnk = 1 ),
dailyData AS (
SELECT
  projectId,
  datasetId,
  runDate,
  ROUND(SUM(shortTermPrice), 4) AS shortTermPrice,
  ROUND(SUM(longTermPrice), 4) AS longTermPrice
FROM
  tablesDedup
GROUP BY
  projectId,
  datasetId,
  runDate )
SELECT
  projectId,
  datasetId,
  AVG(longTermPrice) + AVG(shortTermPrice) AS storagePrice,
  AVG(longTermPrice) AS longTermStoragePrice,
  AVG(shortTermPrice) AS shortTermStoragePrice,
FROM
  dailyData
GROUP BY
  projectId,
  datasetId
ORDER BY
  storagePrice DESC
LIMIT
  10

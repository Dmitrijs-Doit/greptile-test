package bqmodels

const table = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topTables AS (
    SELECT
      projectId,
      datasetId,
      getTableIdBaseName(tableId) as tableId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2, 3
    ORDER BY
      scanTB DESC
    LIMIT
      10 )
  SELECT
    *
  FROM
    topTables
  WHERE
    scanTB > 0
  ORDER BY
    scanTB DESC
`

const tableTopUsers = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topTables AS (
    SELECT
      projectId,
      datasetId,
      tableId,
      CONCAT(projectId, ":", datasetId, ".", getTableIdBaseName(tableId)) as tableFullId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2, 3, 4
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topUsers AS (
    SELECT
      projectId,
      datasetId,
      tableId,
      CONCAT(projectId, ":", datasetId, ".", getTableIdBaseName(tableId)) AS tableFullId,
      user_email,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2, 3, 4, 5),
      topUsersWithRank AS (
      SELECT
        *,
        ROW_NUMBER() OVER(PARTITION BY projectId, datasetId, tableId ORDER BY scanTB DESC) AS _rnk
      FROM
        topUsers
      WHERE
        scanTB>0
        AND tableFullId IN ( SELECT tableFullId FROM topTables) )
    SELECT
      * EXCEPT(_rnk)
    FROM
      topUsersWithRank
    WHERE
      _rnk <= 20
    ORDER BY
      projectId, datasetId, tableId, scanTB DESC
`

const tableTopQueries = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topTables AS (
    SELECT
      projectId,
      datasetId,
      getTableIdBaseName(tableId) AS tableId,
      CONCAT(projectId, ":", datasetId, ".", getTableIdBaseName(tableId)) AS tableFullId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2, 3, 4
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topQueries AS (
    SELECT
      projectId,
      datasetId,
      tableId,
      CONCAT(projectId, ":", datasetId, ".", getTableIdBaseName(tableId)) AS tableFullId,
      user_email AS userId,
      MAX(CONCAT(jobId, '&', location, '&', billingProjectId)) AS jobInfo,
      COUNT(*) AS executedQueries,
      ROUND(AVG(executionTimeMs)/1000, 4) AS avgExecutionTimeSec,
      ROUND(SUM(executionTimeMs)/1000, 4) AS totalExecutionTimeSec,
      ROUND(AVG(totalSlotMs/executionTimeMs), 4) AS avgSlots,
      ROUND(AVG(totalBilledBytes / POW(1024,4)), 4) AS avgScanTB,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS totalScanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      projectId,
      datasetId,
      tableId,
      userId,
      queryHash ),
    topQueriesWithRank AS (
    SELECT
      * EXCEPT(jobInfo),
      SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
      SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
      SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
      ROW_NUMBER() OVER(PARTITION BY projectId, datasetId, tableId ORDER BY totalScanTB DESC) AS _rnk
    FROM
      topQueries
    WHERE
      totalScanTB > 0
      AND tableFullId IN ( SELECT tableFullId FROM topTables) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topQueriesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    totalScanTB DESC
`

const physicalStorage = `
WITH
  costs_per_dataset AS (
  SELECT
    dataset_id AS datasetId,
    project_id AS projectId,
    table_id AS tableId,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk,
    total_logical_bytes,
    total_physical_bytes,
    logical_cost,
    physical_cost,
    storage_model
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY))
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL
    -- Exclude the views, external tables, etc
    AND type = 'BASE TABLE' ),
  latest_costs AS (
  SELECT
    projectId,
    datasetId,
    tableId,
    ROUND(SUM(total_logical_bytes) / POW(1024,3), 6) AS totalLogicalGB,
    ROUND(SUM(total_physical_bytes) / POW(1024,3), 6) AS totalPhysicalGB,
    SUM(logical_cost) AS totalLogicalCost,
    SUM(physical_cost) AS totalPhysicalCost,
  FROM
    costs_per_dataset
  WHERE
    _rnk = 1
    -- Exclude tables already on physical storage
    AND storage_model <> 'PHYSICAL'
  GROUP BY
   1,2,3)
SELECT
  datasetId,
  projectId,
  tableId,
  totalLogicalGB,
  totalPhysicalGB,
  totalLogicalCost,
  totalPhysicalCost,
  SAFE_DIVIDE(c.totalLogicalGB,c.totalPhysicalGB) AS compressionRatio,
  c.totalLogicalCost - c.totalPhysicalCost AS savings
FROM
  latest_costs c
WHERE
  c.totalPhysicalCost < c.totalLogicalCost
ORDER BY
  savings DESC
LIMIT
  20
`

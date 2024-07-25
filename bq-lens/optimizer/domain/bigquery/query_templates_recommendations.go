package bqmodels

const HistoricalJobsUnion = `
  UNION ALL
  -- HISTORICAL JOBS
  SELECT
    user_email,
    ts AS timestamp,
    jobId,
    location,
    projectId AS billingProjectId,
    startTime,
    endTime,
    totalBytesBilled AS totalBilledBytes,
    referencedTables,
    reservationUsage,
    totalSlotMs,
    query,
    SHA256(query) as queryHash
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.historicalJobs`" + `
  WHERE
    jobId IS NOT NULL
    AND totalBytesBilled IS NOT NULL
    AND user_email IS NOT NULL
    AND user_email != ""
    AND DATE(startTime) <= '{endDate}'
    AND DATE(startTime) >= '{startDate}'
    AND (projectId {historicalJobsModePlaceholder} IN {projectsWithReservations}) -- 'NOT' for excluding reservations, '' for reservations only
`

const TablesRecommendations = ` --tablesRecommendations
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
  unnested AS (
  SELECT
    jobsDeduplicated.startTime,
    tRef.projectId,
    tRef.datasetId,
    tRef.tableId
  FROM
    jobsDeduplicated,
    UNNEST(referencedTables) tRef ),
  tablesAccessed AS (
  SELECT
    DISTINCT projectId,
    datasetId,
    tableId,
  FROM
    unnested ),
  allTables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    CASE WHEN partition_info IS NULL THEN table_base_name ELSE table_id END AS tableIdBaseName, -- sharded tables have NULL partition_info
    creation_time AS tableCreateDate,
    ROUND(MAX(total_logical_bytes) / POW(1024,4), 4) AS storageSizeTB,
    MAX(cost) AS cost
  FROM
     ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY))
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL
  GROUP BY
    1,
    2,
    3,
    4,
    5 ),
  tablesAccessedDirectly AS (
  SELECT
    *,
    CASE
      WHEN tablesAccessed.tableId IS NOT NULL THEN TRUE
    ELSE
    FALSE
  END
    AS accessed
  FROM
    tablesAccessed
  FULL JOIN
    allTables
  USING
    (projectId,
      datasetId,
      tableId) ),
  wildCardTables AS (
    SELECT
      * REPLACE(REPLACE(tableId, "*", "%") AS tableId)
    FROM
      tablesAccessed
    WHERE
      tableId LIKE '%*' ),
  tablesAccessedWithWildCard AS (
    SELECT
      allt.projectId,
      allt.datasetId,
      allt.tableId,
      CASE
        WHEN allt.tableId IS NOT NULL THEN TRUE
      ELSE
      FALSE
    END
      AS accessedUsingWildCard
    FROM
      allTables allt
    RIGHT JOIN
      wildCardTables ta
    ON
      ta.projectId=allt.projectId
      AND ta.datasetId=allt.datasetId
      AND allt.tableId LIKE ta.tableId )
SELECT
  * EXCEPT(accessed, accessedUsingWildCard), (SELECT sum(cost) FROM allTables) as totalStorageCost
FROM
  tablesAccessedDirectly
  LEFT JOIN tablesAccessedWithWildCard USING (projectId,datasetId,tableId)
WHERE
  -- Recommend to remove a table if it hasn't been accessed in the last 30 days and the table is older than 30 days
  accessed IS FALSE AND accessedUsingWildCard IS NOT TRUE
  AND DATE(tableCreateDate) <= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 60 DAY))
ORDER BY
  storageSizeTB DESC
LIMIT
  50`

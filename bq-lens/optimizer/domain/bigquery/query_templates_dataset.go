package bqmodels

const dataset = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topDatasets AS (
    SELECT
      projectId,
      datasetId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2
    ORDER BY
      scanTB DESC
    LIMIT
      10 )
  SELECT
    *
  FROM
    topDatasets
  WHERE
    scanTB > 0
  ORDER BY
    scanTB DESC
`

const datasetTopUsers = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topDatasets AS (
    SELECT
      projectId,
      datasetId,
      CONCAT(projectId, ":", datasetId) as datasetFullId,
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
      10 ),
    topUsers AS (
    SELECT
      projectId,
      datasetId,
      CONCAT(projectId, ":", datasetId) as datasetFullId,
      user_email,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2, 3, 4),
    topUsersWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY projectId, datasetId ORDER BY scanTB DESC) AS _rnk
    FROM
      topUsers
    WHERE
      scanTB>0
      AND datasetFullId IN ( SELECT datasetFullId FROM topDatasets) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topUsersWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    projectId,
    user_email,
    scanTB DESC
`

const datasetTopTables = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topDatasets AS (
    SELECT
      projectId,
      datasetId,
      CONCAT(projectId, ":", datasetId) AS datasetFullId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1,
      2,
      3
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topTables AS (
    SELECT
      projectId,
      datasetId,
      CONCAT(projectId, ":", datasetId) AS datasetFullId,
      getTableIdBaseName(tableId) AS tableId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2, 3, 4 ),
    topTablesWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY projectId, datasetId ORDER BY scanTB DESC) AS _rnk
    FROM
      topTables
    WHERE
      scanTB > 0
      AND datasetFullId IN (
      SELECT
        datasetFullId
      FROM
        topDatasets) )
  SELECT
    datasetFullId,
    datasetId,
    tableId,
    scanTB
  FROM
    topTablesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    datasetId,
    tableId,
    scanTB DESC
`

const datasetTopQueries = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topDatasets AS (
    SELECT
      projectId,
      datasetId,
      CONCAT(projectId, ":", datasetId) AS datasetFullId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1,
      2,
      3
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topQueries AS (
    SELECT
      projectId,
      datasetId,
      CONCAT(projectId, ":", datasetId) AS datasetFullId,
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
      userId,
      queryHash ),
    topQueriesWithRank AS (
    SELECT
      * EXCEPT(jobInfo),
      SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
      SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
      SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
      ROW_NUMBER() OVER(PARTITION BY projectId, datasetId ORDER BY totalScanTB DESC) AS _rnk
    FROM
      topQueries
    WHERE
      totalScanTB > 0
      AND datasetFullId IN ( SELECT datasetFullId FROM topDatasets) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topQueriesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    totalScanTB DESC
`

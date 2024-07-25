package bqmodels

const project = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topProjects AS (
    SELECT
      projectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      2 DESC,
      1 DESC
    LIMIT
      10 )
  SELECT
    *
  FROM
    topProjects
  WHERE
    scanTB > 0
  ORDER BY
    scanTB DESC
`

const projectTopUsers = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topProjects AS (
    SELECT
      projectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      2 DESC,
      1 DESC
    LIMIT
      10 ),
    topUsers AS (
    SELECT
      projectId,
      user_email,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1,
      2 ),
    topUsersWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY projectId ORDER BY scanTB DESC) AS _rnk
    FROM
      topUsers
    WHERE
      scanTB>0
      AND projectId IN ( SELECT projectId FROM topProjects) )
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

const projectTopQueries = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topProjects AS (
    SELECT
      projectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      2 DESC,
      1 DESC
    LIMIT
      10 ),
    topQueries AS (
    SELECT
      projectId,
      user_email as userId,
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
      userId,
      queryHash ),
    topQueriesWithRank AS (
    SELECT
      * EXCEPT(jobInfo),
      SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
      SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
      SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
      ROW_NUMBER() OVER(PARTITION BY projectId ORDER BY totalScanTB DESC) AS _rnk
    FROM
      topQueries
    WHERE
      totalScanTB > 0
      AND projectId IN ( SELECT projectId FROM topProjects) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topQueriesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    totalScanTB DESC
`

const projectTopDatasets = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topProjects AS (
    SELECT
      projectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      2 DESC,
      1 DESC
    LIMIT
      10 ),
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
      1,
      2 ),
    topDatasetsWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY projectId ORDER BY scanTB DESC) AS _rnk
    FROM
      topDatasets
    WHERE
      scanTB > 0
      AND projectId IN ( SELECT projectId FROM topProjects) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topDatasetsWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    projectId,
    datasetId,
    scanTB DESC
`

const projectTopTables = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
  topProjects AS (
  SELECT
    projectId,
    ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
  FROM
    scanAttribution
  WHERE
    totalBilledBytes > 0
  GROUP BY
    1
  ORDER BY
    2 DESC,
    1 DESC
  LIMIT
    10 ),
  topTables AS (
  SELECT
    projectId,
    CONCAT(datasetId, ".", getTableIdBaseName(tableId)) as tableId,
    ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
  FROM
    scanAttribution
  WHERE
    totalBilledBytes > 0
  GROUP BY
    1,
    2 ),
  topTablesWithRank AS (
  SELECT
    *,
    ROW_NUMBER() OVER(PARTITION BY projectId ORDER BY scanTB DESC) AS _rnk
  FROM
    topTables
  WHERE
    scanTB > 0
    AND projectId IN ( SELECT projectId FROM topProjects) )
SELECT
  * EXCEPT(_rnk)
FROM
  topTablesWithRank
WHERE
  _rnk <= 20
ORDER BY
  projectId,
  tableId,
  scanTB DESC
`

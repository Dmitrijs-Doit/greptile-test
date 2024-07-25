package bqmodels

const user = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topUsers AS (
    SELECT
      user_email AS userId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      scanTB DESC
    LIMIT
      10 )
SELECT
    *
FROM
    topUsers
WHERE
  scanTB > 0
ORDER BY
  scanTB DESC
`

const userTopProjects = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topUsers AS (
    SELECT
      user_email AS userId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topProjects AS (
    SELECT
      user_email AS userId,
      projectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2 ),
    topProjectsWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY userId ORDER BY scanTB DESC) AS _rnk
    FROM
      topProjects
    WHERE
      scanTB>0
      AND userId IN ( SELECT userId FROM topUsers) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topProjectsWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    scanTB DESC
`

const userTopDatasets = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topUsers AS (
    SELECT
      user_email AS userId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topDatasets AS (
    SELECT
      user_email AS userId,
      CONCAT(projectId, ":", datasetId) as datasetId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2 ),
    topDatasetsWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY userId ORDER BY scanTB DESC) AS _rnk
    FROM
      topDatasets
    WHERE
      scanTB>0
      AND userId IN (SELECT userId FROM topUsers) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topDatasetsWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    scanTB DESC
`

const userTopTables = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
    topUsers AS (
    SELECT
      user_email AS userId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topTables AS (
    SELECT
      user_email AS userId,
      CONCAT(projectId, ":", datasetId, ".", getTableIdBaseName(tableId) ) as tableId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2 ),
    topTablesWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY userId ORDER BY scanTB DESC) AS _rnk
    FROM
      topTables
    WHERE
      scanTB>0
      AND userId IN (SELECT userId FROM topUsers) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topTablesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    scanTB DESC
`

const userTopQueries = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topUsers AS (
    SELECT
      user_email AS userId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      scanAttribution
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topQueries AS (
    SELECT
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
      userId,
      queryHash ),
    topQueriesWithRank AS (
    SELECT
      * EXCEPT(jobInfo),
      SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
      SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
      SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
      ROW_NUMBER() OVER(PARTITION BY userId ORDER BY totalScanTB DESC) AS _rnk
    FROM
      topQueries
    WHERE
      totalScanTB>0
      AND userId IN (SELECT userId FROM topUsers) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topQueriesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    totalScanTB DESC
`

package bqmodels

const billingProject = `
WITH
{jobsDeduplicatedWithClause}
topBillingProjects AS (
    SELECT
      billingProjectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      jobsDeduplicated
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
    topBillingProjects
  WHERE
    scanTB > 0
  ORDER BY
    scanTB DESC
`

const billingProjectTopUsers = `
WITH
{jobsDeduplicatedWithClause}
topBillingProjects AS (
    SELECT
      billingProjectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      jobsDeduplicated
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      scanTB DESC
    LIMIT
      10 ),
    topUsers AS (
    SELECT
      billingProjectId,
      user_email,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      jobsDeduplicated
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1, 2 ),
    topUsersWithRank AS (
    SELECT
      *,
      ROW_NUMBER() OVER(PARTITION BY billingProjectId ORDER BY scanTB DESC) AS _rnk
    FROM
      topUsers
    WHERE
      scanTB>0
      AND billingProjectId IN ( SELECT billingProjectId FROM topBillingProjects) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topUsersWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    scanTB DESC
`

const billingProjectTopQueries = `
WITH
{jobsDeduplicatedWithClause}
topBillingProjects AS (
    SELECT
      billingProjectId,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB
    FROM
      jobsDeduplicated
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
      billingProjectId,
      user_email AS userId,
      MAX(CONCAT(jobId, '&', location)) AS jobInfo,
      COUNT(*) AS executedQueries,
      ROUND(AVG(executionTimeMs)/1000, 4) AS avgExecutionTimeSec,
      ROUND(SUM(executionTimeMs)/1000, 4) AS totalExecutionTimeSec,
      ROUND(AVG(totalSlotMs/executionTimeMs), 4) AS avgSlots,
      ROUND(AVG(totalBilledBytes / POW(1024,4)), 4) AS avgScanTB,
      ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS totalScanTB
    FROM
      jobsDeduplicated
    WHERE
      totalBilledBytes > 0
    GROUP BY
      billingProjectId,
      userId,
      queryHash ),
    topQueriesWithRank AS (
    SELECT
      * EXCEPT(jobInfo),
      SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
      SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
      ROW_NUMBER() OVER(PARTITION BY billingProjectId ORDER BY totalScanTB DESC) AS _rnk
    FROM
      topQueries
    WHERE
      totalScanTB > 0
      AND billingProjectId IN ( SELECT billingProjectId FROM topBillingProjects) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topQueriesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    totalScanTB DESC
`

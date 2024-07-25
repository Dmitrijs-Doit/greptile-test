
WITH

    src AS (
        SELECT
            protopayload_auditlog.authenticationInfo.principalEmail AS user_email,
    timestamp,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.location,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId AS billingProjectId,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.endTime,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalBilledBytes,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.referencedTables,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.reservationUsage,
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalSlotMs,
    NULL AS query,
    SHA256(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobConfiguration.query.query) as queryHash
FROM
    `mock-project-id.mock-dataset-id.cloudaudit_googleapis_com_data_access`
WHERE
    protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId IS NOT NULL
  AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId NOT LIKE 'script_job_%' -- filter BQ script child jobs
  AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.eventName = 'query_job_completed'
  AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalBilledBytes IS NOT NULL
  AND (protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId NOT IN ("project1","project2","project3")) -- 'NOT' for excluding reservtions, '' for reservations only
  AND protopayload_auditlog.authenticationInfo.principalEmail IS NOT NULL
  AND protopayload_auditlog.authenticationInfo.principalEmail != ""
  AND DATE(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime) >= '2021-12-02'
  AND DATE(timestamp) >= '2021-12-02'
    ),
    jobs AS (
SELECT
    *,
    TIMESTAMP_DIFF(endTime, startTime, MILLISECOND) as executionTimeMs,
    ROW_NUMBER() OVER(PARTITION BY jobId ORDER BY timestamp DESC) AS _rnk
FROM
    src ),
    jobsDeduplicated AS (
SELECT
    * EXCEPT(_rnk)
FROM
    jobs
WHERE
    _rnk = 1 ),

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

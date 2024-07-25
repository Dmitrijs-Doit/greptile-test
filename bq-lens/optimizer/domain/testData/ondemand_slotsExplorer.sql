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
  AND DATE(timestamp) >= '2021-12-02' ),
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
    allocatedSlotsEvents as (
SELECT
    startTime,
    endTime,
    totalSlotMs / executionTimeMs as slots,
    GENERATE_TIMESTAMP_ARRAY(startTime, endTime, INTERVAL 1 SECOND) AS tArr
FROM
    jobsDeduplicated ),
    unnested AS (
SELECT
    *
FROM
    allocatedSlotsEvents,
    UNNEST(tArr) t ),
    truncated AS (
SELECT
    startTime,
    endTime,
    TIMESTAMP_TRUNC(t, SECOND) as t,
    slots
FROM
    unnested ),
    groupedBySecond AS (
SELECT
    t,
    SUM(slots) AS slotsPerSecond
FROM
    truncated
GROUP BY
    1 ),
    peakSlotsPerDayAndHour AS (
SELECT
    DATE(t) AS date,
    EXTRACT( HOUR
    FROM
    t ) AS hour,
    MAX(slotsPerSecond) AS maxSlots
FROM
    groupedBySecond
GROUP BY
    1,
    2 ),
    aggregationByDayAndHourAndSecond AS (
SELECT
    DATE(startTime) AS date,
    EXTRACT( HOUR
    FROM
    startTime ) AS hour,
    SUM(totalSlotMs) /(60 * 60 * 1000) AS avgSlots
FROM
    jobsDeduplicated
GROUP BY
    1,
    2 )
SELECT
    a.date AS day,
  a.hour,
  ROUND(a.avgSlots, 2) AS avgSlots,
  ROUND(p.maxSlots, 2) AS maxSlots
FROM
    aggregationByDayAndHourAndSecond a
    LEFT JOIN
    peakSlotsPerDayAndHour p
ON
    a.hour = p.hour
    AND a.date = p.date
ORDER BY
    1,
    2
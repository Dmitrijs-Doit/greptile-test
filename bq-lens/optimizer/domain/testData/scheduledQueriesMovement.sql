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
          AND (protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId IN ("project1","project2","project3")) -- 'NOT' for excluding reservtions, '' for reservations only
          AND protopayload_auditlog.authenticationInfo.principalEmail IS NOT NULL
          AND protopayload_auditlog.authenticationInfo.principalEmail != ""
          AND DATE(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime) >= '0000-12-02'
          AND DATE(timestamp) >= '0000-12-02'
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

slotsByDateAndHour as (
  SELECT
    DATE(startTime) as date,
    EXTRACT(HOUR FROM startTime) AS hour,
    SUM(totalSlotMs) /(60 * 60 * 1000) AS slots,
  FROM
    jobsDeduplicated
  GROUP BY
    1,
    2
),
slotsByHour AS (
  SELECT
    hour,
    ROUND(AVG(slots), 4) AS hourlyAvgSlots
  FROM
    slotsByDateAndHour
  GROUP BY
    1
),
slotsByHourWithThreshold AS (
  SELECT
    *,
    (SELECT AVG(hourlyAvgSlots) FROM slotsByHour) as hourlySlotsThreshold
  FROM
    slotsByHour
),
-- check frequent queries to be able to find scheduled queries among them
frequentQueries as (
  SELECT
    MAX(CONCAT(jobId, '&', location, '&', billingProjectId)) AS jobInfo,
    queryHash,
    FORMAT_TIMESTAMP("%R", startTime) AS scheduledTime,
    ROUND(EXTRACT(HOUR FROM startTime),2) AS scheduledHour,
    count(*) as totalExecutedQueries,
    COUNT(DISTINCT DATE(startTime)) as days,
    SUM(totalSlotMs) / SUM(executionTimeMs) AS slots,
    SUM(totalSlotMs) AS sumTotalSlotMs
  FROM
    jobsDeduplicated
  group by
    2,
    3,
    4
),
-- fetch repeating queries only (repeating every day at the same HH:MM)
scheduledQueries AS (
  SELECT
    SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
    SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
    SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
    *
  EXCEPT(queryHash, days),
  FROM
    frequentQueries
  WHERE
    days >= DATE_DIFF(CURRENT_DATE(), '0000-12-02', DAY)
),
-- only analyze top 100 queries in "peak hours"
relevantQueries AS (
SELECT
  *
FROM
  scheduledQueries l
LEFT JOIN slotsByHourWithThreshold r on l.scheduledHour = r.hour
WHERE
  hourlyAvgSlots > hourlySlotsThreshold
ORDER BY
  slots DESC
LIMIT 100
),
-- compute slots used by repeating queries per scheduled hour
slotsUsedByRepeatingQueries AS (
SELECT
  scheduledHour,
  hourlyAvgSlots,
  hourlySlotsThreshold,
  hourlyAvgSlots - hourlySlotsThreshold AS excessSlots,
  SUM(sumTotalSlotMs) / (60*60*1000) AS slotsUsed,
  (2000*hourlySlotsThreshold/100)/24 AS pricePerHour,
FROM
  relevantQueries
GROUP BY 1,2,3
),
-- compute savings
savingsPerHour AS (
SELECT
  SUM(
    CASE WHEN slotsUsed <= excessSlots THEN (slotsUsed/hourlySlotsThreshold)*pricePerHour
    ELSE (excessSlots/hourlySlotsThreshold)*pricePerHour END
  ) AS saving
FROM
  slotsUsedByRepeatingQueries
)
SELECT
  jobId,
  location,
  billingProjectId,
  scheduledTime,
  totalExecutedQueries as allJobs,
  slots,
  (SELECT saving FROM savingsPerHour) as savingsPrice
FROM
  relevantQueries
ORDER BY
  slots DESC
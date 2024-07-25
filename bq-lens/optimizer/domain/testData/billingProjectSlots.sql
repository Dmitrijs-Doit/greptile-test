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

topBillingProjects AS (
    SELECT
        billingProjectId,
        ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '0000-12-02', MILLISECOND)), 4) AS slots
    FROM
        jobsDeduplicated
    WHERE
        totalBilledBytes > 0
    GROUP BY
        1
    ORDER BY
        slots DESC
    LIMIT
        10
)
SELECT
    *
FROM
    topBillingProjects
WHERE
    slots > 0
ORDER BY
    slots DESC
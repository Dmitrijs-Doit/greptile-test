package bqmodels

const AggregatedJobStatisticsQuery = `
SELECT
      protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.location AS location,
      protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId AS projectId,
      SUM(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalSlotMs) AS totalSlotMs,
      SUM(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalBilledBytes) AS totalBilledBytes,
    FROM
        {projectIdPlaceHolder}.{datasetIdPlaceHolder}.cloudaudit_googleapis_com_data_access
    WHERE
      protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId IS NOT NULL
      AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId NOT LIKE 'script_job_%' -- filter BQ script child jobs
      AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.eventName = 'query_job_completed'
      AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalBilledBytes IS NOT NULL
      AND protopayload_auditlog.authenticationInfo.principalEmail IS NOT NULL
      AND protopayload_auditlog.authenticationInfo.principalEmail != ""
      AND DATE(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime) >= DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY)
      AND DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 90 DAY)
	  AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.reservation  = "unreserved"
    GROUP BY location, projectId
`

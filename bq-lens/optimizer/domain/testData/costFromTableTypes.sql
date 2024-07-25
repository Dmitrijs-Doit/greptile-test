-- costFromTableTypes
CREATE TEMP FUNCTION getTableIdBaseName(tableId STRING)
  RETURNS STRING
  LANGUAGE js AS """
  function getTableIdBaseName(tableId) {
    if (tableId == null || tableId == undefined) {
        return
    }
    // Sharded tables suffix is '_YYYYMMDD'
    let suffix = tableId.substr(tableId.length - 9, tableId.length)
    if (suffix.startsWith("_") && !isNaN(suffix.substr(1, suffix.length)*1)) {
        //If table name ends with 8 digits, then extract them and
        //check if they represent a date in YYYYMMDD format:
        let date = suffix.match(/([0-9]{4})([0-9]{2})([0-9]{2})$/)
        if (date) {
            let possibleDate = new Date(`${date[1]}-${date[2]}-${date[3]}`);
            if (!possibleDate.toString().toLowerCase().startsWith("invalid")) {
                return tableId.substr(0, tableId.length - 9);
            }
        }
    }
    return tableId
};
return getTableIdBaseName(tableId);
""";

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
          AND (protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId IS NOT NULL OR protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId IN ("project1","project2","project3")) -- 'NOT' for excluding reservtions, '' for reservations only
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

    unnested AS (
        SELECT
            *
        FROM
            jobsDeduplicated,
            UNNEST(referencedTables) tRef ),
    user_tables AS (
        SELECT
            project_id AS projectId,
            dataset_id AS datasetId,
            table_id AS tableId,
            CASE WHEN partition_info IS NULL THEN table_base_name ELSE table_id END AS tableIdBaseName, -- sharded tables have NULL partition_info
            total_logical_bytes AS bytesCount,
            clustering,
            partition_info AS partitionInfo,
            type AS tableType,
            ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
        FROM
            `mock-project-id.mock-dataset-id.mock-table-discovery`
        WHERE
            DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY)) ),
    table_sizes AS (
        SELECT
            projectId,
            datasetId,
            tableId,
            --- GET TABLE TYPE
            CASE
                WHEN clustering IS NOT NULL THEN "clustered"
                WHEN partitionInfo IN ('_PARTITIONTIME','_PARTITIONDATE') THEN 'ingestionTimePartition'
                WHEN partitionInfo IS NOT NULL THEN "customPartition"
                WHEN tableId != tableIdBaseName THEN "tableRangePartition"
                WHEN tableType IS NULL THEN "unknown"
                WHEN tableType = "EXTERNAL" THEN "external"
                WHEN tableType = "MATERIALIZED_VIEW" THEN "materializedView"
                WHEN tableType = "VIEW" THEN "view"
                WHEN tableType = "SNAPSHOT" THEN "snapshot"
                ELSE
                    "noPartition"
                END
                            AS type,
            ---
            SUM(bytesCount) AS tableSize
        FROM
            user_tables
        WHERE
                _rnk = 1
        GROUP BY
            1, 2, 3, 4 ),
    joined AS (
        SELECT
            u.*,
            IFNULL(t.tableSize, 0) AS tableSize,
            IFNULL(t.type, "unknown") AS tableType
        FROM
            unnested u
                LEFT JOIN
            table_sizes t
            ON
                        u.projectId = t.projectId
                    AND u.datasetId = t.datasetId
                    AND u.tableId = t.tableId ),
    ranked AS (
        SELECT
            * EXCEPT(tableSize),
      ROW_NUMBER() OVER(PARTITION BY jobId ORDER BY tableSize DESC) AS _sizeRnk
    FROM
      joined ),
    scanAttribution AS (
        SELECT
            * EXCEPT(_sizeRnk)
        FROM
      ranked
    WHERE
      _sizeRnk = 1 ),
    groupedPartitions AS (
        SELECT
            CONCAT(projectId, ':', datasetId, '.', tableId) as tableName,
            tableType,
            SUM(totalBilledBytes) / 1024 / 1024 / 1024 / 1024 as totalTB
        FROM
            ranked
        WHERE
                _sizeRnk = 1
        GROUP BY tableName, tableType),
    typesWithRank AS
        (
            SELECT
                *, ROW_NUMBER() OVER(PARTITION BY tableType ORDER BY totalTB DESC) as _rnk_type
            FROM
                groupedPartitions
        ),
    limitedResults AS (
        SELECT
            tableType,
            totalTB,
            CASE
                WHEN _rnk_type <20 then tableName
                ELSE 'others' END
                as tableName
        FROM
            typesWithRank
    )
SELECT
    tableType,
    tableName,
    sum(totalTB) as totalTB
FROM
    limitedResults
GROUP BY
    1,2
ORDER BY tableType desc, totalTB desc
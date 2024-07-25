package bqmodels

//const { pricePerTBScan } = require("./consts");

const CheckCompleteDays = `
SELECT
  MIN(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime) as min,
  MAX(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime) as max
FROM ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.cloudaudit_googleapis_com_data_access`" + `
WHERE DATE(timestamp) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 30 DAY))
LIMIT 2`

const GetTableIdBaseName = `
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
            let possibleDate = new Date(` + "`${date[1]}-${date[2]}-${date[3]}`" + `);
            if (!possibleDate.toString().toLowerCase().startsWith("invalid")) {
                return tableId.substr(0, tableId.length - 9);
            }
        }
    }
    return tableId
};
return getTableIdBaseName(tableId);
""";
`

const JobsDeduplicatedWithClause = `
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
      {queryPlaceholder} AS query,
      SHA256(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobConfiguration.query.query) as queryHash
    FROM
      ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.cloudaudit_googleapis_com_data_access`" + `
    WHERE
      protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId IS NOT NULL
      AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId NOT LIKE 'script_job_%' -- filter BQ script child jobs
      AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.eventName = 'query_job_completed'
      AND protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.totalBilledBytes IS NOT NULL
      AND (protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId {modePlaceholder} IN {projectsWithReservations}) -- 'NOT' for excluding reservtions, '' for reservations only
      AND protopayload_auditlog.authenticationInfo.principalEmail IS NOT NULL
      AND protopayload_auditlog.authenticationInfo.principalEmail != ""
      AND DATE(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.startTime) >= '{startDate}'
      AND DATE(timestamp) >= '{startDate}'
      {historicalJobsPlaceholder}),
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
`

const ScanAttributionWithClause = `
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
      CASE WHEN partition_info IS NULL THEN table_base_name ELSE table_id END AS tableId, -- sharded tables have NULL partition_info
      total_logical_bytes AS bytesCount,
      ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
    FROM
      ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
    WHERE
      DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY)) ),
    table_sizes AS (
    SELECT
      projectId,
      datasetId,
      tableId,
      SUM(bytesCount) AS tableSize
    FROM
      user_tables
    WHERE
      _rnk = 1
    GROUP BY
      1, 2, 3 ),
    joined AS (
    SELECT
      u.*,
      IFNULL(t.tableSize, 0) AS tableSize
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
    REPLACE(CASE WHEN STARTS_WITH(tableId, "__") THEN REPLACE(tableId, "__", "_") ELSE tableId END AS tableId) -- fields starting in "__" are invalid in Firestore
    FROM
      ranked
    WHERE
      _sizeRnk = 1 ),
`

const costFromTableTypes = `-- costFromTableTypes
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
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
      ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
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
ORDER BY tableType desc, totalTB desc`

const TotalScan = `-- totalScan
WITH
{jobsDeduplicatedWithClause}
dailyScan AS (
  SELECT
    DATE(startTime) d,
    ROUND(SUM(totalBilledBytes / POW(1024,4)),4) AS usageTB
  FROM
    jobsDeduplicated
  GROUP BY
    1 )
SELECT
  (
  SELECT
    SUM(usageTB)
  FROM
    dailyScan
  WHERE
    d >= DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY)) AS total_up_to_1_day_ago,
  (
  SELECT
    SUM(usageTB)
  FROM
    dailyScan
  WHERE
    d >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)) AS total_up_to_7_days_ago,
  (
  SELECT
    SUM(usageTB)
  FROM
    dailyScan
  WHERE
    d >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)) AS total_up_to_30_days_ago`

const limitingJobsSavings = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
{scanAttributionWithClause}
topQueries AS (
  SELECT
    CONCAT(projectId, ":", datasetId, ".", tableId) AS tableFullId,
    MAX(CONCAT(jobId, '&', location, '&', billingProjectId)) AS jobInfo,
    MAX(user_email) AS userId,
    MIN(startTime) AS firstExecution,
    MAX(startTime) AS lastExecution,
    ROUND(TIMESTAMP_DIFF(MAX(startTime), MIN(startTime), HOUR)/24) AS timeSpanDays,
    CAST(COUNT(jobId) AS INT64) AS allJobs,
    ROUND(SUM(totalBilledBytes / POW(1024,4)), 4)/COUNT(jobId) AS scanTBperQuery,
    ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS totalScanTB,
    ROUND(${pricePerTBScan}*SUM(totalBilledBytes / POW(1024,4)), 4)/COUNT(jobId) AS scanPricePerQuery,
    ROUND(${pricePerTBScan}*SUM(totalBilledBytes / POW(1024,4)), 4) AS totalScanPrice,
  FROM
    scanAttribution
  WHERE
    totalBilledBytes > 0
  GROUP BY
    projectId,
    datasetId,
    tableId,
    queryHash )
SELECT
  tableFullId,
  SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
  SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
  SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
  userId,
  firstExecution,
  lastExecution,
  allJobs,
  scanPricePerQuery,
  totalScanPrice,
  FLOOR(0.5*allJobs)*scanPricePerQuery AS reducingBy50,
  FLOOR(0.4*allJobs)*scanPricePerQuery AS reducingBy40,
  FLOOR(0.3*allJobs)*scanPricePerQuery AS reducingBy30,
  FLOOR(0.2*allJobs)*scanPricePerQuery AS reducingBy20,
  FLOOR(0.1*allJobs)*scanPricePerQuery AS reducingBy10
FROM
  topQueries
WHERE
  timeSpanDays>0
  AND allJobs/timeSpanDays>0.8
  AND tableFullId IS NOT NULL
ORDER BY
  totalScanTB DESC
LIMIT
  50
`

const usePartitionField = `
CREATE TEMP FUNCTION partitionFieldNotUsed(
  partitionField STRING,
  query STRING,
  tableId STRING
)
RETURNS BOOLEAN LANGUAGE js AS
"""
  let fieldNotUsed=false;
  try {
      if (partitionField && query && tableId) {
          if (query.toLowerCase().indexOf(tableId.toLowerCase()) === -1) {
              fieldNotUsed=false
          } else if (partitionField == "_PARTITIONTIME" || partitionField == "_PARTITIONDATE") {
              // Handle ingestion-time partitioned tables
              if (query.toLowerCase().indexOf("_PARTITIONTIME".toLowerCase()) === -1 && query.toLowerCase().indexOf("_PARTITIONDATE".toLowerCase()) === -1) {
                  fieldNotUsed = true
              }
          } else if (query.toLowerCase().indexOf(partitionField.toLowerCase()) === -1) {
              fieldNotUsed = true
          }
      }
  } catch(e) {
      // Nowhere to go from here.
  }
  return fieldNotUsed
""";
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
unnested AS (
  SELECT
    *
  FROM
    jobsDeduplicated,
    UNNEST(referencedTables) tRef
),
user_tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    partition_info AS partitionInfo,
    ddl,
    ROUND(total_logical_bytes/ POW(1024,2), 4) AS sizeMB,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY)) ),
  table_sizes AS (
  SELECT
    projectId,
    datasetId,
    tableId,
    MAX(ddl) as ddl,
    SUM(sizeMB) AS sizeMB,
    MAX(partitionInfo) AS partitionInfo
  FROM
    user_tables
  WHERE
    _rnk = 1
  GROUP BY
    1,
    2,
    3),
  joined AS (
  SELECT
    u.*,
    IFNULL(t.sizeMB, 0) AS sizeMB,
    partitionInfo,
    ddl
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
    *,
    ROW_NUMBER() OVER(PARTITION BY jobId ORDER BY sizeMB DESC) AS _sizeRnk
  FROM
    joined ),
  scanAttribution AS (
  SELECT
    * EXCEPT(_sizeRnk)
  FROM
    ranked
  WHERE
    _sizeRnk = 1 ),
fields_used AS (
SELECT
  MAX(CONCAT(jobId, '&', location, '&', billingProjectId)) AS jobInfo,
  ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB,
  ROUND(${pricePerTBScan}*SUM(totalBilledBytes / POW(1024,4)), 4) AS scanPrice,
  projectId,
  datasetId,
  tableId,
  partitionInfo AS partitionField,
  ddl,
  partitionFieldNotUsed(partitionInfo, MAX(query), tableId) as partitionFieldNotUsed,
FROM
  scanAttribution
WHERE
  totalBilledBytes > 0
  AND partitionInfo IS NOT NULL
GROUP BY
  projectId,
  datasetId,
  tableId,
  queryHash,
  ddl,
  partitionInfo
)
SELECT
  SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
  SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
  SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
  scanTB,
  scanPrice,
  CONCAT(projectId, ":", datasetId, ".", tableId) AS tableId,
  partitionField,
  ddl
FROM
  fields_used
WHERE
  partitionFieldNotUsed IS TRUE
  AND partitionField IS NOT NULL
  AND ddl IS NOT NULL
ORDER BY
  scanTB DESC
LIMIT
  50
`

const partitionTables = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
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
    -- sharded tables have NULL partitionInfo
    CASE WHEN partition_info IS NULL THEN FALSE ELSE TRUE END AS isPartitioned,
    ROUND(total_logical_bytes/ POW(1024,2), 4) AS sizeMB,
    ddl,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY)) ),
  table_sizes AS (
  SELECT
    projectId,
    datasetId,
    tableId,
    tableIdBaseName,
    SUM(sizeMB) AS sizeMB,
    MAX(isPartitioned) AS isPartitioned,
    MAX(ddl) AS ddl
  FROM
    user_tables
  WHERE
    _rnk = 1
  GROUP BY
    1, 2, 3, 4 ),
  joined AS (
  SELECT
    u.*,
    IFNULL(t.sizeMB, 0) AS sizeMB,
    tableIdBaseName,
    isPartitioned,
    ddl
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
    *,
    ROW_NUMBER() OVER(PARTITION BY jobId ORDER BY sizeMB DESC) AS _sizeRnk
  FROM
    joined ),
  scanAttribution AS (
  SELECT
    * EXCEPT(_sizeRnk)
  FROM
    ranked
  WHERE
    _sizeRnk = 1 )
SELECT
  queryHash,
  MAX(query) AS query,
  MAX(sizeMB) AS sizeMB,
  CONCAT(projectId, ".", datasetId, ".", tableId) AS tableId,
  tableId AS tableName,
  tableIdBaseName,
  ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB,
  ROUND(${pricePerTBScan}*SUM(totalBilledBytes / POW(1024,4)), 4) AS scanPrice,
  MAX(ddl) as ddl
FROM
  scanAttribution
WHERE
  totalBilledBytes > 0
  AND isPartitioned IS FALSE
  AND tableIdBaseName = tableId
GROUP BY
  queryHash,
  tableId,
  tableIdBaseName,
  tableName
ORDER BY
  scanTB DESC
LIMIT
  500
`

const clusterTables = `
{getTableIdBaseName}
WITH
{jobsDeduplicatedWithClause}
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
    -- sharded tables have NULL partitionInfo
    CASE WHEN clustering IS NULL THEN FALSE ELSE TRUE END AS isClustered,
    ROUND(total_logical_bytes/ POW(1024,2), 4) AS sizeMB,
    ddl,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`${projectIdPlaceHolder}.{datasetIdPlaceHolder}.${tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY)) ),
  table_sizes AS (
  SELECT
    projectId,
    datasetId,
    tableId,
    tableIdBaseName,
    SUM(sizeMB) AS sizeMB,
    MAX(isClustered) AS isClustered,
    MAX(ddl) AS ddl
  FROM
    user_tables
  WHERE
    _rnk = 1
  GROUP BY
    1, 2, 3, 4 ),
  joined AS (
  SELECT
    u.*,
    IFNULL(t.sizeMB, 0) AS sizeMB,
    tableIdBaseName,
    isClustered,
    ddl
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
    *,
    ROW_NUMBER() OVER(PARTITION BY jobId ORDER BY sizeMB DESC) AS _sizeRnk
  FROM
    joined ),
  scanAttribution AS (
  SELECT
    * EXCEPT(_sizeRnk)
  FROM
    ranked
  WHERE
    _sizeRnk = 1 )

  SELECT
    queryHash,
    MAX(query) AS query,
    CONCAT(projectId, ".", datasetId, ".", tableId) AS tableId,
    tableId AS tableName,
    tableIdBaseName,
    ROUND(SUM(totalBilledBytes / POW(1024,4)), 4) AS scanTB,
    ROUND(${pricePerTBScan}*SUM(totalBilledBytes / POW(1024,4)), 4) AS scanPrice,
    MAX(sizeMB) as sizeMB,
    MAX(ddl) as ddl
  FROM
    scanAttribution
  WHERE
    totalBilledBytes > 0
    AND isClustered IS FALSE
    AND tableIdBaseName = tableId
  GROUP BY
    queryHash,
    tableId,
    tableIdBaseName,
    tableName
  ORDER BY
    scanTB DESC
  LIMIT
    500
`

const scheduledQueriesMovement = `
WITH
{jobsDeduplicatedWithClause}
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
    days >= DATE_DIFF(CURRENT_DATE(), '{startDate}', DAY)
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
`

const projectStorageTB = `
WITH
  tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    CASE WHEN storage_model="PHYSICAL" THEN active_physical_bytes ELSE active_logical_bytes END AS active_bytes,
    CASE WHEN storage_model="PHYSICAL" THEN long_term_physical_bytes ELSE long_term_logical_bytes END AS long_term_bytes,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY))
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL ),
  tablesDedup AS (
  SELECT
    * EXCEPT(_rnk)
  FROM
    tables
  WHERE
    _rnk = 1 )
SELECT
  projectId,
  ROUND(SUM((active_bytes+long_term_bytes)/ POW(1024,4)), 4) AS storageTB,
  ROUND(SUM(active_bytes / POW(1024,4)), 4) AS shortTermStorageTB,
  ROUND(SUM(long_term_bytes / POW(1024,4)), 4) AS longTermStorageTB
FROM
  tablesDedup
GROUP BY
  projectId
ORDER BY
  storageTB DESC
LIMIT
  10
`

const projectStoragePrice = `
WITH
  tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(active_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(active_logical_bytes,total_logical_bytes),0)*logical_cost END AS shortTermPrice,
    CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(long_term_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(long_term_logical_bytes,total_logical_bytes),0)*logical_cost END AS longTermPrice,
    DATE(ts) AS runDate,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id, DATE(ts) ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= '{startDate}'
    AND DATE(ts) >= '{startDate}'
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL ),
  tablesDedup AS (
  SELECT
    * EXCEPT(_rnk)
  FROM
    tables
  WHERE
    _rnk = 1 ),
  dailyData AS (
  SELECT
    projectId,
    runDate,
    ROUND(SUM(shortTermPrice), 4) AS shortTermPrice,
    ROUND(SUM(longTermPrice), 4) AS longTermPrice
  FROM
    tablesDedup
  GROUP BY
    projectId,
    runDate )
SELECT
  projectId,
  AVG(longTermPrice) + AVG(shortTermPrice) AS storagePrice,
  AVG(longTermPrice) AS longTermStoragePrice,
  AVG(shortTermPrice) AS shortTermStoragePrice,
FROM
  dailyData
GROUP BY
  projectId
ORDER BY
  storagePrice DESC
LIMIT
  10
`

const datasetStorageTB = `
WITH
  tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    CASE WHEN storage_model="PHYSICAL" THEN active_physical_bytes ELSE active_logical_bytes END AS active_bytes,
    CASE WHEN storage_model="PHYSICAL" THEN long_term_physical_bytes ELSE long_term_logical_bytes END AS long_term_bytes,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY))
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL ),
  tablesDedup AS (
  SELECT
    * EXCEPT(_rnk)
  FROM
    tables
  WHERE
    _rnk = 1 )
SELECT
  projectId,
  datasetId,
  ROUND(SUM((active_bytes+long_term_bytes)/ POW(1024,4)), 4) AS storageTB,
  ROUND(SUM(active_bytes / POW(1024,4)), 4) AS shortTermStorageTB,
  ROUND(SUM(long_term_bytes / POW(1024,4)), 4) AS longTermStorageTB
FROM
  tablesDedup
GROUP BY
  projectId,
  datasetId
ORDER BY
  storageTB DESC
LIMIT
  10
`

const datasetStoragePrice = `
WITH
  tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(active_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(active_logical_bytes,total_logical_bytes),0)*logical_cost END AS shortTermPrice,
    CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(long_term_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(long_term_logical_bytes,total_logical_bytes),0)*logical_cost END AS longTermPrice,
    DATE(ts) AS runDate,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id, DATE(ts) ORDER BY ts DESC) AS _rnk
  FROM
   ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= '{startDate}'
    AND DATE(ts) >= '{startDate}'
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL ),
  tablesDedup AS (
  SELECT
    * EXCEPT(_rnk)
  FROM
    tables
  WHERE
    _rnk = 1 ),
  dailyData AS (
  SELECT
    projectId,
    datasetId,
    runDate,
    ROUND(SUM(shortTermPrice), 4) AS shortTermPrice,
    ROUND(SUM(longTermPrice), 4) AS longTermPrice
  FROM
    tablesDedup
  GROUP BY
    projectId,
    datasetId,
    runDate )
SELECT
  projectId,
  datasetId,
  AVG(longTermPrice) + AVG(shortTermPrice) AS storagePrice,
  AVG(longTermPrice) AS longTermStoragePrice,
  AVG(shortTermPrice) AS shortTermStoragePrice,
FROM
  dailyData
GROUP BY
  projectId,
  datasetId
ORDER BY
  storagePrice DESC
LIMIT
  10
`

const tableStorageTB = `
WITH
  tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    CASE WHEN partition_info IS NULL THEN table_base_name ELSE table_id END AS tableId, -- sharded tables have NULL partition_info
    CASE WHEN storage_model="PHYSICAL" THEN active_physical_bytes ELSE active_logical_bytes END AS active_bytes,
    CASE WHEN storage_model="PHYSICAL" THEN long_term_physical_bytes ELSE long_term_logical_bytes END AS long_term_bytes,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 1 DAY))
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL ),
  tablesDedup AS (
  SELECT
    * EXCEPT(_rnk)
  FROM
    tables
  WHERE
    _rnk = 1 )
SELECT
  projectId,
  datasetId,
  tableId,
  ROUND(SUM((active_bytes+long_term_bytes)/ POW(1024,4)), 4) AS storageTB,
  ROUND(SUM(active_bytes / POW(1024,4)), 4) AS shortTermStorageTB,
  ROUND(SUM(long_term_bytes / POW(1024,4)), 4) AS longTermStorageTB
FROM
  tablesDedup
GROUP BY
  projectId,
  datasetId,
  tableId
ORDER BY
  storageTB DESC
LIMIT
  10
`

const tableStoragePrice = `
WITH
  tables AS (
  SELECT
    project_id AS projectId,
    dataset_id AS datasetId,
    table_id AS tableId,
    CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(active_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(active_logical_bytes,total_logical_bytes),0)*logical_cost END AS shortTermPrice,
    CASE WHEN storage_model="PHYSICAL" THEN IFNULL(SAFE_DIVIDE(long_term_physical_bytes,total_physical_bytes),0)*physical_cost ELSE IFNULL(SAFE_DIVIDE(long_term_logical_bytes,total_logical_bytes),0)*logical_cost END AS longTermPrice,
    DATE(ts) AS runDate,
    ROW_NUMBER() OVER(PARTITION BY project_id, dataset_id, table_id, DATE(ts) ORDER BY ts DESC) AS _rnk
  FROM
    ` + "`{projectIdPlaceHolder}.{datasetIdPlaceHolder}.{tablesDiscoveryTable}`" + `
  WHERE
    DATE(_PARTITIONTIME) >= '{startDate}'
    AND DATE(ts) >= '{startDate}'
    AND project_id IS NOT NULL
    AND dataset_id IS NOT NULL
    AND table_id IS NOT NULL ),
  tablesDedup AS (
  SELECT
    * EXCEPT(_rnk)
  FROM
    tables
  WHERE
    _rnk = 1 ),
  dailyData AS (
  SELECT
    projectId,
    datasetId,
    tableId,
    runDate,
    ROUND(SUM(shortTermPrice), 4) AS shortTermPrice,
    ROUND(SUM(longTermPrice), 4) AS longTermPrice
  FROM
    tablesDedup
  GROUP BY
    projectId,
    datasetId,
    tableId,
    runDate )
SELECT
  projectId,
  datasetId,
  tableId,
  AVG(longTermPrice) + AVG(shortTermPrice) AS storagePrice,
  AVG(longTermPrice) AS longTermStoragePrice,
  AVG(shortTermPrice) AS shortTermStoragePrice,
FROM
  dailyData
GROUP BY
  projectId,
  datasetId,
  tableId
ORDER BY
  storagePrice DESC
LIMIT
  10
`

const slotsExplorer = `
WITH
{jobsDeduplicatedWithClause}
allocatedSlotsEvents as (
  SELECT
    startTime,
    endTime,
    totalSlotMs / executionTimeMs as slots,
    GENERATE_TIMESTAMP_ARRAY(startTime, endTime, INTERVAL 1 SECOND) AS tArr
  FROM
    jobsDeduplicated
),
unnested AS (
  SELECT
    *
  FROM
    allocatedSlotsEvents,
    UNNEST(tArr) t
),
truncated AS (
  SELECT
    startTime,
    endTime,
    TIMESTAMP_TRUNC(t, SECOND) as t,
    slots
  FROM
    unnested
),
groupedBySecond AS (
  SELECT
    t,
    SUM(slots) AS slotsPerSecond
  FROM
    truncated
  GROUP BY
    1
),
peakSlotsPerDayAndHour AS (
  SELECT
    DATE(t) AS date,
    EXTRACT(
      HOUR
      FROM
        t
    ) AS hour,
    MAX(slotsPerSecond) AS maxSlots
  FROM
    groupedBySecond
  GROUP BY
    1,
    2
),
aggregationByDayAndHourAndSecond AS (
  SELECT
    DATE(startTime) AS date,
    EXTRACT(
      HOUR
      FROM
        startTime
    ) AS hour,
    SUM(totalSlotMs) /(60 * 60 * 1000) AS avgSlots
  FROM
    jobsDeduplicated
  GROUP BY
    1,
    2
)
SELECT
  a.date AS day,
  a.hour,
  ROUND(a.avgSlots, 2) AS avgSlots,
  ROUND(p.maxSlots, 2) AS maxSlots
FROM
  aggregationByDayAndHourAndSecond a
  LEFT JOIN peakSlotsPerDayAndHour p ON a.hour = p.hour
  AND a.date = p.date
ORDER BY
  1,
  2
`

const billingProjectSlots = `
WITH
{jobsDeduplicatedWithClause}
  topBillingProjects AS (
    SELECT
      billingProjectId,
      ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '{startDate}', MILLISECOND)), 4) AS slots
    FROM
      jobsDeduplicated
    WHERE
      totalBilledBytes > 0
    GROUP BY
      1
    ORDER BY
      slots DESC
    LIMIT
      10 )
  SELECT
    *
  FROM
    topBillingProjects
  WHERE
    slots > 0
  ORDER BY
    slots DESC
`

const billingProjectSlotsTopUsers = `
WITH
{jobsDeduplicatedWithClause}
topBillingProjects AS (
  SELECT
    billingProjectId,
    ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '{startDate}', MILLISECOND)), 4) AS slots
  FROM
    jobsDeduplicated
  GROUP BY
    1
  ORDER BY
    slots DESC
  LIMIT
    10 ),
  topUsers AS (
  SELECT
    billingProjectId,
    user_email,
    ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '{startDate}', MILLISECOND)), 4) AS slots
  FROM
    jobsDeduplicated
  GROUP BY
    1, 2 ),
  topUsersWithRank AS (
  SELECT
    *,
    ROW_NUMBER() OVER(PARTITION BY billingProjectId ORDER BY slots DESC) AS _rnk
  FROM
    topUsers
  WHERE
    slots > 0
    AND billingProjectId IN (SELECT billingProjectId FROM topBillingProjects)
  )
SELECT
  * EXCEPT(_rnk)
FROM
  topUsersWithRank
WHERE
  _rnk <= 20
ORDER BY
  slots DESC
`

const billingProjectSlotsTopQueries = `
WITH
{jobsDeduplicatedWithClause}
  topBillingProjects AS (
    SELECT
      billingProjectId,
      ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '{startDate}', MILLISECOND)), 4) AS slots
    FROM
      jobsDeduplicated
    GROUP BY
      1
    ORDER BY
      slots DESC
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
      ROW_NUMBER() OVER(PARTITION BY billingProjectId ORDER BY avgSlots DESC) AS _rnk
    FROM
      topQueries
    WHERE
      avgSlots > 0
      AND billingProjectId IN ( SELECT billingProjectId FROM topBillingProjects) )
  SELECT
    * EXCEPT(_rnk)
  FROM
    topQueriesWithRank
  WHERE
    _rnk <= 20
  ORDER BY
    avgSlots DESC
`

const userSlots = `
WITH
{jobsDeduplicatedWithClause}
    topUsers AS (
    SELECT
      user_email AS userId,
      ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '{startDate}', MILLISECOND)), 4) AS slots
    FROM
      jobsDeduplicated
    GROUP BY
      1
    ORDER BY
    slots DESC
    LIMIT
      10 )
SELECT
  *
FROM
  topUsers
WHERE
  slots > 0
ORDER BY
  slots DESC
`

const userSlotsTopQueries = `
WITH
{jobsDeduplicatedWithClause}
topUsers AS (
  SELECT
    user_email AS userId,
    ROUND(SUM(totalSlotMs / TIMESTAMP_DIFF(CURRENT_TIMESTAMP(), '{startDate}', MILLISECOND)), 4) AS slots
  FROM
    jobsDeduplicated
  GROUP BY
    1
  ORDER BY
  slots DESC
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
    jobsDeduplicated
  GROUP BY
    userId,
    queryHash ),
  topQueriesWithRank AS (
  SELECT
    * EXCEPT(jobInfo),
    SPLIT(jobInfo, '&')[OFFSET(0)] AS jobId,
    SPLIT(jobInfo, '&')[OFFSET(1)] AS location,
    SPLIT(jobInfo, '&')[OFFSET(2)] AS billingProjectId,
    ROW_NUMBER() OVER(PARTITION BY userId ORDER BY avgSlots DESC) AS _rnk
  FROM
    topQueries
  WHERE
    avgSlots>0
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

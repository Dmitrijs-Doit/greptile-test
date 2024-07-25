UNION ALL
  -- HISTORICAL JOBS
SELECT
    user_email,
    ts AS timestamp,
    jobId,
    location,
    projectId AS billingProjectId,
    startTime,
    endTime,
    totalBytesBilled AS totalBilledBytes,
    referencedTables,
    reservationUsage,
    totalSlotMs,
    query,
    SHA256(query) as queryHash
FROM
    `mock-project-id.mock-dataset-id.historicalJobs`
WHERE
    jobId IS NOT NULL
  AND totalBytesBilled IS NOT NULL
  AND user_email IS NOT NULL
  AND user_email != ""
  AND DATE(startTime) <= '2024-03-26'
  AND DATE(startTime) >= '2024-03-27'
  AND (projectId IS NOT NULL OR projectId IN ("project1","project2","project3")) -- 'NOT' for excluding reservations, '' for reservations only
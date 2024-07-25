package savings

const rdsOnDemandQuery = `
@getKeyFromSystemLabelsQuery
SELECT
  SUM(report[
  OFFSET
    (0)].cost) AS cost,
  TIMESTAMP(DATE_TRUNC(usage_date_time, MONTH)) AS usage_date
FROM
  @table
WHERE
  getKeyFromSystemLabels(system_labels, "cmp/flexsave_eligibility") IN
  ("flexsave_eligible_uncovered", "flexsave_eligible_covered", "flexsave_eligible_customer_covered")
  AND DATE(usage_date_time) BETWEEN @start
  AND @end
  AND DATE(export_time) BETWEEN @start
  AND @end
  AND cost_type IN ("Usage","FlexsaveUsage")
  AND (sku_description LIKE "%HeavyUsage%"
    OR sku_description LIKE "%Usage:db%")
  AND (service_id IN ("AmazonRDS")
    OR service_description IN ("Amazon Relational Database Service"))
GROUP BY
  usage_date
ORDER BY
  usage_date`

const rdsSavingsQuery = `
	SELECT
		SUM(report[
		OFFSET
			(0)].savings) AS cost,
		TIMESTAMP(DATE_TRUNC(usage_date_time, MONTH)) AS usage_date
	FROM
		@table
	WHERE
		DATETIME(export_time) >= DATETIME(@start)
		AND DATETIME(export_time) < DATETIME(@end)
		AND cost_type IN ("FlexsaveManagementFee")
		AND (service_id IN ("AmazonRDS")
		OR service_description IN ("Amazon Relational Database Service"))
	GROUP BY
		usage_date
	ORDER BY usage_date
`

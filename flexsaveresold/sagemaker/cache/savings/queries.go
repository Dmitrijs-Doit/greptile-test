package savings

const sageMakerOnDemandQuery = `
@getKeyFromSystemLabelsQuery
SELECT
  SUM(report[
  OFFSET
    (0)].cost) AS cost,
	TIMESTAMP(DATE_TRUNC(usage_date_time, MONTH)) AS usage_date
FROM @table
WHERE
  getKeyFromSystemLabels(system_labels, "cmp/flexsave_eligibility") IN ("flexsave_eligible_uncovered", "flexsave_eligible_covered")
AND DATE(usage_date_time) BETWEEN @start AND @end
AND DATE(export_time) BETWEEN @start AND @end
AND cost_type IN ("Usage","FlexsaveCoveredUsage")
AND (service_id IN ('MachineLearningSavingsPlans','AmazonSageMaker')
OR service_description IN ("Amazon SageMaker"))
GROUP BY
  usage_date
ORDER BY
  usage_date
`

const sageMakerSavingsQuery = `
	SELECT
		-1 * SUM(cost) AS cost, TIMESTAMP(DATE_TRUNC(usage_date_time, MONTH)) AS usage_date
	FROM
		@table
	WHERE
		DATE(usage_start_time) >= DATETIME(@start)
		AND DATE(usage_start_time) < DATETIME(@end)
		AND cost_type IN ("FlexsaveCharges", "FlexsaveNegation")
		AND (service_id IN ('MachineLearningSavingsPlans','AmazonSageMaker')
		OR service_description IN ("Amazon SageMaker"))
		AND DATETIME(export_time) >= DATETIME(@start)
		AND DATETIME(export_time) < DATETIME(@end)
	GROUP BY
		usage_date
	ORDER BY usage_date
`

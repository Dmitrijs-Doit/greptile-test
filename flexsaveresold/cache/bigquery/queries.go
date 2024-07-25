package bq

const computeSavingsQuery = `WITH savings AS (
	SELECT
		-1 * SUM(cost) AS cost, TIMESTAMP(DATE_TRUNC(usage_date_time, MONTH)) AS usage_date
	FROM
		@table
	WHERE
		DATE(usage_start_time) >= DATETIME(@start)
		AND DATE(usage_start_time) < DATETIME(@end)
		AND cost_type IN ("FlexsaveCharges", "FlexsaveNegation")
		AND service_id NOT IN ('MachineLearningSavingsPlans','AmazonSageMaker')
		AND service_description NOT IN ("Amazon SageMaker")
		AND DATETIME(export_time) >= DATETIME(@start)
		AND DATETIME(export_time) < DATETIME(@end)
	GROUP BY
		usage_date
	ORDER BY usage_date
),
recurring AS (
	SELECT
		-1 * SUM(cost) AS cost, TIMESTAMP(DATE_TRUNC(usage_date_time, MONTH)) AS usage_date
	FROM
		@table
	WHERE
		DATE(usage_start_time) >= DATETIME(@start)
		AND DATE(usage_start_time) < DATETIME(@end)
		AND cost_type = "FlexsaveRecurringFee"
		AND service_id NOT IN ('MachineLearningSavingsPlans','AmazonSageMaker')
		AND service_description NOT IN ("Amazon SageMaker")
		AND DATETIME(export_time) >= DATETIME(@start)
		AND DATETIME(export_time) < DATETIME(@end)
	GROUP BY
		usage_date
	ORDER BY usage_date
)
SELECT
	savings.usage_date,
	savings.cost + recurring.cost AS cost
FROM
	savings
LEFT JOIN
	recurring ON savings.usage_date = recurring.usage_date`

const computeOnDemandQuery = `
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
	AND service_id NOT IN ('MachineLearningSavingsPlans','AmazonSageMaker')
	AND service_description NOT IN ("Amazon SageMaker")
	GROUP BY
	  usage_date
	ORDER BY
	  usage_date
	`

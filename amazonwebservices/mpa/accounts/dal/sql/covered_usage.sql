CREATE TEMP FUNCTION `getKeyFromSystemLabels`(labels ARRAY<STRUCT<key STRING, value STRING>>, key_lookup STRING)
	RETURNS STRING AS (
	  (SELECT l.value FROM UNNEST(labels) l WHERE l.key = key_lookup LIMIT 1)
	);

SELECT
  IFNULL(SUM(aws_metric.sp_effective_cost), 0) AS sp_cost,
  IFNULL(SUM(aws_metric.ri_effective_cost), 0) AS ri_cost
FROM
  `payer_accounts.payer_account_doit_reseller_account_n%d_%s`
WHERE
  DATE(export_time) = DATE(@export_date) # this needs to be 3 days ago
  AND project_id = @account_id
  AND cost_type in ('SavingsPlanCoveredUsage', 'DiscountedUsage')
  AND (getKeyFromSystemLabels(system_labels, 'aws/sp_arn') IN UNNEST(@sp_arns) %s)

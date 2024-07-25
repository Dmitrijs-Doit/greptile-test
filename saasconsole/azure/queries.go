package azure

func (s *AzureSaaSConsoleService) buildAssetsQuery(customerID string, isProd bool) string {
	env := ""
	if !isProd {
		env = "-dev"
	}

	return `CREATE TEMP FUNCTION
	getKeyFromSystemLabels(labels ARRAY<STRUCT<KEY STRING,
	  value STRING>>,
	  key_lookup STRING)
	RETURNS STRING AS ( (
	  SELECT
		l.value
	  FROM
		UNNEST(labels) l
	  WHERE
		l.key = key_lookup
	  LIMIT
		1) );
  
  SELECT
	billing_account_id AS subscriptionId,
	getKeyFromSystemLabels(system_labels,
		'azure/subscription_name') AS subscription_name,
	getKeyFromSystemLabels(system_labels,
		'azure/billing_profile_name') AS billing_profile_name,
	getKeyFromSystemLabels(system_labels,
		'azure/product_order_name') AS product_order_name,
	sku_id,
	sku_description,
	project_id,
	getKeyFromSystemLabels(system_labels,
		'azure/billing_profile_id') AS billing_profile_id,
	getKeyFromSystemLabels(system_labels,
	  'azure/billing_account_id') AS billing_account_id,
	resource_id,
	customer_type,
	customer_id
  FROM (
	SELECT
	  billing_account_id,
	  sku_id,
	  sku_description,
	  project_id,
	  system_labels,
	  resource_id,
	  customer_type,
	  customer_id,
	  ROW_NUMBER() OVER(PARTITION BY billing_account_id ORDER BY billing_account_id) AS row_num
	FROM
	  doitintl-cmp-azure-data` + env + `.azure_billing_` + customerID + `.doitintl_billing_export_v1_` + customerID + `
	WHERE
	  TIMESTAMP_TRUNC(export_time, DAY) = @lastDay
	  AND customer_type = "standalone"
  )
  WHERE
	row_num = 1
	`
}

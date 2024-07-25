package bqmodels

type BillingProjectsWithReservationsResult struct {
	CustomerID string `bigquery:"customer"`
	Project    string `bigquery:"project"`
	Location   string `bigquery:"location"`
}

const BillingProjectsWithEditionsQuery = `
SELECT
  discounts.customer,
  project.id as project,
  location.location
FROM
  ` + "`doitintl-cmp-gcp-data.gcp_billing.gcp_raw_billing`" + ` billing
RIGHT JOIN (
  SELECT
    DISTINCT billing_account_id,
    customer
  FROM
    ` + "`doitintl-cmp-gcp-data.gcp_billing.gcp_discounts_v1`" + `
  WHERE
    DATE(_PARTITIONTIME) = CURRENT_DATE()) discounts
ON
  billing.billing_account_id = discounts.billing_account_id
WHERE
  DATE(export_time) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 3 DAY))
  AND service.description like "%BigQuery%"
  AND (LOWER(sku.description) LIKE "%edition%" OR (LOWER(sku.description) LIKE "%flat rate%" AND LOWER(sku.description) NOT LIKE "%bi engine%"))
GROUP BY
1,2,3
`

const BillingProjectsWithEditionsSingleCustomerQuery = `
SELECT
  discounts.customer,
  project.id as project,
  location.location
FROM
  doitintl-cmp-gcp-data.gcp_billing.gcp_raw_billing  billing
RIGHT JOIN (
  SELECT
    DISTINCT billing_account_id,
    customer
  FROM
    doitintl-cmp-gcp-data.gcp_billing.gcp_discounts_v1
  WHERE
    DATE(_PARTITIONTIME) = CURRENT_DATE()) discounts
ON
  billing.billing_account_id = discounts.billing_account_id
WHERE
  DATE(export_time) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 3 DAY))
  AND service.description like "%BigQuery%"
  AND (LOWER(sku.description) LIKE "%edition%" OR (LOWER(sku.description) LIKE "%flat rate%" AND LOWER(sku.description) NOT LIKE "%bi engine%"))
  AND customer = "{customer_id}"
GROUP BY
1,2,3
`

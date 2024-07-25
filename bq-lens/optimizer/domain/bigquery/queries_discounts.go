package bqmodels

type DiscountsAllCustomersResult struct {
	CustomerID string  `bigquery:"customer"`
	Discount   float64 `bigquery:"discount"`
}

// TODO: CMP-19076 - Improve the query to use the latest discount and not max.
const DiscountsAllCustomers = `
SELECT
  customer,
  MAX(discount) AS discount
FROM
` + "`doitintl-cmp-gcp-data.gcp_billing.gcp_discounts_v1`" + `
WHERE
  DATE(_PARTITIONTIME) = CURRENT_DATE()
  AND is_active
  AND start_date<=CURRENT_DATE()
  AND (end_date IS NULL
    OR end_date >= CURRENT_DATE())
  AND discount > 0
GROUP BY
  customer
`

package domain

import "time"

type SKUBillingRecord struct {
	BillingAccountID string    `bigquery:"billing_account_id"`
	SKUID            string    `bigquery:"sku_id"`
	LatestUsageDate  time.Time `bigquery:"latest_usage_date"`
}

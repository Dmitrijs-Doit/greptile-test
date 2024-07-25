package domain

type SharedPayerSavingsDiscrepancies []SharedPayerSavingsDiscrepancy

type SharedPayerSavingsDiscrepancy struct {
	CustomerID       string  `bigquery:"customer_id"`
	LastMonthSavings float64 `bigquery:"last_month_savings"`
}

package pkg

import (
	"time"

	fspkg "github.com/doitintl/firestore/pkg"
)

type CustomerInputAttributes struct {
	CustomerID              string     `json:"customerId"`
	TimeEnabled             *time.Time `json:"timeEnabled"`
	TimeDisabled            *time.Time `json:"timeDisabled"`
	IsEnabled               bool       `json:"isEnabled"`
	PayerIDs                []string
	AssetIDs                []string
	DedicatedPayerStartTime *time.Time
}

type TimeParams struct {
	Now                time.Time
	CurrentMonth       string
	ApplicableMonths   []string
	DaysInCurrentMonth float64
	DaysInNextMonth    float64
	PreviousMonth      string
}

type SpendDataMonthly = map[string]*fspkg.FlexsaveMonthSummary

type ItemType struct {
	Cost float64   `bigquery:"cost"`
	Date time.Time `bigquery:"usage_date"`
}

type CreditRow struct {
	BillingAccountID string  `bigquery:"billing_account_id"`
	Cost             float64 `bigquery:"cost"`
}

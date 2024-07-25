package flexsaveresold

import (
	"time"

	"cloud.google.com/go/bigquery"
)

type project struct {
	ID     string   `json:"id"`
	Number string   `json:"number"`
	Name   string   `json:"name"`
	Labels []*label `json:"labels"`
}

type usage struct {
	Amount               float64 `json:"amount"`
	Unit                 string  `json:"unit"`
	AmountInPricingUnits float64 `json:"amount_in_pricing_units"`
	PricingUnit          string  `json:"pricing_unit"`
}

type label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type location struct {
	Location string `json:"location"`
	Country  string `json:"country"`
	Region   string `json:"region"`
	Zone     string `json:"zone"`
}

type invoice struct {
	Month string `json:"month"`
}

type report struct {
	Cost    float64              `json:"cost"`
	Usage   bigquery.NullFloat64 `json:"usage"`
	Savings float64              `json:"savings"`
	Credit  bigquery.NullString  `json:"credit"`
}

type billingItem struct {
	Customer               string                `json:"customer"`
	BillingAccountID       string                `json:"billing_account_id"`
	ServiceID              string                `json:"service_id"`
	ServiceDescription     string                `json:"service_description"`
	SkuID                  string                `json:"sku_id"`
	SkuDescription         string                `json:"sku_description"`
	UsageStartTime         time.Time             `json:"usage_start_time"`
	UsageEndTime           time.Time             `json:"usage_end_time"`
	UsageDateTime          bigquery.NullDateTime `json:"usage_date_time"`
	Project                *project              `json:"project"`
	Labels                 []*label              `json:"labels"`
	SystemLabels           []*label              `json:"system_labels"`
	Location               *location             `json:"location"`
	Cost                   float64               `json:"cost"`
	Currency               string                `json:"currency"`
	CurrencyConversionRate float64               `json:"currency_conversion_rate"`
	Usage                  *usage                `json:"usage"`
	Invoice                *invoice              `json:"invoice"`
	ExportTime             time.Time             `json:"export_time"`
	CostType               string                `json:"cost_type"`
	Report                 []report              `json:"report"`
	RowID                  string                `json:"row_id"`
	Description            string                `json:"description"`
}

var insertRowsSchema = bigquery.Schema{
	{Name: "billing_account_id", Required: false, Type: bigquery.StringFieldType},
	{Name: "project_id", Required: false, Type: bigquery.StringFieldType},
	{Name: "service_description", Required: false, Type: bigquery.StringFieldType},
	{Name: "service_id", Required: false, Type: bigquery.StringFieldType},
	{Name: "sku_description", Required: false, Type: bigquery.StringFieldType},
	{Name: "sku_id", Required: false, Type: bigquery.StringFieldType},
	{Name: "usage_date_time", Required: false, Type: bigquery.DateTimeFieldType},
	{Name: "usage_start_time", Required: false, Type: bigquery.TimestampFieldType},
	{Name: "usage_end_time", Required: false, Type: bigquery.TimestampFieldType},
	{Name: "project", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "id", Required: false, Type: bigquery.StringFieldType},
			{Name: "number", Required: false, Type: bigquery.StringFieldType},
			{Name: "name", Required: false, Type: bigquery.StringFieldType},
			{Name: "labels", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
				Schema: bigquery.Schema{
					{Name: "key", Required: false, Type: bigquery.StringFieldType},
					{Name: "value", Required: false, Type: bigquery.StringFieldType},
				},
			},
		},
	},
	{Name: "labels", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "key", Required: false, Type: bigquery.StringFieldType},
			{Name: "value", Required: false, Type: bigquery.StringFieldType},
		},
	},

	{Name: "system_labels", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "key", Required: false, Type: bigquery.StringFieldType},
			{Name: "value", Required: false, Type: bigquery.StringFieldType},
		},
	},
	{Name: "location", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "location", Required: false, Type: bigquery.StringFieldType},
			{Name: "country", Required: false, Type: bigquery.StringFieldType},
			{Name: "region", Required: false, Type: bigquery.StringFieldType},
			{Name: "zone", Required: false, Type: bigquery.StringFieldType},
		},
	},
	{Name: "export_time", Required: false, Type: bigquery.TimestampFieldType},
	{Name: "cost", Required: false, Type: bigquery.FloatFieldType},
	{Name: "currency", Required: false, Type: bigquery.StringFieldType},
	{Name: "currency_conversion_rate", Required: false, Type: bigquery.FloatFieldType},
	{Name: "usage", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "amount", Required: false, Type: bigquery.FloatFieldType},
			{Name: "unit", Required: false, Type: bigquery.StringFieldType},
			{Name: "amount_in_pricing_units", Required: false, Type: bigquery.FloatFieldType},
			{Name: "pricing_unit", Required: false, Type: bigquery.StringFieldType},
		},
	},
	{Name: "invoice", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "month", Required: false, Type: bigquery.StringFieldType},
		},
	},
	{Name: "cost_type", Required: false, Type: bigquery.StringFieldType},
	{Name: "report", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "cost", Required: false, Type: bigquery.FloatFieldType},
			{Name: "usage", Required: false, Type: bigquery.FloatFieldType},
			{Name: "savings", Required: false, Type: bigquery.FloatFieldType},
			{Name: "credit", Required: false, Type: bigquery.StringFieldType},
		},
	},
	{Name: "aws_metric", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "sp_amortized_commitment", Required: false, Type: bigquery.FloatFieldType},
			{Name: "sp_recurring_commitment", Required: false, Type: bigquery.FloatFieldType},
			{Name: "sp_effective_cost", Required: false, Type: bigquery.FloatFieldType},
			{Name: "sp_plan_rate", Required: false, Type: bigquery.FloatFieldType},
			{Name: "sp_total_commitment_to_date", Required: false, Type: bigquery.FloatFieldType},
			{Name: "sp_used_commitment", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_amortized_upfront_usage_cost", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_amortized_upfront_period_fee", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_effective_cost", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_recurring_usage_fee", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_unused_amortized_period_fee", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_unused_nfu", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_unused_recurring_fee", Required: false, Type: bigquery.FloatFieldType},
			{Name: "ri_upfront_value", Required: false, Type: bigquery.FloatFieldType},
			{Name: "public_ondemand_rate", Required: false, Type: bigquery.FloatFieldType},
			{Name: "public_ondemand_cost", Required: false, Type: bigquery.FloatFieldType},
		},
	},
	{Name: "row_id", Required: false, Type: bigquery.StringFieldType},
	{Name: "description", Required: false, Type: bigquery.StringFieldType},
	{Name: "etl", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "ts", Required: false, Type: bigquery.TimestampFieldType},
			{Name: "file_update_time", Required: false, Type: bigquery.TimestampFieldType},
			{Name: "session_id", Required: false, Type: bigquery.StringFieldType},
			{Name: "manifest_update_time", Required: false, Type: bigquery.TimestampFieldType},
			{Name: "run_start_time", Required: false, Type: bigquery.TimestampFieldType},
		},
	},
	{Name: "customer", Required: false, Type: bigquery.StringFieldType},
}

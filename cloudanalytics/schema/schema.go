package schema

import (
	"time"

	"cloud.google.com/go/bigquery"
)

// see here on how to extend schema: https://cloud.google.com/bigquery/docs/managing-table-schemas#go
var BaseSchema = bigquery.Schema{
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
			{Name: "ancestry_numbers", Required: false, Type: bigquery.StringFieldType},
			{Name: "ancestors", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
				Schema: bigquery.Schema{
					{Name: "resource_name", Required: false, Type: bigquery.StringFieldType},
					{Name: "display_name", Required: false, Type: bigquery.StringFieldType},
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
	{Name: "credits", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "name", Required: false, Type: bigquery.StringFieldType},
			{Name: "amount", Required: false, Type: bigquery.FloatFieldType},
			{Name: "full_name", Required: false, Type: bigquery.StringFieldType},
			{Name: "id", Required: false, Type: bigquery.StringFieldType},
			{Name: "type", Required: false, Type: bigquery.StringFieldType},
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
			{Name: "ext_metric", Required: false, Type: bigquery.RecordFieldType,
				Schema: bigquery.Schema{
					{Name: "key", Required: false, Type: bigquery.StringFieldType},
					{Name: "value", Required: false, Type: bigquery.FloatFieldType},
					{Name: "type", Required: false, Type: bigquery.StringFieldType},
				},
			},
		},
	},
	{Name: "adjustment_info", Required: false, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "id", Required: false, Type: bigquery.StringFieldType},
			{Name: "description", Required: false, Type: bigquery.StringFieldType},
			{Name: "mode", Required: false, Type: bigquery.StringFieldType},
			{Name: "type", Required: false, Type: bigquery.StringFieldType},
		},
	},
	{Name: "tags", Required: false, Repeated: true, Type: bigquery.RecordFieldType,
		Schema: bigquery.Schema{
			{Name: "key", Required: false, Type: bigquery.StringFieldType},
			{Name: "value", Required: false, Type: bigquery.StringFieldType},
			{Name: "inherited", Required: false, Type: bigquery.BooleanFieldType},
			{Name: "namespace", Required: false, Type: bigquery.StringFieldType},
		},
	},
}

var customer = bigquery.FieldSchema{Name: "customer", Required: true, Type: bigquery.StringFieldType}
var cloudProvider = bigquery.FieldSchema{Name: "cloud_provider", Required: true, Type: bigquery.StringFieldType}

var CreditsSchema = append(bigquery.Schema{&customer, &cloudProvider}, BaseSchema...)

type Project struct {
	ID     string  `json:"id"`
	Number string  `json:"number"`
	Name   string  `json:"name"`
	Labels []Label `json:"labels"`
}

type Usage struct {
	Amount               float64 `json:"amount"`
	Unit                 string  `json:"unit"`
	AmountInPricingUnits float64 `json:"amount_in_pricing_units"`
	PricingUnit          string  `json:"pricing_unit"`
}

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Location struct {
	Location string `json:"location"`
	Country  string `json:"country"`
	Region   string `json:"region"`
	Zone     string `json:"zone"`
}

type Invoice struct {
	Month string `json:"month"`
}

type Report struct {
	Cost      bigquery.NullFloat64 `json:"cost"`
	Usage     bigquery.NullFloat64 `json:"usage"`
	Savings   bigquery.NullFloat64 `json:"savings"`
	Credit    bigquery.NullString  `json:"credit"`
	ExtMetric *ReportExtMetric     `json:"ext_metric"`
}

type ReportExtMetric struct {
	Key   string  `json:"key"`
	Value float64 `json:"value"`
	Type  string  `json:"type"`
}

type Credit struct {
	Name     bigquery.NullString  `json:"name"`
	Amount   bigquery.NullFloat64 `json:"amount"`
	FullName bigquery.NullString  `json:"full_name"`
	ID       bigquery.NullString  `json:"id"`
	Type     bigquery.NullString  `json:"type"`
}

type AdjustmentInfo struct {
	ID          bigquery.NullString `json:"id"`
	Description bigquery.NullString `json:"description"`
	Mode        bigquery.NullString `json:"mode"`
	Type        bigquery.NullString `json:"type"`
}
type BillingRow struct {
	AdjustmentInfo         *AdjustmentInfo       `json:"adjustment_info"`
	BillingAccountID       string                `json:"billing_account_id"`
	CloudProvider          string                `json:"cloud_provider"`
	Cost                   float64               `json:"cost"`
	CostType               string                `json:"cost_type"`
	Credits                []Credit              `json:"credits"`
	Currency               string                `json:"currency"`
	CurrencyConversionRate float64               `json:"currency_conversion_rate"`
	Customer               string                `json:"customer"`
	ExportTime             time.Time             `json:"export_time"`
	Invoice                *Invoice              `json:"invoice"`
	Labels                 []Label               `json:"labels"`
	Location               *Location             `json:"location"`
	Project                *Project              `json:"project"`
	ProjectID              bigquery.NullString   `json:"project_id"`
	Report                 []Report              `json:"report"`
	ServiceDescription     bigquery.NullString   `json:"service_description"`
	ServiceID              bigquery.NullString   `json:"service_id"`
	SkuDescription         bigquery.NullString   `json:"sku_description"`
	SkuID                  bigquery.NullString   `json:"sku_id"`
	SystemLabels           []Label               `json:"system_labels"`
	Usage                  *Usage                `json:"usage"`
	UsageDateTime          bigquery.NullDateTime `json:"usage_date_time"`
	UsageEndTime           time.Time             `json:"usage_end_time"`
	UsageStartTime         time.Time             `json:"usage_start_time"`
}

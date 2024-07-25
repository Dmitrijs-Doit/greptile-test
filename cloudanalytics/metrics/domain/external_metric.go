package metrics

import (
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

const MetricField = "metric"

type ExternalBasicMetric string

// swagger:enum ExternalMetricType
type ExternalMetricType string

const (
	ExternalMetricTypeBasic    ExternalMetricType = "basic"
	ExternalMetricTypeCustom   ExternalMetricType = "custom"
	ExternalMetricTypeExtended ExternalMetricType = "extended"
)

// The metric to apply.
type ExternalMetric struct {
	Type ExternalMetricType `json:"type" binding:"required"`
	// For basic metrics the value can be one of: ["cost", "usage", "savings"]
	//
	// If using custom metrics, the value must refer to an existing custom metric id.
	Value string `json:"value" binding:"required"`
}

type InternalMetricParameters struct {
	Metric         *report.Metric
	CustomMetric   *firestore.DocumentRef
	ExtendedMetric *string
}

const (
	ExternalBasicMetricCost    ExternalBasicMetric = "cost"
	ExternalBasicMetricUsage   ExternalBasicMetric = "usage"
	ExternalBasicMetricSavings ExternalBasicMetric = "savings"
)

var ExternalToInternalBasicMetricMap = map[ExternalBasicMetric]report.Metric{
	ExternalBasicMetricCost:    report.MetricCost,
	ExternalBasicMetricUsage:   report.MetricUsage,
	ExternalBasicMetricSavings: report.MetricSavings,
}

var InternalToExternalBasicMetricMap = map[report.Metric]ExternalBasicMetric{
	report.MetricCost:    ExternalBasicMetricCost,
	report.MetricUsage:   ExternalBasicMetricUsage,
	report.MetricSavings: ExternalBasicMetricSavings,
}

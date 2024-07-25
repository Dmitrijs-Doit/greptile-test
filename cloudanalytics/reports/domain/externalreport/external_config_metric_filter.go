package externalreport

import metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"

// The metric filter to apply to this report
// example:
//
//	{
//		"metric": {
//			"type":  "basic",
//			"value": "cost"
//		  },
//		"operator" : "gt",
//		"values" : [50]
//	}
type ExternalConfigMetricFilter struct {
	Metric   metrics.ExternalMetric `json:"metric" binding:"required"`
	Operator ExternalMetricFilter   `json:"operator" binding:"required"`
	Values   []float64              `json:"values" binding:"required"`
}

package metrics

import (
	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// Metric Format
const (
	MetricNumericFormat int = iota
	MetricPercentageFormat
)

type CalculatedMetric struct {
	Name        string                      `json:"name" firestore:"name"`
	Description string                      `json:"description" firestore:"description"`
	Customer    *firestore.DocumentRef      `json:"customer" firestore:"customer"`
	Type        string                      `json:"type" firestore:"type"`
	Owner       string                      `json:"owner" firestore:"owner"`
	Formula     string                      `json:"formula" firestore:"formula"`
	Variables   []*CalculatedMetricVariable `json:"variables" firestore:"variables"`
	Format      int                         `json:"format" firestore:"format"`
	Labels      []*firestore.DocumentRef    `json:"labels" firestore:"labels"`

	ID string `json:"id" firestore:"-"`
}

type CalculatedMetricVariable struct {
	Metric      report.Metric          `json:"metric" firestore:"metric"`
	Attribution *firestore.DocumentRef `json:"attribution" firestore:"attribution"`
}

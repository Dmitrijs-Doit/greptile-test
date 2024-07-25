package domain

import (
	"cloud.google.com/go/bigquery"

	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type RunReportFromExternalConfigRequest struct {
	Config domainExternalReport.ExternalConfig `json:"config"`
}

type RunReportResult struct {
	Schema       []*SchemaField     `json:"schema"`
	MlFeatures   []report.Feature   `json:"mlFeatures,omitempty"`
	Rows         [][]bigquery.Value `json:"rows"`
	ForecastRows [][]bigquery.Value `json:"forecastRows,omitempty"`
}

type SchemaField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type DateTimeIndex struct {
	Year  int
	Month int
	Day   int
	Hour  int
	Week  int
}

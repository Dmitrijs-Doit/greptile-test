package domain

import "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"

type DataHubMetric struct {
	DataSource report.DataSource `firestore:"dataSource"`
	Key        string            `firestore:"key"`
	Label      string            `firestore:"label"`
}

type DataHubMetrics struct {
	Metrics []DataHubMetric `firestore:"metrics"`
}

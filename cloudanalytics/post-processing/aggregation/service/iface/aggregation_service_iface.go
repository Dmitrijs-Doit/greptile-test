//go:generate mockery --name=AggregationService --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"cloud.google.com/go/bigquery"

	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type AggregationService interface {
	ApplyAggregation(aggregator reportDomain.Aggregator, numRows int, numCols int, resRows [][]bigquery.Value) error
}

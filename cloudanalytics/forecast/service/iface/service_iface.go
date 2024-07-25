//go:generate mockery --name Service --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/domain"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

type Service interface {
	GetForecastOriginAndResultRows(
		ctx context.Context,
		queryResultRows [][]bigquery.Value,
		queryRequestRows int,
		queryRequestCols []*domainQuery.QueryRequestX,
		interval string,
		metric int,
		maxRefreshTime, from, to time.Time,
	) ([]*domain.OriginSeries, [][]bigquery.Value, error)
}

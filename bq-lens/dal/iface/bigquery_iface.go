//go:generate mockery --name Bigquery --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/dal"
	discoveryDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/domain"
)

type Bigquery interface {
	EnsureTableIsCorrect(ctx context.Context, bq *bigquery.Client) (*bigquery.Table, error)
	GetRegionsAndStorageBillingModelForProject(ctx context.Context, projectID string, bq *bigquery.Client) (discoveryDomain.DatasetStorageBillingModel, []string, error)
	RunDiscoveryQuery(
		ctx context.Context,
		bq *bigquery.Client,
		query string,
		destinationTable *bigquery.Table,
		rowProcessor dal.RowProcessor,
	) error
}

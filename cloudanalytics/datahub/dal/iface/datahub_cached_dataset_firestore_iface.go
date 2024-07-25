//go:generate mockery --name=DataHubCachedDatasetFirestore --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

type DataHubCachedDatasetFirestore interface {
	Get(ctx context.Context, customerID string) (*domain.CachedDatasetsRes, error)
	Update(
		ctx context.Context,
		customerID string,
		cachedDatasetsRes *domain.CachedDatasetsRes,
	) error
	DeleteItems(
		ctx context.Context,
		customerID string,
		datasets []string,
	) error
}

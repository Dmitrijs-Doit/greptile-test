//go:generate mockery --name=DataHubDatasetFirestore --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

type DataHubDatasetFirestore interface {
	List(ctx context.Context, customerID string) ([]domain.DatasetMetadata, error)
	Create(ctx context.Context, customerID string, dataset domain.DatasetMetadata) error
	Delete(
		ctx context.Context,
		customerID string,
		datasetIDs []string,
	) error
}

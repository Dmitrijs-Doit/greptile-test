//go:generate mockery --name=DataHubBatchesFirestore --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

type DataHubBatchesFirestore interface {
	Get(ctx context.Context, customerID string, datasetName string) (*domain.DatasetBatchesRes, error)
	Update(
		ctx context.Context,
		customerID string,
		datasetName string,
		datasetBatchesRes *domain.DatasetBatchesRes,
	) error
	Delete(ctx context.Context, customerID string, datasetName string) error
	DeleteBatches(
		ctx context.Context,
		customerID string,
		datasetName string,
		batches []string,
	) error
}

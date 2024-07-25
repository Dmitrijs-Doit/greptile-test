//go:generate mockery --name DataHubService --output ../mocks
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
)

// DataHubService is the interface for the DataHub service
type DataHubService interface {
	DeleteCustomerData(ctx context.Context, customerID string) error
	DeleteCustomerDataByEventIDs(
		ctx context.Context,
		customerID string,
		deleteEventsReq domain.DeleteEventsReq,
		deletedBy string,
	) error
	DeleteCustomerDataByClouds(
		ctx context.Context,
		customerID string,
		deleteRequest domain.DeleteDatasetsReq,
		deletedBy string,
	) error
	DeleteCustomerDataHard(ctx context.Context, customerID string) error
	DeleteAllCustomersDataHard(ctx context.Context) error
	DeleteDatasetBatches(
		ctx context.Context,
		customerID string,
		datasetName string,
		deleteBatchesReq domain.DeleteBatchesReq,
		deletedBy string,
	) error
	GetCustomerDatasets(
		ctx context.Context,
		customerID string,
		forceRefresh bool,
	) (*domain.CachedDatasetsRes, error)
	GetCustomerDatasetBatches(
		ctx context.Context,
		customerID string,
		datasetName string,
		forceRefresh bool,
	) (*domain.DatasetBatchesRes, error)
	AddRawEvents(
		ctx context.Context,
		customerID string,
		email string,
		rawEventsReq domain.RawEventsReq,
	) ([]*domain.Event, []errormsg.ErrorMsg, error)
	CreateDataset(
		ctx context.Context,
		customerID string,
		email string,
		datasetReq domain.CreateDatasetRequest,
	) error
}

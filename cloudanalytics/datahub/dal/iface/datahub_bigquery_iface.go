//go:generate mockery --name=DataHubBigQuery --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
)

type DataHubBigQuery interface {
	DeleteBigQueryData(
		ctx context.Context,
		customerID string,
	) error
	DeleteBigQueryDataByEventIDs(
		ctx context.Context,
		customerID string,
		deleteEventsReq domain.DeleteEventsReq,
		deletedBy string,
	) error
	DeleteBigQueryDataByClouds(
		ctx context.Context,
		customerID string,
		deleteReq domain.DeleteDatasetsReq,
		deletedBy string,
	) error
	DeleteBigQueryDataByBatches(
		ctx context.Context,
		customerID string,
		datasetName string,
		deleteReq domain.DeleteBatchesReq,
		deletedBy string,
	) error
	DeleteBigQueryDataHard(
		ctx context.Context,
		customerID string,
		softDeleteIntervalDays int,
	) error
	GetCustomerDatasets(
		ctx context.Context,
		customerID string,
	) ([]domain.CachedDataset, error)
	GetCustomerDatasetBatches(
		ctx context.Context,
		customerID string,
		datasetName string,
	) ([]domain.DatasetBatch, error)
	GetCustomersWithSoftDeleteData(
		ctx context.Context,
		softDeleteIntervalDays int,
	) ([]string, error)
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	cloudtasks "github.com/doitintl/cloudtasks/iface"
	doitFirestore "github.com/doitintl/firestore"
	datahubDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	datahubMetricDALIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/iface"
	metadataDALIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	httpDoit "github.com/doitintl/http"
)

const (
	cacheValidityDuration = 15 * time.Minute

	softDeleteIntervalDays = 7

	deleteCustomerDatahubDataTaskPathTemplate = "/tasks/datahub/events/customers/%s/hard"
)

type Service struct {
	loggerProvider          logger.Provider
	datahubMetadataDAL      metadataDALIface.DataHubMetadataFirestore
	datahubMetricDAL        datahubMetricDALIface.DataHubMetricFirestore
	bqDAL                   datahubDalIface.DataHubBigQuery
	datahubCachedDatasetDAL datahubDalIface.DataHubCachedDatasetFirestore
	datahubDatasetDAL       datahubDalIface.DataHubDatasetFirestore
	datahubBatchesDAL       datahubDalIface.DataHubBatchesFirestore
	datahubInternalAPIDAL   datahubDalIface.DatahubInternalAPIDAL
	customerDAL             customerDal.Customers
	cloudTaskClient         cloudtasks.CloudTaskClient
	timeNow                 func() time.Time
}

func NewService(
	loggerProvider logger.Provider,
	datahubMetadataDAL metadataDALIface.DataHubMetadataFirestore,
	datahubMetricDAL datahubMetricDALIface.DataHubMetricFirestore,
	bqDAL datahubDalIface.DataHubBigQuery,
	datahubCachedDatasetDAL datahubDalIface.DataHubCachedDatasetFirestore,
	datahubDatasetDAL datahubDalIface.DataHubDatasetFirestore,
	datahubBatchesDAL datahubDalIface.DataHubBatchesFirestore,
	datahubInternalAPIDAL datahubDalIface.DatahubInternalAPIDAL,
	customerDAL customerDal.Customers,
	cloudTaskClient cloudtasks.CloudTaskClient,
) iface.DataHubService {
	return &Service{
		loggerProvider:          loggerProvider,
		datahubMetadataDAL:      datahubMetadataDAL,
		datahubMetricDAL:        datahubMetricDAL,
		bqDAL:                   bqDAL,
		datahubCachedDatasetDAL: datahubCachedDatasetDAL,
		datahubDatasetDAL:       datahubDatasetDAL,
		datahubBatchesDAL:       datahubBatchesDAL,
		datahubInternalAPIDAL:   datahubInternalAPIDAL,
		customerDAL:             customerDAL,
		cloudTaskClient:         cloudTaskClient,
		timeNow:                 func() time.Time { return time.Now().UTC() },
	}
}

var (
	now                = time.Now()
	processingDuration = 90 * time.Minute
)

// DeleteAllCustomersDataHard deletes all datahub soft-deleted data for all customers
func (s *Service) DeleteAllCustomersDataHard(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	customerIDs, err := s.bqDAL.GetCustomersWithSoftDeleteData(ctx, softDeleteIntervalDays)
	if err != nil {
		l.Errorf("failed to delete customer BQ data with error: %s", err)
		return err
	}

	for _, customerID := range customerIDs {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf(deleteCustomerDatahubDataTaskPathTemplate, customerID),
			Queue:  common.TaskQueueDatahubDeleteCustomerData,
		}

		conf := config.Config(nil)
		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			l.Errorf(err.Error())
			continue
		}
	}

	return nil
}

// DeleteCustomerData deletes all datahub data for a customer
func (s *Service) DeleteCustomerData(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)

	// Delete customer DataHub API BQ data
	if err := s.bqDAL.DeleteBigQueryData(ctx, customerID); err != nil {
		l.Errorf("failed to delete customer BQ data with error: %s", err)
		return err
	}

	// Delete customer DataHub API metadata
	if err := s.datahubMetadataDAL.DeleteCustomerMetadata(ctx, customerID); err != nil {
		l.Errorf("failed to delete customer metadata with error: %s", err)
		return err
	}

	// Delete customer DataHub API metrics list
	if err := s.datahubMetricDAL.Delete(ctx, customerID); err != nil {
		l.Errorf("failed to delete customer datahub metrics with error: %s", err)
		return err
	}

	return nil
}

func (s *Service) DeleteCustomerDataByEventIDs(
	ctx context.Context,
	customerID string,
	deleteEventsReq domain.DeleteEventsReq,
	deletedBy string,
) error {
	l := s.loggerProvider(ctx)

	if len(deleteEventsReq.EventIDs) == 0 {
		return domain.ErrEventsIDsCanNotBeEmpty
	}

	if err := s.bqDAL.DeleteBigQueryDataByEventIDs(
		ctx,
		customerID,
		deleteEventsReq,
		deletedBy,
	); err != nil {
		l.Errorf("failed to delete customer BQ data by event IDs with error: %s", err)
		return err
	}

	return nil
}

func (s *Service) DeleteCustomerDataByClouds(
	ctx context.Context,
	customerID string,
	deleteDatasetsReq domain.DeleteDatasetsReq,
	deletedBy string,
) error {
	l := s.loggerProvider(ctx)

	if len(deleteDatasetsReq.Datasets) == 0 {
		return domain.ErrCloudsCanNotBeEmpty
	}

	datasets, err := s.bqDAL.GetCustomerDatasets(
		ctx,
		customerID,
	)
	if err != nil {
		l.Errorf("failed to get customer datasets with error: %s", err)
		return err
	}

	cachedDatasetsRes := domain.CachedDatasetsRes{
		Items:    datasets,
		CachedAt: s.timeNow(),
	}

	if err := s.datahubCachedDatasetDAL.Update(ctx, customerID, &cachedDatasetsRes); err != nil {
		l.Errorf("error during saving datahub summary %s", err)
	}

	if isDatasetProcessing(datasets, deleteDatasetsReq.Datasets) {
		return domain.ErrDatasetIsProcessing
	}

	if err := s.bqDAL.DeleteBigQueryDataByClouds(
		ctx,
		customerID,
		deleteDatasetsReq,
		deletedBy,
	); err != nil {
		l.Errorf("failed to delete customer BQ data by clouds with error: %s", err)
		return err
	}

	if err := s.datahubDatasetDAL.Delete(
		ctx,
		customerID,
		deleteDatasetsReq.Datasets,
	); err != nil {
		l.Errorf("failed to delete customer metadata of datasets (clouds) with error: %s", err)
		return err
	}

	if err := s.datahubCachedDatasetDAL.DeleteItems(
		ctx,
		customerID,
		deleteDatasetsReq.Datasets,
	); err != nil {
		l.Errorf("failed to delete customer cached summary items with error: %s", err)
	}

	return nil
}

func (s *Service) DeleteDatasetBatches(
	ctx context.Context,
	customerID string,
	datasetName string,
	deleteBatchesReq domain.DeleteBatchesReq,
	deletedBy string,
) error {
	l := s.loggerProvider(ctx)

	if len(deleteBatchesReq.Batches) == 0 {
		return domain.ErrBatchesCanNotBeEmpty
	}

	batches, err := s.bqDAL.GetCustomerDatasetBatches(ctx, customerID, datasetName)
	if err != nil {
		l.Errorf("failed to get customer dataset batches with error: %s", err)
		return err
	}

	datasetBatchesRes := domain.DatasetBatchesRes{
		Items:    batches,
		CachedAt: s.timeNow(),
	}

	if err := s.datahubBatchesDAL.Update(ctx, customerID, datasetName, &datasetBatchesRes); err != nil {
		l.Errorf("error during saving datahub batches %s", err)
	}

	if isBatchProcessing(batches, deleteBatchesReq.Batches, s.timeNow()) {
		return domain.ErrBatchIsProcessing
	}

	if err := s.bqDAL.DeleteBigQueryDataByBatches(
		ctx,
		customerID,
		datasetName,
		deleteBatchesReq,
		deletedBy,
	); err != nil {
		l.Errorf("failed to delete customer BQ data by batches with error: %s", err)
		return err
	}

	if err := s.datahubBatchesDAL.DeleteBatches(
		ctx,
		customerID,
		datasetName,
		deleteBatchesReq.Batches,
	); err != nil {
		l.Errorf("failed to delete customer cached summary items with error: %s", err)
	}

	return nil
}

func (s *Service) DeleteCustomerDataHard(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)

	if err := s.bqDAL.DeleteBigQueryDataHard(ctx, customerID, softDeleteIntervalDays); err != nil {
		l.Errorf("failed to delete customer BQ data with error: %s", err)
		return err
	}

	return nil
}

func (s *Service) GetCustomerDatasets(
	ctx context.Context,
	customerID string,
	forceRefresh bool,
) (*domain.CachedDatasetsRes, error) {
	l := s.loggerProvider(ctx)

	customer, err := s.customerDAL.GetCustomerOrPresentationModeCustomer(ctx, customerID)
	if err != nil {
		l.Errorf("failed to get customer or persentation customer with error: %s", err)
		return nil, err
	}

	datasetsMetadata, err := s.datahubDatasetDAL.List(ctx, customer.ID)
	if err != nil {
		l.Errorf("failed to get customer datasets metadata with error: %s", err)
		return nil, err
	}

	if !forceRefresh {
		cachedDatasetsRes, err := s.datahubCachedDatasetDAL.Get(ctx, customer.ID)
		if err != nil && !errors.Is(err, doitFirestore.ErrNotFound) {
			return nil, err
		}

		if (err == nil || errors.Is(err, doitFirestore.ErrNotFound)) &&
			cachedDatasetsRes != nil &&
			s.isCacheValid(cachedDatasetsRes.CachedAt) {
			cachedDatasetsRes.Items = enrichSummaryListItems(cachedDatasetsRes.Items, datasetsMetadata)
			return cachedDatasetsRes, nil
		}
	}

	datasetSummaryItems, err := s.bqDAL.GetCustomerDatasets(
		ctx,
		customer.ID,
	)
	if err != nil {
		l.Errorf("failed to get customer BQ data summary with error: %s", err)
		return nil, err
	}

	cachedDatasetsRes := domain.CachedDatasetsRes{
		Items:    datasetSummaryItems,
		CachedAt: s.timeNow(),
	}

	if err := s.datahubCachedDatasetDAL.Update(ctx, customer.ID, &cachedDatasetsRes); err != nil {
		l.Errorf("error during saving datahub summary %s", err)
	}

	cachedDatasetsRes.Items = enrichSummaryListItems(cachedDatasetsRes.Items, datasetsMetadata)

	return &cachedDatasetsRes, nil
}

func (s *Service) GetCustomerDatasetBatches(
	ctx context.Context,
	customerID string,
	datasetName string,
	forceRefresh bool,
) (*domain.DatasetBatchesRes, error) {
	l := s.loggerProvider(ctx)

	if !forceRefresh {
		datasetBatchesRes, err := s.datahubBatchesDAL.Get(ctx, customerID, datasetName)
		if err != nil && !errors.Is(err, doitFirestore.ErrNotFound) {
			return nil, err
		}

		if (err == nil || errors.Is(err, doitFirestore.ErrNotFound)) && datasetBatchesRes != nil && s.isCacheValid(datasetBatchesRes.CachedAt) {
			return datasetBatchesRes, nil
		}
	}

	batches, err := s.bqDAL.GetCustomerDatasetBatches(ctx, customerID, datasetName)
	if err != nil {
		l.Errorf("failed to get customer dataset batches with error: %s", err)
		return nil, err
	}

	datasetBatchesRes := domain.DatasetBatchesRes{
		Items:    batches,
		CachedAt: s.timeNow(),
	}

	if err := s.datahubBatchesDAL.Update(ctx, customerID, datasetName, &datasetBatchesRes); err != nil {
		l.Errorf("error during saving datahub batches %s", err)
	}

	return &datasetBatchesRes, nil
}

func (s *Service) AddRawEvents(
	ctx context.Context,
	customerID string,
	email string,
	rawEventsReq domain.RawEventsReq,
) ([]*domain.Event, []errormsg.ErrorMsg, error) {
	l := s.loggerProvider(ctx)

	rawSchema := rawEventsReq.Schema
	source := rawEventsReq.Source
	filename := rawEventsReq.Filename
	dataset := rawEventsReq.Dataset
	rawEvents := rawEventsReq.RawEvents
	execute := rawEventsReq.Execute

	schema, validationErrs := domain.NewSchema(rawSchema)
	if validationErrs != nil {
		return nil, validationErrs, nil
	}

	events, validationErrs := domain.NewEventsFromRawEvents(
		dataset,
		schema,
		rawEvents,
	)
	if validationErrs != nil {
		return nil, validationErrs, nil
	}

	req := domain.IngestEventsInternalReq{
		CustomerID: customerID,
		Email:      email,
		Source:     source,
		Execute:    execute,
		FileName:   filename,
		Events:     events,
	}

	_, err := s.datahubInternalAPIDAL.IngestEvents(ctx, req)
	if err != nil {
		var webErr httpDoit.WebError

		// do not expose other errors to the client, unless it's a validation error
		if errors.As(err, &webErr) && webErr.Code == http.StatusBadRequest {
			var internalErrRes domain.InternalErrRes

			err := json.Unmarshal([]byte(webErr.Err.Error()), &internalErrRes)
			if err != nil {
				l.Warningf("Error unmarshalling response JSON: %v", err)
				return nil, nil, errors.New("internal datahub api error")
			}

			return nil, internalErrRes.Errors, err
		}

		l.Errorf("error occurred when calling datahubInternalApiDAL %s", webErr)

		return nil, nil, domain.ErrInternalDatahub
	}

	return events, nil, nil
}

func (s *Service) CreateDataset(
	ctx context.Context,
	customerID string,
	email string,
	datasetReq domain.CreateDatasetRequest,
) error {
	l := s.loggerProvider(ctx)

	dataset := domain.DatasetMetadata{
		Name:        datasetReq.Name,
		Description: datasetReq.Description,
		CreatedBy:   email,
		CreatedAt:   time.Now().Truncate(time.Minute),
	}

	if err := s.datahubDatasetDAL.Create(ctx, customerID, dataset); err != nil {
		l.Errorf("failed to create dataset metadata with error: %s", err)
		return err
	}

	return nil
}

func (s *Service) isCacheValid(cachedAt time.Time) bool {
	return s.timeNow().Sub(cachedAt) < cacheValidityDuration
}

func enrichSummaryListItems(
	summaryListItems []domain.CachedDataset,
	datasetsMetadata []domain.DatasetMetadata,
) []domain.CachedDataset {
	summaryListItemsMap := make(map[string]int, len(datasetsMetadata))
	for i, listItem := range summaryListItems {
		summaryListItemsMap[listItem.Dataset] = i
	}

	for _, datasetMetadata := range datasetsMetadata {
		if pos, ok := summaryListItemsMap[datasetMetadata.Name]; ok {
			summaryListItems[pos].Description = datasetMetadata.Description
		} else {
			summaryListItems = append(summaryListItems, domain.CachedDataset{
				Dataset:     datasetMetadata.Name,
				UpdatedBy:   datasetMetadata.CreatedBy,
				LastUpdated: datasetMetadata.CreatedAt,
				Description: datasetMetadata.Description,
			})
		}
	}

	return summaryListItems
}

func isDatasetProcessing(allDatasets []domain.CachedDataset, deleteDatasets []string) bool {
	for _, datasetItem := range allDatasets {
		if !slice.Contains(deleteDatasets, datasetItem.Dataset) {
			continue
		}

		if now.Sub(datasetItem.LastUpdated) <= processingDuration {
			return true
		}
	}

	return false
}

func isBatchProcessing(allBatches []domain.DatasetBatch, deleteBatches []string, now time.Time) bool {
	for _, batchItem := range allBatches {
		if !slice.Contains(deleteBatches, batchItem.Batch) {
			continue
		}

		if now.Sub(batchItem.SubmittedAt) <= processingDuration {
			return true
		}
	}

	return false
}

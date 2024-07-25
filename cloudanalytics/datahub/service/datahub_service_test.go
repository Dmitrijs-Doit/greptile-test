package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/cloudtasks/mocks"
	doitFirestore "github.com/doitintl/firestore"
	datahubDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	datahubMetricDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric/mocks"
	metadataDalMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	httpDoit "github.com/doitintl/http"
)

func TestDataHubService_DeleteCustomerData(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider     logger.Provider
		datahubMetadataDAL *metadataDalMocks.DataHubMetadataFirestore
		datahubMetricDAL   *datahubMetricDalMocks.DataHubMetricFirestore
		bqDAL              *datahubDalMocks.DataHubBigQuery
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	const (
		customerID = "12345"
	)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful delete all events",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"DeleteBigQueryData",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(nil).
					Once()
				f.datahubMetadataDAL.On(
					"DeleteCustomerMetadata",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(nil).
					Once()
				f.datahubMetricDAL.On(
					"Delete",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:     logger.FromContext,
				datahubMetadataDAL: metadataDalMocks.NewDataHubMetadataFirestore(t),
				datahubMetricDAL:   datahubMetricDalMocks.NewDataHubMetricFirestore(t),
				bqDAL:              datahubDalMocks.NewDataHubBigQuery(t),
			}

			s := &Service{
				loggerProvider:     tt.fields.loggerProvider,
				datahubMetadataDAL: tt.fields.datahubMetadataDAL,
				datahubMetricDAL:   tt.fields.datahubMetricDAL,
				bqDAL:              tt.fields.bqDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteCustomerData(
				ctx,
				tt.args.customerID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.DeleteCustomerData() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.DeleteCustomerData() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_DeleteCustomerDataByEventIDs(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider     logger.Provider
		datahubMetadataDAL *metadataDalMocks.DataHubMetadataFirestore
		datahubMetricDAL   *datahubMetricDalMocks.DataHubMetricFirestore
		bqDAL              *datahubDalMocks.DataHubBigQuery
	}

	type args struct {
		ctx           context.Context
		customerID    string
		deleteRequest domain.DeleteEventsReq
	}

	const (
		customerID = "12345"
	)

	events := []string{"event1", "event2"}

	deletedBy := "test@doit.com"

	deleteEventsReq := domain.DeleteEventsReq{
		EventIDs: events,
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful delete events",
			args: args{
				ctx:           ctx,
				customerID:    customerID,
				deleteRequest: deleteEventsReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"DeleteBigQueryDataByEventIDs",
					testutils.ContextBackgroundMock,
					customerID,
					deleteEventsReq,
					deletedBy).
					Return(nil).
					Once()
			},
		},
		{
			name: "error on delete empty events",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				deleteRequest: domain.DeleteEventsReq{
					EventIDs: nil,
				},
			},
			wantErr:     true,
			expectedErr: domain.ErrEventsIDsCanNotBeEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:     logger.FromContext,
				datahubMetadataDAL: metadataDalMocks.NewDataHubMetadataFirestore(t),
				datahubMetricDAL:   datahubMetricDalMocks.NewDataHubMetricFirestore(t),
				bqDAL:              datahubDalMocks.NewDataHubBigQuery(t),
			}

			s := &Service{
				loggerProvider:     tt.fields.loggerProvider,
				datahubMetadataDAL: tt.fields.datahubMetadataDAL,
				datahubMetricDAL:   tt.fields.datahubMetricDAL,
				bqDAL:              tt.fields.bqDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteCustomerDataByEventIDs(
				ctx,
				tt.args.customerID,
				tt.args.deleteRequest,
				deletedBy,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.DeleteCustomerDataByEventIDs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.DeleteCustomerDataByEventIDs() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_DeleteCustomerDataByClouds(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider          logger.Provider
		datahubMetadataDAL      *metadataDalMocks.DataHubMetadataFirestore
		datahubMetricDAL        *datahubMetricDalMocks.DataHubMetricFirestore
		bqDAL                   *datahubDalMocks.DataHubBigQuery
		datahubDatasetDAL       *datahubDalMocks.DataHubDatasetFirestore
		datahubCachedDatasetDAL *datahubDalMocks.DataHubCachedDatasetFirestore
	}

	type args struct {
		ctx           context.Context
		customerID    string
		deleteRequest domain.DeleteDatasetsReq
	}

	const (
		customerID = "12345"
	)

	var (
		mockTime         = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)
		lastUpdatedTime  = time.Date(2021, 03, 04, 12, 0, 0, 0, time.UTC)
		lastUpdatedTime2 = time.Date(2021, 04, 04, 12, 0, 0, 0, time.UTC)
	)

	deletedBy := "test@doit.com"

	clouds := []string{"datadog", "openai"}

	deleteDatasetReq := domain.DeleteDatasetsReq{
		Datasets: clouds,
	}

	datasetSummaryItems := []domain.CachedDataset{
		{
			Dataset:     clouds[0],
			UpdatedBy:   "user@doit.com",
			Records:     12,
			LastUpdated: lastUpdatedTime,
		},
		{
			Dataset:     clouds[1],
			UpdatedBy:   "user2@doit.com",
			Records:     15,
			LastUpdated: lastUpdatedTime2,
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful delete events by cloud",
			args: args{
				ctx:           ctx,
				customerID:    customerID,
				deleteRequest: deleteDatasetReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomerDatasets",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(datasetSummaryItems, nil).
					Once()
				f.datahubCachedDatasetDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					&domain.CachedDatasetsRes{
						Items:    datasetSummaryItems,
						CachedAt: mockTime,
					},
				).
					Return(nil).
					Once()
				f.bqDAL.On(
					"DeleteBigQueryDataByClouds",
					testutils.ContextBackgroundMock,
					customerID,
					deleteDatasetReq,
					deletedBy).
					Return(nil).
					Once()
				f.datahubDatasetDAL.On(
					"Delete",
					testutils.ContextBackgroundMock,
					customerID,
					deleteDatasetReq.Datasets).
					Return(nil).Once()
				f.datahubCachedDatasetDAL.On(
					"DeleteItems",
					testutils.ContextBackgroundMock,
					customerID,
					deleteDatasetReq.Datasets).
					Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:          logger.FromContext,
				datahubMetadataDAL:      metadataDalMocks.NewDataHubMetadataFirestore(t),
				datahubMetricDAL:        datahubMetricDalMocks.NewDataHubMetricFirestore(t),
				bqDAL:                   datahubDalMocks.NewDataHubBigQuery(t),
				datahubDatasetDAL:       datahubDalMocks.NewDataHubDatasetFirestore(t),
				datahubCachedDatasetDAL: datahubDalMocks.NewDataHubCachedDatasetFirestore(t),
			}

			s := &Service{
				loggerProvider:          tt.fields.loggerProvider,
				datahubMetadataDAL:      tt.fields.datahubMetadataDAL,
				datahubMetricDAL:        tt.fields.datahubMetricDAL,
				bqDAL:                   tt.fields.bqDAL,
				datahubDatasetDAL:       tt.fields.datahubDatasetDAL,
				datahubCachedDatasetDAL: tt.fields.datahubCachedDatasetDAL,
				timeNow:                 func() time.Time { return mockTime },
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteCustomerDataByClouds(
				ctx,
				tt.args.customerID,
				tt.args.deleteRequest,
				deletedBy,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.DeleteCustomerDataByClouds error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.DeleteCustomerDataByClouds error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_DeleteDatasetBatches(t *testing.T) {
	ctx := context.Background()

	mockTime := time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)

	type fields struct {
		loggerProvider    logger.Provider
		bqDAL             *datahubDalMocks.DataHubBigQuery
		datahubBatchesDAL *datahubDalMocks.DataHubBatchesFirestore
	}

	type args struct {
		ctx              context.Context
		customerID       string
		datasetName      string
		deleteBatchesReq domain.DeleteBatchesReq
		deletedBy        string
	}

	const (
		customerID  = "12345"
		datasetName = "some-dataset-name"
		deletedBy   = "test@doit.com"
	)

	deleteBatchesReq := domain.DeleteBatchesReq{
		Batches: []string{"batch1", "batch2"},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful delete of dataset batches",
			args: args{
				ctx:              ctx,
				customerID:       customerID,
				datasetName:      datasetName,
				deleteBatchesReq: deleteBatchesReq,
				deletedBy:        deletedBy,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&domain.DatasetBatchesRes{
						CachedAt: mockTime,
					},
				).
					Return(nil).
					Once()
				f.bqDAL.On(
					"DeleteBigQueryDataByBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					deleteBatchesReq,
					deletedBy,
				).
					Return(nil).
					Once()
				f.datahubBatchesDAL.On(
					"DeleteBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					[]string{"batch1", "batch2"},
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "error no batches to delete",
			args: args{
				ctx:         ctx,
				customerID:  customerID,
				datasetName: datasetName,
				deletedBy:   deletedBy,
			},
			wantErr:     true,
			expectedErr: domain.ErrBatchesCanNotBeEmpty,
		},
		{
			name: "error on delete from BQ",
			args: args{
				ctx:              ctx,
				customerID:       customerID,
				datasetName:      datasetName,
				deleteBatchesReq: deleteBatchesReq,
				deletedBy:        deletedBy,
			},
			wantErr: true,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&domain.DatasetBatchesRes{
						CachedAt: mockTime,
					},
				).
					Return(nil).
					Once()
				f.bqDAL.On(
					"DeleteBigQueryDataByBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					deleteBatchesReq,
					deletedBy,
				).
					Return(errors.New("some error")).
					Once()
			},
		},
		{
			name: "error on delete from cache, but no reason to fail",
			args: args{
				ctx:              ctx,
				customerID:       customerID,
				datasetName:      datasetName,
				deleteBatchesReq: deleteBatchesReq,
				deletedBy:        deletedBy,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&domain.DatasetBatchesRes{
						CachedAt: mockTime,
					},
				).
					Return(nil).
					Once()
				f.bqDAL.On(
					"DeleteBigQueryDataByBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					deleteBatchesReq,
					deletedBy,
				).
					Return(nil).
					Once()
				f.datahubBatchesDAL.On(
					"DeleteBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					[]string{"batch1", "batch2"},
				).
					Return(errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:    logger.FromContext,
				bqDAL:             datahubDalMocks.NewDataHubBigQuery(t),
				datahubBatchesDAL: datahubDalMocks.NewDataHubBatchesFirestore(t),
			}

			s := &Service{
				loggerProvider:    tt.fields.loggerProvider,
				bqDAL:             tt.fields.bqDAL,
				datahubBatchesDAL: tt.fields.datahubBatchesDAL,
				timeNow:           func() time.Time { return mockTime },
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteDatasetBatches(
				ctx,
				tt.args.customerID,
				tt.args.datasetName,
				tt.args.deleteBatchesReq,
				tt.args.deletedBy,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.DeleteDatasetBatches() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.DeleteDatasetBatches() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_DeleteAllCustomersDataHard(t *testing.T) {
	ctx := context.Background()

	const (
		customer1 = "customer 1"
		customer2 = "customer 2"
	)

	type fields struct {
		loggerProvider  logger.Provider
		bqDAL           *datahubDalMocks.DataHubBigQuery
		cloudTaskClient *mocks.CloudTaskClient
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	const (
		customerID = "12345"
	)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful create tasks for all customers who have soft-deleted data",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomersWithSoftDeleteData",
					testutils.ContextBackgroundMock,
					softDeleteIntervalDays,
				).
					Return([]string{customer1, customer2}, nil).Once()
				f.cloudTaskClient.On(
					"CreateTask",
					testutils.ContextBackgroundMock,
					mock.AnythingOfType("*iface.Config"),
				).
					Return(nil, nil).
					Twice()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:  logger.FromContext,
				bqDAL:           datahubDalMocks.NewDataHubBigQuery(t),
				cloudTaskClient: mocks.NewCloudTaskClient(t),
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				loggerProvider:  tt.fields.loggerProvider,
				bqDAL:           tt.fields.bqDAL,
				cloudTaskClient: tt.fields.cloudTaskClient,
			}

			err := s.DeleteAllCustomersDataHard(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.DeleteAllCustomersDataHard() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.DeleteAllCustomersDataHard() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_DeleteCustomerDataHard(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider logger.Provider
		bqDAL          *datahubDalMocks.DataHubBigQuery
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	const (
		customerID = "12345"
	)

	expectedError := errors.New("failed to delete customer data hard")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful delete customer data hard",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"DeleteBigQueryDataHard",
					testutils.ContextBackgroundMock,
					customerID,
					softDeleteIntervalDays,
				).
					Return(nil).
					Once()
			},
		},
		{
			name: "error on delete customer data hard",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr:     true,
			expectedErr: expectedError,
			on: func(f *fields) {
				f.bqDAL.On(
					"DeleteBigQueryDataHard",
					testutils.ContextBackgroundMock,
					customerID,
					softDeleteIntervalDays,
				).
					Return(expectedError).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				bqDAL:          datahubDalMocks.NewDataHubBigQuery(t),
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				loggerProvider: tt.fields.loggerProvider,
				bqDAL:          tt.fields.bqDAL,
			}

			err := s.DeleteCustomerDataHard(ctx, tt.args.customerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.DeleteCustomerDataHard() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.DeleteCustomerDataHard() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_GetCustomerDataSummary(t *testing.T) {
	ctx := context.Background()

	var (
		mockTime                 = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)
		mockTimeShouldNotRefresh = mockTime.Add(-cacheValidityDuration + 1*time.Minute)
		mockTimeShouldRefresh    = mockTime.Add(-cacheValidityDuration - 1*time.Minute)
	)

	type fields struct {
		loggerProvider          logger.Provider
		bqDAL                   *datahubDalMocks.DataHubBigQuery
		datahubCachedDatasetDAL *datahubDalMocks.DataHubCachedDatasetFirestore
		datahubDatasetDAL       *datahubDalMocks.DataHubDatasetFirestore
		customerDAL             *customerDalMock.Customers
	}

	type args struct {
		ctx          context.Context
		customerID   string
		forceRefresh bool
	}

	const (
		customerID = "12345"
	)

	var lastUpdatedTime = time.Date(2021, 03, 04, 12, 0, 0, 0, time.UTC)
	var lastUpdatedTime2 = time.Date(2021, 04, 04, 12, 0, 0, 0, time.UTC)

	datasetSummaryItems := []domain.CachedDataset{
		{
			Dataset:     "newrelic",
			UpdatedBy:   "user@doit.com",
			Records:     12,
			LastUpdated: lastUpdatedTime,
		},
		{
			Dataset:     "datadog",
			UpdatedBy:   "user2@doit.com",
			Records:     15,
			LastUpdated: lastUpdatedTime2,
		},
	}

	datasetsMetadata := []domain.DatasetMetadata{
		{
			Name:        "test UI created dataset",
			Description: "This is a dataset created by the UI",
			CreatedBy:   "user@doit.com",
			CreatedAt:   lastUpdatedTime,
		},
	}

	enrichedSummaryItems := []domain.CachedDataset{
		{
			Dataset:     "newrelic",
			UpdatedBy:   "user@doit.com",
			Records:     12,
			LastUpdated: lastUpdatedTime,
		},
		{
			Dataset:     "datadog",
			UpdatedBy:   "user2@doit.com",
			Records:     15,
			LastUpdated: lastUpdatedTime2,
		},
		{
			Dataset:     datasetsMetadata[0].Name,
			UpdatedBy:   datasetsMetadata[0].CreatedBy,
			Records:     0,
			LastUpdated: datasetsMetadata[0].CreatedAt,
			Description: datasetsMetadata[0].Description,
		},
	}
	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: customerID,
			},
		},
		ID: customerID,
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		expectedRes *domain.CachedDatasetsRes
		on          func(*fields)
	}{
		{
			name: "successful get from bq and store in firestore, no cache used, with enriched data",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On(
					"GetCustomerOrPresentationModeCustomer",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(customer, nil)
				f.datahubDatasetDAL.On(
					"List",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(datasetsMetadata, nil).
					Once()
				f.datahubCachedDatasetDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(nil, nil).
					Once()
				f.bqDAL.On(
					"GetCustomerDatasets",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(datasetSummaryItems, nil).
					Once()
				f.datahubCachedDatasetDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					&domain.CachedDatasetsRes{
						Items:    datasetSummaryItems,
						CachedAt: mockTime,
					},
				).
					Return(nil).
					Once()
			},
			expectedRes: &domain.CachedDatasetsRes{
				Items:    enrichedSummaryItems,
				CachedAt: mockTime,
			},
		},
		{
			name: "successful hit the cache + enriched data",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On(
					"GetCustomerOrPresentationModeCustomer",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(customer, nil)
				f.datahubDatasetDAL.On(
					"List",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(datasetsMetadata, nil).
					Once()
				f.datahubCachedDatasetDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(&domain.CachedDatasetsRes{
						Items:    datasetSummaryItems,
						CachedAt: mockTimeShouldNotRefresh,
					}, nil).
					Once()
			},
			expectedRes: &domain.CachedDatasetsRes{
				Items:    enrichedSummaryItems,
				CachedAt: mockTimeShouldNotRefresh,
			},
		},
		{
			name: "successful get from bq and store in firestore, no cache used because data is expired",
			args: args{
				ctx:        ctx,
				customerID: customerID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On(
					"GetCustomerOrPresentationModeCustomer",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(customer, nil)
				f.datahubDatasetDAL.On(
					"List",
					testutils.ContextBackgroundMock,
					customerID,
				).Return(nil, nil).
					Once()
				f.datahubCachedDatasetDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(&domain.CachedDatasetsRes{
						Items:    datasetSummaryItems,
						CachedAt: mockTimeShouldRefresh,
					}, nil).
					Once()
				f.bqDAL.On(
					"GetCustomerDatasets",
					testutils.ContextBackgroundMock,
					customerID,
				).
					Return(datasetSummaryItems, nil).
					Once()
				f.datahubCachedDatasetDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					&domain.CachedDatasetsRes{
						Items:    datasetSummaryItems,
						CachedAt: mockTime,
					},
				).
					Return(nil).
					Once()
			},
			expectedRes: &domain.CachedDatasetsRes{
				Items:    datasetSummaryItems,
				CachedAt: mockTime,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:          logger.FromContext,
				bqDAL:                   datahubDalMocks.NewDataHubBigQuery(t),
				datahubCachedDatasetDAL: datahubDalMocks.NewDataHubCachedDatasetFirestore(t),
				datahubDatasetDAL:       datahubDalMocks.NewDataHubDatasetFirestore(t),
				customerDAL:             customerDalMock.NewCustomers(t),
			}

			s := &Service{
				loggerProvider:          tt.fields.loggerProvider,
				bqDAL:                   tt.fields.bqDAL,
				datahubCachedDatasetDAL: tt.fields.datahubCachedDatasetDAL,
				datahubDatasetDAL:       tt.fields.datahubDatasetDAL,
				customerDAL:             tt.fields.customerDAL,
				timeNow:                 func() time.Time { return mockTime },
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res, err := s.GetCustomerDatasets(
				ctx,
				tt.args.customerID,
				tt.args.forceRefresh,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.GetCustomerDatasets() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.GetCustomerDatasets() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if err == nil && !tt.wantErr {
				assert.Equal(t, res, tt.expectedRes)
			}
		})
	}
}

func TestDataHubService_GetCustomerDatasetBatches(t *testing.T) {
	ctx := context.Background()

	var (
		mockTime                 = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)
		mockTimeShouldNotRefresh = mockTime.Add(-cacheValidityDuration + 1*time.Minute)
		mockTimeShouldRefresh    = mockTime.Add(-cacheValidityDuration - 1*time.Minute)
	)

	type fields struct {
		loggerProvider          logger.Provider
		bqDAL                   *datahubDalMocks.DataHubBigQuery
		datahubCachedDatasetDAL *datahubDalMocks.DataHubCachedDatasetFirestore
		datahubDatasetDAL       *datahubDalMocks.DataHubDatasetFirestore
		datahubBatchesDAL       *datahubDalMocks.DataHubBatchesFirestore
	}

	type args struct {
		ctx          context.Context
		customerID   string
		datasetName  string
		forceRefresh bool
	}

	customerID := "some-customer-id"
	datasetName := "openAI"

	batches := []domain.DatasetBatch{
		{
			Batch:       "openai-jan-2022-1",
			Origin:      "csv",
			Records:     100,
			SubmittedBy: customerID,
			SubmittedAt: mockTime,
		},
		{
			Batch:       "openai-jan-2022-2",
			Origin:      "csv",
			Records:     105,
			SubmittedBy: customerID,
			SubmittedAt: mockTime,
		},
	}

	datasetBatchesRes := domain.DatasetBatchesRes{
		Items:    batches,
		CachedAt: mockTime,
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		expectedRes *domain.DatasetBatchesRes
		on          func(*fields)
	}{
		{
			name: "successful get from bq and store in fs, no cache used",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: true,
			},
			wantErr: false,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(batches, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&datasetBatchesRes,
				).
					Return(nil).
					Once()
			},
			expectedRes: &datasetBatchesRes,
		},
		{
			name: "successful cache hit",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: false,
			},
			wantErr: false,
			on: func(f *fields) {
				f.datahubBatchesDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(&domain.DatasetBatchesRes{
						Items:    batches,
						CachedAt: mockTimeShouldNotRefresh,
					}, nil).
					Once()
			},
			expectedRes: &domain.DatasetBatchesRes{
				Items:    batches,
				CachedAt: mockTimeShouldNotRefresh,
			},
		},
		{
			name: "successful get from bq and store in fs, no cache used because data is expired",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: false,
			},
			wantErr: false,
			on: func(f *fields) {
				f.datahubBatchesDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(&domain.DatasetBatchesRes{
						Items:    batches,
						CachedAt: mockTimeShouldRefresh,
					}, nil).
					Once()
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(batches, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&datasetBatchesRes,
				).
					Return(nil).
					Once()
			},
			expectedRes: &domain.DatasetBatchesRes{
				Items:    batches,
				CachedAt: mockTime,
			},
		},
		{
			name: "successful get from bq and store in fs, no cache used because saved batch in fs is nil",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: false,
			},
			wantErr: false,
			on: func(f *fields) {
				f.datahubBatchesDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, nil).
					Once()
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(batches, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&datasetBatchesRes,
				).
					Return(nil).
					Once()
			},
			expectedRes: &datasetBatchesRes,
		},
		{
			name: "successful get from bq, no cache used because cache was not found in fs",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: false,
			},
			wantErr: false,
			on: func(f *fields) {
				f.datahubBatchesDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, doitFirestore.ErrNotFound).
					Once()
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(batches, nil).
					Once()
				f.datahubBatchesDAL.On(
					"Update",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
					&datasetBatchesRes,
				).
					Return(nil).
					Once()
			},
			expectedRes: &datasetBatchesRes,
		},
		{
			name: "error while getting cached batches from fs",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: false,
			},
			wantErr: true,
			on: func(f *fields) {
				f.datahubBatchesDAL.On(
					"Get",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
		{
			name: "error while getting batches from bq",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: true,
			},
			wantErr: true,
			on: func(f *fields) {
				f.bqDAL.On(
					"GetCustomerDatasetBatches",
					testutils.ContextBackgroundMock,
					customerID,
					datasetName,
				).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:          logger.FromContext,
				bqDAL:                   datahubDalMocks.NewDataHubBigQuery(t),
				datahubCachedDatasetDAL: datahubDalMocks.NewDataHubCachedDatasetFirestore(t),
				datahubDatasetDAL:       datahubDalMocks.NewDataHubDatasetFirestore(t),
				datahubBatchesDAL:       datahubDalMocks.NewDataHubBatchesFirestore(t),
			}

			s := &Service{
				loggerProvider:          tt.fields.loggerProvider,
				bqDAL:                   tt.fields.bqDAL,
				datahubCachedDatasetDAL: tt.fields.datahubCachedDatasetDAL,
				datahubDatasetDAL:       tt.fields.datahubDatasetDAL,
				datahubBatchesDAL:       tt.fields.datahubBatchesDAL,
				timeNow:                 func() time.Time { return mockTime },
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res, err := s.GetCustomerDatasetBatches(
				ctx,
				tt.args.customerID,
				tt.args.datasetName,
				tt.args.forceRefresh,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.GetCustomerDatasetBatches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.GetCustomerDatasetBatches() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if err == nil && !tt.wantErr {
				assert.Equal(t, res, tt.expectedRes)
			}
		})
	}

}

func TestDataHubService_enrichSummaryListItems(t *testing.T) {
	var lastUpdatedTime = time.Date(2021, 03, 04, 12, 0, 0, 0, time.UTC)
	var lastUpdatedTime2 = time.Date(2021, 04, 04, 12, 0, 0, 0, time.UTC)

	summaryListItems := []domain.CachedDataset{
		{
			Dataset:     "newrelic",
			UpdatedBy:   "user@doit.com",
			Records:     12,
			LastUpdated: lastUpdatedTime,
		},
		{
			Dataset:     "datadog",
			UpdatedBy:   "user2@doit.com",
			Records:     15,
			LastUpdated: lastUpdatedTime2,
		},
	}

	datasetsMetadata := []domain.DatasetMetadata{
		{
			Name:        "test UI created dataset",
			Description: "This is a dataset created by the UI",
			CreatedBy:   "user@doit.com",
			CreatedAt:   lastUpdatedTime,
		},
	}

	datasetsOverlappingMetadata := []domain.DatasetMetadata{
		{
			Name:        "newrelic",
			Description: "This is a dataset created by the UI",
			CreatedBy:   "user@doit.com",
			CreatedAt:   lastUpdatedTime,
		},
	}

	enrichedItems := append(summaryListItems, domain.CachedDataset{
		Dataset:     datasetsMetadata[0].Name,
		UpdatedBy:   datasetsMetadata[0].CreatedBy,
		Records:     0,
		LastUpdated: datasetsMetadata[0].CreatedAt,
		Description: datasetsMetadata[0].Description,
	})

	tests := []struct {
		name             string
		summaryListItems []domain.CachedDataset
		datasetsMetadata []domain.DatasetMetadata
		want             []domain.CachedDataset
	}{
		{
			name: "overlap of metadata and summary",
			summaryListItems: []domain.CachedDataset{
				{
					Dataset:     "newrelic",
					UpdatedBy:   "user@doit.com",
					Records:     12,
					LastUpdated: lastUpdatedTime,
				}, {
					Dataset:     "datadog",
					UpdatedBy:   "user2@doit.com",
					Records:     15,
					LastUpdated: lastUpdatedTime2,
				},
			},
			datasetsMetadata: []domain.DatasetMetadata{
				{
					Name:        "datadog",
					Description: "This is a datadog description",
					CreatedBy:   "user@doit.com",
					CreatedAt:   lastUpdatedTime,
				},
				{
					Name:        "test UI created dataset",
					Description: "This is a dataset created by the UI",
					CreatedBy:   "user@doit.com",
					CreatedAt:   lastUpdatedTime,
				},
			},
			want: []domain.CachedDataset{
				{
					Dataset:     "newrelic",
					UpdatedBy:   "user@doit.com",
					Records:     12,
					LastUpdated: lastUpdatedTime,
				}, {
					Dataset:     "datadog",
					UpdatedBy:   "user2@doit.com",
					Records:     15,
					LastUpdated: lastUpdatedTime2,
					Description: "This is a datadog description",
				},
				{
					Dataset:     "test UI created dataset",
					Description: "This is a dataset created by the UI",
					UpdatedBy:   "user@doit.com",
					LastUpdated: lastUpdatedTime,
				},
			},
		},
		{
			name:             "add enriched item to summaryListItems",
			summaryListItems: summaryListItems,
			datasetsMetadata: datasetsMetadata,
			want:             enrichedItems,
		},
		{
			name:             "no results from datasetsMetadata, return summaryListItems as is",
			summaryListItems: summaryListItems,
			datasetsMetadata: nil,
			want:             summaryListItems,
		},
		{
			name:             "overlapping dataset names, return summaryListItems with updated description",
			summaryListItems: summaryListItems,
			datasetsMetadata: datasetsOverlappingMetadata,
			want: []domain.CachedDataset{
				{
					Dataset:     "newrelic",
					UpdatedBy:   "user@doit.com",
					Records:     12,
					LastUpdated: lastUpdatedTime,
					Description: "This is a dataset created by the UI",
				},
				{
					Dataset:     "datadog",
					UpdatedBy:   "user2@doit.com",
					Records:     15,
					LastUpdated: lastUpdatedTime2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enrichSummaryListItems(tt.summaryListItems, tt.datasetsMetadata)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDataHubService_CreateDataset(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider    logger.Provider
		datahubDatasetDAL *datahubDalMocks.DataHubDatasetFirestore
	}

	type args struct {
		ctx        context.Context
		customerID string
		dataset    domain.DatasetMetadata
	}

	const (
		customerID = "12345"
		email      = "user@doit.com"
	)

	errFailed := errors.New("failed to create dataset")

	datasetReq := domain.CreateDatasetRequest{
		Name:        "test UI created dataset",
		Description: "This is a dataset created by the UI",
	}

	dataset := domain.DatasetMetadata{
		Name:        datasetReq.Name,
		Description: datasetReq.Description,
		CreatedBy:   email,
		CreatedAt:   time.Now().Truncate(time.Minute),
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successful dataset creation",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				dataset:    dataset,
			},
			wantErr: false,
			on: func(f *fields) {
				f.datahubDatasetDAL.On(
					"Create",
					testutils.ContextBackgroundMock,
					customerID,
					dataset,
				).Return(nil).Once()
			},
		},
		{
			name: "error on dataset creation",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				dataset:    dataset,
			},
			wantErr:     true,
			expectedErr: errFailed,
			on: func(f *fields) {
				f.datahubDatasetDAL.On(
					"Create",
					testutils.ContextBackgroundMock,
					customerID,
					dataset,
				).Return(errFailed).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:    logger.FromContext,
				datahubDatasetDAL: datahubDalMocks.NewDataHubDatasetFirestore(t),
			}

			s := &Service{
				loggerProvider:    tt.fields.loggerProvider,
				datahubDatasetDAL: tt.fields.datahubDatasetDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.CreateDataset(
				ctx,
				tt.args.customerID,
				email,
				datasetReq,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.CreateDataset() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.CreateDataset() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestDataHubService_AddRawEvents(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider        logger.Provider
		datahubInternalAPIDAL *datahubDalMocks.DatahubInternalAPIDAL
	}

	type args struct {
		ctx          context.Context
		customerID   string
		email        string
		rawEventsReq domain.RawEventsReq
	}

	customerID := "123"
	email := "test@doit.com"
	filename := "myfile.csv"

	dataset := "datadog"
	sourceCsv := "csv"

	id1 := "73f8b9da-ebb2-4046-8226-1015ee94b499"
	id2 := "73f8b9da-ebb2-4046-8226-1015ee94b222"

	rawEvents := [][]string{
		{
			"2024-03-01T00:00:00Z", id1, "pr1", "adoption", "12",
		},
		{
			"2024-04-01T00:00:00Z", id2, "pr2", "another_house", "15",
		},
	}

	schema := []string{"usage_date", "event_id", "project_id", "label.house", "metric.cost"}

	internalErrRes := domain.InternalErrRes{
		Errors: []errormsg.ErrorMsg{
			{
				Field:   "field1",
				Message: "field1 value is not valid",
			},
		},
	}

	resJSON, _ := json.Marshal(internalErrRes)

	webValidationErr := httpDoit.WebError{
		Code: http.StatusBadRequest,
		Err:  errors.New(string(resJSON)),
	}

	rawEventsReq := domain.RawEventsReq{
		Dataset:   dataset,
		Source:    sourceCsv,
		Schema:    schema,
		RawEvents: rawEvents,
		Filename:  filename,
		Execute:   true,
	}

	events := []*domain.Event{
		{
			Cloud: &dataset,
			ID:    &id1,
			Time:  time.Date(2024, 03, 1, 0, 0, 0, 0, time.UTC),
			Dimensions: []*domain.Dimension{
				{
					Type:  domain.SchemaFieldTypeFixed,
					Key:   "project_id",
					Value: "pr1",
				},
				{
					Type:  domain.SchemaFieldTypeLabel,
					Key:   "house",
					Value: "adoption",
				},
			},
			Metrics: []*domain.Metric{
				{
					Type:  "cost",
					Value: 12,
				},
			},
		},
		{
			Cloud: &dataset,
			ID:    &id2,
			Time:  time.Date(2024, 04, 1, 0, 0, 0, 0, time.UTC),
			Dimensions: []*domain.Dimension{
				{
					Type:  domain.SchemaFieldTypeFixed,
					Key:   "project_id",
					Value: "pr2",
				},
				{
					Type:  domain.SchemaFieldTypeLabel,
					Key:   "house",
					Value: "another_house",
				},
			},
			Metrics: []*domain.Metric{
				{
					Type:  "cost",
					Value: 15,
				},
			},
		},
	}
	tests := []struct {
		name                   string
		fields                 fields
		args                   args
		wantErr                bool
		expectedRes            []*domain.Event
		expectedValidationErrs []errormsg.ErrorMsg
		expectedErr            error
		on                     func(*fields)
	}{
		{
			name: "successfully ingest raw data",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				email:        email,
				rawEventsReq: rawEventsReq,
			},
			wantErr:     false,
			expectedRes: events,
			on: func(f *fields) {
				f.datahubInternalAPIDAL.On(
					"IngestEvents",
					testutils.ContextBackgroundMock,
					domain.IngestEventsInternalReq{
						CustomerID: customerID,
						Email:      email,
						Source:     "csv",
						Execute:    true,
						FileName:   filename,
						Events:     events,
					},
				).Return(nil, nil, nil).
					Once()
			},
		},
		{
			name: "successfully invoke 'ingest' endpoint with execute=false",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				rawEventsReq: domain.RawEventsReq{
					Dataset:   dataset,
					Source:    sourceCsv,
					Schema:    schema,
					RawEvents: rawEvents,
					Filename:  filename,
					Execute:   false,
				},
			},
			on: func(f *fields) {
				f.datahubInternalAPIDAL.On(
					"IngestEvents",
					testutils.ContextBackgroundMock,
					domain.IngestEventsInternalReq{
						CustomerID: customerID,
						Email:      email,
						Source:     sourceCsv,
						Execute:    false,
						FileName:   filename,
						Events:     events,
					},
				).Return(nil, nil, nil).
					Once()
			},
			wantErr:     false,
			expectedRes: events,
		},
		{
			name: "schema is no valid, usage_date is missing",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				rawEventsReq: domain.RawEventsReq{
					Dataset:   dataset,
					Source:    sourceCsv,
					Schema:    []string{"event_id", "project_id", "label.house", "metric.cost"},
					RawEvents: rawEvents,
					Filename:  filename,
					Execute:   true,
				},
			},
			wantErr: false,
			expectedValidationErrs: []errormsg.ErrorMsg{
				{
					Field:   "usage_date",
					Message: "field must be provided in schema",
				},
			},
			expectedRes: nil,
		},
		{
			name: "raw data is not valid",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				email:      email,
				rawEventsReq: domain.RawEventsReq{
					Dataset: dataset,
					Source:  sourceCsv,
					Schema:  schema,
					RawEvents: [][]string{
						{"2024-03-01T00:00:00Z", id1, "pr1", "adoption", "12"},
						{"2024-04-01T00:00:00Z", id2, "pr2", "another_house"},
					},
					Filename: filename,
					Execute:  true,
				},
			},
			wantErr: false,
			expectedValidationErrs: []errormsg.ErrorMsg{
				{
					Field:   "row: 2",
					Message: "number of columns does not match the number of header fields",
				},
			},
			expectedRes: nil,
		},
		{
			name: "internal error during ingesting events",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				email:        email,
				rawEventsReq: rawEventsReq,
			},
			wantErr:     true,
			expectedErr: domain.ErrInternalDatahub,
			on: func(f *fields) {
				f.datahubInternalAPIDAL.On(
					"IngestEvents",
					testutils.ContextBackgroundMock,
					domain.IngestEventsInternalReq{
						CustomerID: customerID,
						Email:      email,
						Source:     "csv",
						Execute:    true,
						FileName:   filename,
						Events:     events,
					},
				).Return(nil, errors.New("error ingesting events")).
					Once()
			},
		},
		{
			name: "validation errors from datahub-api should be returned as validation errors",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				email:        email,
				rawEventsReq: rawEventsReq,
			},
			wantErr: false,
			expectedValidationErrs: []errormsg.ErrorMsg{
				{
					Field:   "field1",
					Message: "field1 value is not valid",
				},
			},
			on: func(f *fields) {
				f.datahubInternalAPIDAL.On(
					"IngestEvents",
					testutils.ContextBackgroundMock,
					domain.IngestEventsInternalReq{
						CustomerID: customerID,
						Email:      email,
						Source:     "csv",
						Execute:    true,
						FileName:   filename,
						Events:     events,
					},
				).Return(nil, webValidationErr).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider:        logger.FromContext,
				datahubInternalAPIDAL: datahubDalMocks.NewDatahubInternalAPIDAL(t),
			}

			s := &Service{
				loggerProvider:        tt.fields.loggerProvider,
				datahubInternalAPIDAL: tt.fields.datahubInternalAPIDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res, validationErrs, err := s.AddRawEvents(
				ctx,
				tt.args.customerID,
				tt.args.email,
				tt.args.rawEventsReq,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("DataHubService.AddRawEvents() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("DataHubService.AddRawEvents() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if tt.expectedValidationErrs != nil {
				assert.Equal(t, tt.expectedValidationErrs, validationErrs)
			}

			if err == nil {
				assert.Equal(t, tt.expectedRes, res)
			}
		})
	}
}

func TestDataHubService_isDatasetProcessing(t *testing.T) {
	now := time.Now()

	datasets := []domain.CachedDataset{
		{
			Dataset:     "dataset1",
			LastUpdated: now.Add(-85 * time.Minute), // within the last 90 minutes
		},
		{
			Dataset:     "dataset2",
			LastUpdated: now.Add(-95 * time.Minute), // outside the last 90 minutes
		},
	}

	tests := []struct {
		name           string
		deleteDatasets []string
		want           bool
	}{
		{
			name:           "dataset1 is processing",
			deleteDatasets: []string{"dataset1"},
			want:           true,
		},
		{
			name:           "no datasets are processing",
			deleteDatasets: []string{"dataset2"},
			want:           false,
		},
		{
			name:           "dataset not in delete list",
			deleteDatasets: []string{"dataset3"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDatasetProcessing(datasets, tt.deleteDatasets)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDataHubService_isBatchProcessing(t *testing.T) {
	mockTime := time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)

	batches := []domain.DatasetBatch{
		{
			Batch:       "batch1",
			SubmittedAt: mockTime.Add(-85 * time.Minute), // within the last 90 minutes
		},
		{
			Batch:       "batch2",
			SubmittedAt: mockTime.Add(-96 * time.Minute), // outside the last 90 minutes
		},
	}

	tests := []struct {
		name          string
		deleteBatches []string
		want          bool
	}{
		{
			name:          "batch1 is processing",
			deleteBatches: []string{"batch1"},
			want:          true,
		},
		{
			name:          "no batches are processing",
			deleteBatches: []string{"batch2"},
			want:          false,
		},
		{
			name:          "batch not in delete list",
			deleteBatches: []string{"batch3"},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isBatchProcessing(batches, tt.deleteBatches, mockTime)
			assert.Equal(t, tt.want, got)
		})
	}
}

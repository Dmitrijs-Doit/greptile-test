package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var ginContextMock = mock.AnythingOfType("*gin.Context")

func TestDataHubAPIHandler_DeleteCustomerSpecificEvents(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		body *domain.DeleteEventsReq
	}

	events := []string{"111", "222"}
	clouds := []string{"datadog", "openai"}
	deletedBy := "test@delete.com"

	deleteByEventIDsReq := &domain.DeleteEventsReq{
		EventIDs: events,
	}

	deleteByCloudsReq := &domain.DeleteEventsReq{
		Clouds: clouds,
	}

	deleteByDatasetsReq := &domain.DeleteDatasetsReq{
		Datasets: clouds,
	}

	generalFilterWithSpecificFilterReq := &domain.DeleteEventsReq{
		EventIDs: events,
		Clouds:   clouds,
	}

	noConditionsReq := &domain.DeleteEventsReq{}

	customerID := "some-customer-id"

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful delete events by ids",
			args: args{
				body: deleteByEventIDsReq,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataByEventIDs",
					ginContextMock,
					customerID,
					*deleteByEventIDsReq,
					deletedBy,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "successful delete events by cloud",
			args: args{
				body: deleteByCloudsReq,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataByClouds",
					ginContextMock,
					customerID,
					*deleteByDatasetsReq,
					deletedBy,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "forbidden general filter with specific filter",
			args: args{
				body: generalFilterWithSpecificFilterReq,
			},
			wantErr: true,
		},
		{
			name: "forbidden if no conditions specified in the request",
			args: args{
				body: noConditionsReq,
			},
			wantErr: true,
		},
		{
			name: "service returns error",
			args: args{
				body: deleteByEventIDsReq,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataByEventIDs",
					ginContextMock,
					customerID,
					*deleteByEventIDsReq,
					deletedBy,
				).
					Return(errors.New("some service error")).
					Once()
			},
			wantErr: true,
		},
		{
			name:    "ShouldBindJSON error",
			args:    args{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Set("email", deletedBy)
			ctx.Request = request

			respond := h.DeleteCustomerSpecificEvents(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.DeleteCustomerSpecificEvents() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_GetCustomerDataSummary(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		forceRefresh bool
	}

	customerID := "some-customer-id"

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		args         args
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful get customer datasets with force refresh",
			on: func(f *fields) {
				f.service.On(
					"GetCustomerDatasets",
					ginContextMock,
					customerID,
					true,
				).
					Return(nil, nil).
					Once()
			},
			args: args{
				forceRefresh: true,
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "successful get customer datasets without force refresh",
			on: func(f *fields) {
				f.service.On(
					"GetCustomerDatasets",
					ginContextMock,
					customerID,
					false,
				).
					Return(nil, nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			target := "/someRequest"

			if tt.args.forceRefresh {
				target = target + "?forceRefresh=true"
			}

			request := httptest.NewRequest(http.MethodGet, target, nil)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Set("email", "test@doit.com")
			ctx.Request = request

			respond := h.GetCustomerDatasets(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.GetCustomerDataSummary() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_GetCustomerDatasetBatches(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		customerID   string
		datasetName  string
		forceRefresh bool
	}

	customerID := "some-customer-id"

	datasetName := "some-dataset-name"

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		args         args
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful get customer dataset batches with force refresh",
			on: func(f *fields) {
				f.service.On(
					"GetCustomerDatasetBatches",
					ginContextMock,
					customerID,
					datasetName,
					true,
				).
					Return(nil, nil).
					Once()
			},
			args: args{
				customerID:   customerID,
				datasetName:  datasetName,
				forceRefresh: true,
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "successful get customer dataset batches without force refresh",
			on: func(f *fields) {
				f.service.On(
					"GetCustomerDatasetBatches",
					ginContextMock,
					customerID,
					datasetName,
					false,
				).
					Return(nil, nil).
					Once()
			},
			args: args{
				customerID:  customerID,
				datasetName: datasetName,
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error no customerID",
			args: args{
				customerID:  "",
				datasetName: datasetName,
			},
			wantErr: true,
		},
		{
			name: "error no datasetName",
			args: args{
				customerID:  customerID,
				datasetName: "",
			},
			wantErr: true,
		},
		{
			name: "error in get customer batches",
			args: args{
				customerID:  customerID,
				datasetName: datasetName,
			},
			on: func(f *fields) {
				f.service.On(
					"GetCustomerDatasetBatches",
					ginContextMock,
					customerID,
					datasetName,
					false,
				).
					Return(nil, errors.New("some error")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			target := "/someRequest"

			if tt.args.forceRefresh {
				target = target + "?forceRefresh=true"
			}

			request := httptest.NewRequest(http.MethodGet, target, nil)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: tt.args.customerID,
				},
				{
					Key:   "datasetName",
					Value: tt.args.datasetName,
				},
			}

			ctx.Set("email", "test@doit.com")
			ctx.Request = request

			respond := h.GetCustomerDatasetBatches(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.GetCustomerDatasetBatches() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_DeleteDatasetBatches(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		body        domain.DeleteBatchesReq
		customerID  string
		datasetName string
	}

	const (
		customerID  = "some-customer-id"
		datasetName = "some-dataset-name"
		deletedBy   = "test@doit.com"
	)

	deleteBatchesReq := domain.DeleteBatchesReq{
		Batches: []string{"batch1", "batch2"},
	}

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		args         args
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful delete of dataset batches",
			args: args{
				body:        deleteBatchesReq,
				customerID:  customerID,
				datasetName: datasetName,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteDatasetBatches",
					ginContextMock,
					customerID,
					datasetName,
					deleteBatchesReq,
					deletedBy,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error no customerID",
			args: args{
				customerID:  "",
				datasetName: datasetName,
			},
			wantErr: true,
		},
		{
			name: "error no datasetName",
			args: args{
				customerID:  customerID,
				datasetName: "",
			},
			wantErr: true,
		},
		{
			name: "ShouldBindJSON error",
			args: args{
				customerID:  customerID,
				datasetName: datasetName,
			},
			wantErr: true,
		},
		{
			name: "error missing batches ids",
			args: args{
				body:        domain.DeleteBatchesReq{Batches: []string{}},
				customerID:  customerID,
				datasetName: datasetName,
			},
			wantErr: true,
		},
		{
			name: "error in delete dataset batches",
			args: args{
				body:        deleteBatchesReq,
				customerID:  customerID,
				datasetName: datasetName,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteDatasetBatches",
					ginContextMock,
					customerID,
					datasetName,
					deleteBatchesReq,
					deletedBy,
				).
					Return(errors.New("some error")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodGet, "/someRequest", bodyReader)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: tt.args.customerID,
				},
				{
					Key:   "datasetName",
					Value: tt.args.datasetName,
				},
			}

			ctx.Set("email", "test@doit.com")
			ctx.Request = request

			respond := h.DeleteDatasetBatches(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.DeleteDatasetBatches() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_DeleteCustomerDatasets(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		body *domain.DeleteDatasetsReq
	}

	deletedBy := "test@delete.com"

	deleteByDatasetsReq := domain.DeleteDatasetsReq{
		Datasets: []string{"datadog", "some other dataset"},
	}

	customerID := "some-customer-id"

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		args         args
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful delete specific datasets",
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataByClouds",
					ginContextMock,
					customerID,
					deleteByDatasetsReq,
					deletedBy,
				).
					Return(nil, nil).
					Once()
			},
			args: args{
				body: &deleteByDatasetsReq,
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "fail do delete when no clouds provided",
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataByClouds",
					ginContextMock,
					customerID,
					domain.DeleteDatasetsReq{
						Datasets: []string{},
					},
					deletedBy,
				).
					Return(domain.ErrCloudsCanNotBeEmpty, nil).
					Once()
			},
			args: args{
				body: &domain.DeleteDatasetsReq{
					Datasets: []string{},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			target := "/someRequest"

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))

			request := httptest.NewRequest(http.MethodGet, target, bodyReader)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Set("email", deletedBy)
			ctx.Request = request

			respond := h.DeleteCustomerDatasets(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.DeleteCustomerDatasets() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_DeleteCustomerDataHard(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	const (
		customerID = "12345"
	)

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful delete customer data hard",
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataHard",
					ginContextMock,
					customerID,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error on delete customer data hard",
			on: func(f *fields) {
				f.service.On(
					"DeleteCustomerDataHard",
					ginContextMock,
					customerID,
				).
					Return(errors.New("failed to delete customer data hard")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			request := httptest.NewRequest(http.MethodDelete, "/someRequest", nil)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Request = request

			respond := h.DeleteCustomerDataHard(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.DeleteCustomerDataHard() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_DeleteAllCustomersDataHard(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	const (
		customerID = "12345"
	)

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful delete all customers data",
			on: func(f *fields) {
				f.service.On(
					"DeleteAllCustomersDataHard",
					ginContextMock,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error on delete all customer data",
			on: func(f *fields) {
				f.service.On(
					"DeleteAllCustomersDataHard",
					ginContextMock,
				).
					Return(errors.New("failed to get customers")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			request := httptest.NewRequest(http.MethodDelete, "/someRequest", nil)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Request = request

			respond := h.DeleteAllCustomersDataHard(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.DeleteAllCustomersDataHard() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_CreateDataset(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		body *domain.CreateDatasetRequest
	}

	customerID := "some-customer-id"
	email := "test@doit.com"

	datasetReq := &domain.CreateDatasetRequest{
		Name:        "test UI created dataset",
		Description: "This is a dataset created by the UI",
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful dataset creation",
			args: args{
				body: datasetReq,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateDataset",
					ginContextMock,
					customerID,
					email,
					*datasetReq,
				).Return(nil).Once()
			},
			wantedStatus: http.StatusCreated,
		},
		{
			name: "service returns error",
			args: args{
				body: datasetReq,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateDataset",
					ginContextMock,
					customerID,
					email,
					*datasetReq,
				).Return(errors.New("some service error")).Once()
			},
			wantErr: true,
		},
		{
			name:    "ShouldBindJSON error",
			args:    args{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Set("email", email)
			ctx.Request = request

			respond := h.CreateDataset(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.CreateDataset() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestDataHubAPIHandler_AddRawEvents(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.DataHubService
	}

	type args struct {
		body *domain.RawEventsReq
	}

	const (
		customerID = "some-customer-id"
		email      = "test@doit.com"

		datasetName = "some dataset name"
		source      = "csv"
		filename    = "my file.csv"
	)

	schema := []string{"usage_date", "event_id", "project_id", "label.house", "metric.cost"}
	rawEvents := [][]string{
		{"2024-03-01T00:00:00Z", "73f8b9da-ebb2-4046-8226-1015ee94b499", "pr1", "adoption", "12"},
		{"2024-04-01T00:00:00Z", "73f8b9da-ebb2-4046-8226-1015ee94b222", "pr2", "adoption", "15"},
	}

	rawEventsReq := domain.RawEventsReq{
		Dataset:   datasetName,
		Source:    source,
		Schema:    schema,
		RawEvents: rawEvents,
		Filename:  filename,
		Execute:   true,
	}

	tests := []struct {
		name         string
		fields       fields
		wantErr      bool
		args         args
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful ingest raw data",
			on: func(f *fields) {
				f.service.On(
					"AddRawEvents",
					ginContextMock,
					customerID,
					email,
					rawEventsReq,
				).
					Return(nil, nil, nil).
					Once()
			},
			args: args{
				body: &rawEventsReq,
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "fail on error",
			on: func(f *fields) {
				f.service.On(
					"AddRawEvents",
					ginContextMock,
					customerID,
					email,
					rawEventsReq,
				).
					Return(nil, nil, errors.New("error calling ingestion-api")).
					Once()
			},
			args: args{
				body: &rawEventsReq,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewDataHubService(t),
			}

			h := &DataHub{
				loggerProvider: tt.fields.loggerProvider,
				datahubService: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Params = []gin.Param{
				{
					Key:   "customerID",
					Value: customerID,
				},
			}

			ctx.Set("email", email)
			ctx.Request = request

			respond := h.AddRawEvents(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("DataHub.AddRawEvents() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

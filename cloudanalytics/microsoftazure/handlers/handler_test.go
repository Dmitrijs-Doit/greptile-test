package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	azureTableMgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure/tablemanagement"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure/tablemanagement/mocks"
)

func TestAnalyticsAzureAggregation_UpdateAggregateAllAzureCustomers(t *testing.T) {
	type fields struct {
		service *mocks.BillingTableManagementService
	}

	tests := []struct {
		name         string
		on           func(*fields)
		queryParams  map[string]string
		wantedStatus int
	}{
		{
			name: "successful call to function to update all aggregated tables for all customers",
			on: func(f *fields) {
				f.service.On("UpdateAllAggregatedTablesAllCustomers", mock.AnythingOfType("*gin.Context"), true).
					Return(nil).
					Once()
			},
			queryParams:  map[string]string{"allPartitions": "true"},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error return on update of all aggregated tables for all customers",
			on: func(f *fields) {
				f.service.On("UpdateAllAggregatedTablesAllCustomers", mock.AnythingOfType("*gin.Context"), true).
					Return([]error{errors.New("error updating tables")}).
					Once()
			},
			queryParams:  map[string]string{"allPartitions": "true"},
			wantedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service: &mocks.BillingTableManagementService{},
			}

			h := &AnalyticsMicrosoftAzure{
				tableMgmt: fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			query := url.Values{}
			for key, value := range tt.queryParams {
				query.Add(key, value)
			}

			request.URL.RawQuery = query.Encode()

			ctx.Request = request

			retErr := h.UpdateAggregateAllAzureCustomers(ctx)
			if retErr != nil && tt.wantedStatus == http.StatusOK {
				t.Errorf("AnalyticsAzureAggregation.UpdateAggregateAllAzureCustomers() error = %v, wantedStatus %v", retErr, tt.wantedStatus)
			}

			assert.Equal(t, tt.wantedStatus, recorder.Code)
		})
	}
}

func TestAnalyticsAzureAggregation_UpdateAggregatedTableAzure(t *testing.T) {
	type fields struct {
		service *mocks.BillingTableManagementService
	}

	tests := []struct {
		name        string
		on          func(*fields)
		queryParams map[string]string
		paramSuffix string
		wantErr     error
	}{
		{
			name: "successful update of aggregated table",
			on: func(f *fields) {
				f.service.On("UpdateAggregatedTable",
					mock.AnythingOfType("*gin.Context"), "suffix1", "interval1", true).Return(nil).Once()
			},
			queryParams: map[string]string{
				queryParamAllPartitions: "true",
				queryParamInterval:      "interval1",
				pathParamCustomerID:     "suffix1",
			},
			paramSuffix: "suffix1",
		},
		{
			name: "error return on update of aggregated table when suffix is empty",
			on: func(f *fields) {
				f.service.On("UpdateAggregatedTable",
					mock.AnythingOfType("*gin.Context"), "", "interval1", true).Return(azureTableMgmt.ErrSuffixIsEmpty).
					Once()
			},
			queryParams: map[string]string{
				queryParamAllPartitions: "true",
				queryParamInterval:      "interval1",
				pathParamCustomerID:     "",
			},
			paramSuffix: "",
			wantErr:     azureTableMgmt.ErrSuffixIsEmpty,
		},
		{
			name: "error return on update of aggregated table when interval is empty",
			on: func(f *fields) {
				f.service.On("UpdateAggregatedTable",
					mock.AnythingOfType("*gin.Context"), "suffix1", "", true).Return(errors.New("interval query param is required")).
					Once()
			},
			queryParams: map[string]string{
				queryParamAllPartitions: "true",
				queryParamInterval:      "",
				pathParamCustomerID:     "suffix1",
			},
			paramSuffix: "suffix1",
			wantErr:     errors.New("interval query param is required"),
		},
		{
			name: "error return on update of aggregated table",
			on: func(f *fields) {
				f.service.On("UpdateAggregatedTable",
					mock.AnythingOfType("*gin.Context"), "suffix1", "interval1", true).Return(nil).
					Return(errors.New("error updating table")).
					Once()
			},
			queryParams: map[string]string{
				queryParamAllPartitions: "true",
				queryParamInterval:      "interval1",
				pathParamCustomerID:     "suffix1",
			},
			paramSuffix: "suffix1",
			wantErr:     errors.New("error updating table"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			fields := fields{
				service: &mocks.BillingTableManagementService{},
			}

			h := &AnalyticsMicrosoftAzure{
				tableMgmt: fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			query := url.Values{}
			for key, value := range tt.queryParams {
				query.Add(key, value)
			}

			request.URL.RawQuery = query.Encode()

			ctx.Request = request
			ctx.Params = []gin.Param{
				{Key: "customerID", Value: tt.paramSuffix},
			}

			retErr := h.UpdateAggregatedTableAzure(ctx)
			if retErr != nil && tt.wantErr == nil {
				t.Errorf("AnalyticsAzureAggregation.UpdateAggregatedTableAzure() error = %v, wantedError %v", retErr, tt.wantErr)
			}

			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), retErr.Error())
			}
		})
	}
}

func TestAnalyticsAzureAggregation_UpdateAllAzureAggregatedTables(t *testing.T) {
	type fields struct {
		service *mocks.BillingTableManagementService
	}

	tests := []struct {
		name        string
		on          func(*fields)
		queryParams map[string]string
		paramSuffix string
		wantErr     error
	}{
		{
			name: "successful update of all aggregated tables",
			on: func(f *fields) {
				f.service.On("UpdateAllAggregatedTables", mock.AnythingOfType("*gin.Context"), "suffix1", true).Return(nil).Once()
			},
			queryParams: map[string]string{queryParamAllPartitions: "true"},
			paramSuffix: "suffix1",
		},
		{
			name: "error return on update of all aggregated tables",
			on: func(f *fields) {
				f.service.On("UpdateAllAggregatedTables", mock.AnythingOfType("*gin.Context"), "suffix2", false).
					Return([]error{errors.New("error updating tables")}).
					Once()
			},
			queryParams: map[string]string{queryParamAllPartitions: "false"},
			paramSuffix: "suffix2",
		},
		{
			name: "error return on update of all aggregated tables when suffix is empty",
			on: func(f *fields) {
				f.service.On("UpdateAllAggregatedTables",
					mock.AnythingOfType("*gin.Context"), "", false).Return([]error{azureTableMgmt.ErrSuffixIsEmpty}).
					Once()
			},
			queryParams: map[string]string{"allPartitions": "false"},
			paramSuffix: "",
			wantErr:     azureTableMgmt.ErrSuffixIsEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service: &mocks.BillingTableManagementService{},
			}

			h := &AnalyticsMicrosoftAzure{
				tableMgmt: fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			query := url.Values{}
			for key, value := range tt.queryParams {
				query.Add(key, value)
			}

			request.URL.RawQuery = query.Encode()

			ctx.Request = request
			ctx.Params = []gin.Param{
				{Key: "customerID", Value: tt.paramSuffix},
			}

			retErr := h.UpdateAllAzureAggregatedTables(ctx)
			if retErr != nil && tt.wantErr == nil {
				t.Errorf("AnalyticsAzureAggregation.UpdateAllAzureAggregatedTables() error = %v, wantedError %v", retErr, tt.wantErr)
			}

			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), retErr.Error())
			}
		})
	}
}

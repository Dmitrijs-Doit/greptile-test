package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service"
	alertTierMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier/mocks"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

var (
	customerID = "test-customer-id"
)

func GetContextExternalAPI() *gin.Context {
	request := httptest.NewRequest(http.MethodGet, "/alerts", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	ctx.Set("email", email)
	ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

	return ctx
}

func TestAnalyticsAlerts_ExternalAPIListAlerts(t *testing.T) {
	ctx := GetContextExternalAPI()

	ctxWithSorting := GetContextExternalAPI()
	ctxWithSorting.Request.URL.RawQuery = "sortBy=createTime&sortOrder=desc"

	ctxWithWrongParameters := GetContextExternalAPI()
	ctxWithWrongParameters.Request.URL.RawQuery = "sortBy=createTime&sortOrder=bad-order-type"

	tests := []test{
		{
			name:     "Should handle request without parameters",
			args:     args{ctx},
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("ListAlerts", ctx, service.ExternalAPIListArgsReq{
						CustomerID: customerID,
						Email:      email,
						SortBy:     "createTime",
						SortOrder:  firestore.Desc,
						Filters:    []customerapi.Filter{},
					}).
					Return(&service.ExternalAlertList{}, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ListAlerts", 1)
			},
		},
		{
			name:     "Should handle request with valid parameters",
			args:     args{ctxWithSorting},
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("ListAlerts", ctxWithSorting, service.ExternalAPIListArgsReq{
						CustomerID: customerID,
						Email:      email,
						SortBy:     "createTime",
						SortOrder:  firestore.Desc,
						Filters:    []customerapi.Filter{},
					}).
					Return(&service.ExternalAlertList{}, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ListAlerts", 1)
			},
		},
		{
			name:     "Should handle request with invalid parameters",
			args:     args{ctxWithWrongParameters},
			outErr:   errors.New("query parameter error, invalid sort order key: bad-order-type"),
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("ListAlerts", ctxWithSorting, service.ExternalAPIListArgsReq{
						CustomerID: customerID,
						Email:      email,
						SortBy:     "createTime",
						SortOrder:  firestore.Desc,
						Filters:    []customerapi.Filter{},
					}).
					Return(&service.ExternalAlertList{}, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ListAlerts", 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AlertsService{},
				alertTierMocks.NewAlertTierService(t),
			}
			h := &AnalyticsAlerts{
				loggerProvider:   tt.fields.loggerProvider,
				service:          tt.fields.service,
				alertTierService: tt.fields.alertTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.ExternalAPIListAlerts(tt.args.ctx)
			status := ctx.Writer.Status()

			if tt.outSatus != 0 && tt.outSatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.outSatus)
			}

			if result != nil && result.Error() != tt.outErr.Error() {
				t.Errorf("got %v, want %v", result, tt.outErr)
			}

			if tt.assert != nil {
				tt.assert(t, &tt.fields)
			}
		})
	}
}

func TestAnalyticsAlerts_ExternalAPICreateAlert(t *testing.T) {
	ctx := GetContextExternalAPI()

	var validAlert = map[string]interface{}{
		"config": map[string]interface{}{
			"condition": "percentage-change",
			"currency":  "USD",
			"metric": map[string]interface{}{
				"type":  "basic",
				"value": "cost",
			},
			"operator":     "gt",
			"attributions": []string{"attributionId"},
			"timeInterval": "year",
			"value":        423,
		},
		"name": "Alert name",
	}

	validBody, _ := json.Marshal(validAlert)
	invalidAlert := validAlert
	invalidAlert["name"] = ""
	invalidBody, _ := json.Marshal(invalidAlert)

	tests := []struct {
		name        string
		fields      fields
		on          func(*fields)
		args        args
		aborted     bool
		wantErr     error
		requestBody []byte
	}{
		{
			name: "Happy path",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("CreateAlert", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx},
			aborted:     false,
		},
		{
			name: "Failed on JSON validation tags",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("CreateAlert", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{}).
					Once()
			},
			requestBody: invalidBody,
			args:        args{ctx: ctx},
			aborted:     true,
		},
		{
			name: "Service returned validation errors",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("CreateAlert", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{
						ValidationErrors: []error{errormsg.ErrorMsg{Field: "field", Message: "error"}},
						Error:            domain.ErrValidationErrors,
					}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx},
			aborted:     true,
		},
		{
			name: "Service returned unknown error",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("CreateAlert", mock.AnythingOfType("*gin.Context"), mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{Error: errors.New("unknown error")}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx},
			aborted:     true,
			wantErr:     errors.New("unknown error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AlertsService{},
				alertTierMocks.NewAlertTierService(t),
			}
			h := &AnalyticsAlerts{
				loggerProvider:   tt.fields.loggerProvider,
				service:          tt.fields.service,
				alertTierService: tt.fields.alertTierService,
			}
			ctx.Request.Body = io.NopCloser(bytes.NewReader(tt.requestBody))

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := h.ExternalAPICreateAlert(tt.args.ctx)
			aborted := ctx.IsAborted()

			if tt.aborted != aborted {
				t.Errorf("AnalyticsAlerts.ExternalAPICreateAlert() request aborted %v, want %v", aborted, tt.aborted)
			}

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAnalyticsAlerts_ExternalAPIUpdateAlert(t *testing.T) {
	alertID := "alert-id"
	ctx := GetContextExternalAPI()

	type args struct {
		ctx     *gin.Context
		alertID string
	}

	var validAlert = map[string]interface{}{
		"config": map[string]interface{}{
			"condition": "percentage-change",
			"currency":  "USD",
			"metric": map[string]interface{}{
				"type":  "basic",
				"value": "cost",
			},
			"operator":     "gt",
			"attributions": []string{"attributionId"},
			"timeInterval": "year",
			"value":        423,
		},
		"name": "Alert name",
	}

	validBody, _ := json.Marshal(validAlert)
	invalidAlert := validAlert
	invalidAlert["name"] = 1111
	invalidBody, _ := json.Marshal(invalidAlert)

	emptyBody, _ := json.Marshal(map[string]interface{}{})
	validBodyNameUpdate, _ := json.Marshal(map[string]interface{}{
		"name": "Alert new name",
	})

	tests := []struct {
		name        string
		fields      fields
		on          func(*fields)
		args        args
		aborted     bool
		wantErr     error
		requestBody []byte
	}{
		{
			name: "Happy path",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("UpdateAlert", mock.AnythingOfType("*gin.Context"), alertID, mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx, alertID: alertID},
			aborted:     false,
		},
		{
			name: "Happy path, only name update",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("UpdateAlert", mock.AnythingOfType("*gin.Context"), alertID, mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{}).
					Once()
			},
			requestBody: validBodyNameUpdate,
			args:        args{ctx: ctx, alertID: alertID},
			aborted:     false,
		},
		{
			name:    "Missing alert ID",
			args:    args{ctx: ctx, alertID: ""},
			wantErr: domain.ErrMissingAlertID,
			aborted: true,
		},
		{
			name:        "Failed on JSON validation tags",
			requestBody: invalidBody,
			args:        args{ctx: ctx, alertID: alertID},
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
			},
			aborted: true,
		},
		{
			name:        "Failed on empty body",
			requestBody: emptyBody,
			args:        args{ctx: ctx, alertID: alertID},
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
			},
			wantErr: domain.ErrEmptyBody,
			aborted: true,
		},
		{
			name: "Service returned validation errors",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("UpdateAlert", mock.AnythingOfType("*gin.Context"), alertID, mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{
						ValidationErrors: []error{errormsg.ErrorMsg{Field: "field", Message: "error"}},
						Error:            domain.ErrValidationErrors,
					}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx, alertID: alertID},
			aborted:     true,
		},
		{
			name: "Service returned unknown error",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("UpdateAlert", mock.AnythingOfType("*gin.Context"), alertID, mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{Error: errors.New("unknown error")}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx, alertID: alertID},
			aborted:     true,
			wantErr:     errors.New("unknown error"),
		},
		{
			name: "Alert with this ID not found",
			on: func(f *fields) {
				f.alertTierService.On(
					"CheckAccessToAlerts",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil)
				f.service.
					On("UpdateAlert", mock.AnythingOfType("*gin.Context"), alertID, mock.AnythingOfType("service.ExternalAPICreateUpdateArgsReq")).
					Return(service.ExternalAPICreateUpdateResp{Error: doitFirestore.ErrNotFound}).
					Once()
			},
			requestBody: validBody,
			args:        args{ctx: ctx, alertID: alertID},
			aborted:     true,
			wantErr:     domain.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.AlertsService{},
				alertTierMocks.NewAlertTierService(t),
			}
			h := &AnalyticsAlerts{
				loggerProvider:   tt.fields.loggerProvider,
				service:          tt.fields.service,
				alertTierService: tt.fields.alertTierService,
			}
			ctx.Request.Body = io.NopCloser(bytes.NewReader(tt.requestBody))
			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.alertID},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := h.ExternalAPIUpdateAlert(tt.args.ctx)
			aborted := ctx.IsAborted()

			if tt.aborted != aborted {
				t.Errorf("AnalyticsAlerts.ExternalAPIUpdateAlert() request aborted %v, want %v", aborted, tt.aborted)
			}

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAnalyticsAlerts_APIDeleteAlertHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	email := "requester@example.com"
	alertID := "alert-id"
	customerID := "test_customer_id"

	labels := map[string]string{"customerId": customerID, "email": email, "alertId": alertID}
	labelsNoAlertID := map[string]string{"customerId": customerID, "email": email, "alertId": ""}

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            serviceMock.AlertsService
	}

	type args struct {
		ctx     *gin.Context
		alertID string
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successfully deleted alert",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.service.
					On(
						"DeleteAlert",
						mock.AnythingOfType("*gin.Context"),
						customerID,
						email,
						alertID,
					).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error returned when deleting alert",
			args: args{
				ctx:     ctx,
				alertID: alertID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.service.
					On(
						"DeleteAlert",
						mock.AnythingOfType("*gin.Context"),
						customerID,
						email,
						alertID,
					).
					Return(domain.ErrForbidden).
					Once()
			},
		},
		{
			name: "error returned when no alert id",
			args: args{
				ctx:     ctx,
				alertID: "",
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labelsNoAlertID).Once()
			},
			wantedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service:            serviceMock.AlertsService{},
				loggerProviderMock: loggerMocks.ILogger{},
			}

			h := &AnalyticsAlerts{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodDelete, "/someRequest", nil)

			ctx.Set("email", email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)
			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.alertID},
			}

			ctx.Request = request

			respond := h.ExternalAPIDeleteAlert(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("ExternalAPIDeleteAlert() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

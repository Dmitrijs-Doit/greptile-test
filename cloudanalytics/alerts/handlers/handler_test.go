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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service"
	alertTierMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier/mocks"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

var (
	body    = make(map[string]interface{})
	email   = "requester@example.com"
	alertID = "my_alert_id"
)

func GetContext() (*gin.Context, []byte) {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("email", email)
	ctx.Set("doitEmployee", false)
	ctx.Request = request
	ctx.Params = []gin.Param{
		{Key: "customerID", Value: "123"},
		{Key: "alertID", Value: alertID}}

	body["public"] = "viewer"
	body["collaborators"] = []map[string]interface{}{
		{
			"email": email,
			"role":  "owner",
		},
	}

	jsonbytes, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}

	return ctx, jsonbytes
}

type fields struct {
	loggerProvider   logger.Provider
	service          *serviceMock.AlertsService
	alertTierService *alertTierMocks.AlertTierService
}

type args struct {
	ctx *gin.Context
}

type test struct {
	name   string
	fields fields
	args   args

	outErr   error
	outSatus int
	on       func(*fields)
	assert   func(*testing.T, *fields)

	errCtxBody         bool
	errNoCollaborators bool
}

func TestAnalyticsAlerts_UpdateAlertSharingHandler(t *testing.T) {
	ctx, jsonbytes := GetContext()

	ctxWithoutAlertID := ctx.Copy()
	ctxWithoutAlertID.Params = []gin.Param{
		{Key: "customerID", Value: "123"}}

	var bodyWithoutCollaborators = make(map[string]interface{})
	bodyWithoutCollaborators["public"] = "viewer"

	jsonbytesWithoutCollaborators, err := json.Marshal(bodyWithoutCollaborators)
	if err != nil {
		panic(err)
	}

	tests := []test{
		{
			name:     "Happy path",
			args:     args{ctx: ctx},
			outErr:   nil,
			outSatus: http.StatusOK,
			on: func(f *fields) {
				f.service.
					On("ShareAlert", ctx, mock.Anything, mock.Anything, alertID, email, "", ctx.Params.ByName("customerID")).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ShareAlert", 1)
			},
		},
		{
			name:   "ShareAlert returns error",
			args:   args{ctx: ctx},
			outErr: errors.New("error"),
			on: func(f *fields) {
				f.service.
					On("ShareAlert", ctx, mock.Anything, mock.Anything, alertID, email, "", ctx.Params.ByName("customerID")).
					Return(errors.New("error")).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ShareAlert", 1)
			},
		},
		{
			name:   "block sharing alert in presentation mode",
			args:   args{ctx: ctx},
			outErr: service.ErrNoAuthorization,
			// outSatus: http.StatusForbidden,
			on: func(f *fields) {
				f.service.
					On("ShareAlert", ctx, mock.Anything, mock.Anything, alertID, email, "", ctx.Params.ByName("customerID")).
					Return(service.ErrNoAuthorization).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.service.AssertNumberOfCalls(t, "ShareAlert", 1)
			},
		},
		{
			name:       "ShouldBindJSON returns error",
			args:       args{ctx: ctx},
			outErr:     errors.New("invalid request"),
			errCtxBody: true,
		},
		{
			name:   "alertID is not present",
			args:   args{ctx: ctxWithoutAlertID},
			outErr: service.ErrNoAlertID,
		},
		{
			name:               "collaborators are not present",
			args:               args{ctx: ctx},
			outErr:             service.ErrNoCollaborators,
			errNoCollaborators: true,
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
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))
			if tt.errCtxBody {
				ctx.Request.Body = nil
			}

			if tt.errNoCollaborators {
				ctx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytesWithoutCollaborators))
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result := h.UpdateAlertSharingHandler(tt.args.ctx)
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

func TestAnalyticsAlerts_RefreshAlerts(t *testing.T) {
	ctx, jsonbytes := GetContext()

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			on: func(f *fields) {
				f.service.
					On("RefreshAlerts", mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Once()
			},
			args: args{ctx: ctx},
		},
		{
			name: "RefreshAlerts returns error",
			on: func(f *fields) {
				f.service.
					On("RefreshAlerts", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("error")).
					Once()
			},
			args:    args{ctx: ctx},
			wantErr: true,
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
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := h.RefreshAlerts(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlerts.RefreshAlerts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlerts_RefreshAlert(t *testing.T) {
	ctx, jsonbytes := GetContext()

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			on: func(f *fields) {
				f.service.
					On("RefreshAlert", mock.Anything, mock.Anything, mock.Anything).
					Return(nil).
					Once()
			},
			args: args{ctx: ctx},
		},
		{
			name: "RefreshAlert returns error",
			on: func(f *fields) {
				f.service.
					On("RefreshAlert", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("error")).
					Once()
			},
			args:    args{ctx: ctx},
			wantErr: true,
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
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := h.RefreshAlert(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlerts.RefreshAlert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewAnalyticsAlerts(t *testing.T) {
	type args struct {
		log  logger.Provider
		conn *connection.Connection
	}

	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Errorf("main: could not initialize db connections. error %s", err)
	}

	tests := []struct {
		name string
		args args
	}{
		{
			name: "Happy path",
			args: args{
				log:  logger.FromContext,
				conn: conn,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAnalyticsAlerts(ctx, tt.args.log, tt.args.conn)
			if a == nil {
				t.Errorf("NewAnalyticsAlerts() = nil, want non-nil")
			}
		})
	}
}

func TestAnalyticsAlerts_DeleteAlert(t *testing.T) {
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
			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
				{Key: "alertID", Value: tt.args.alertID},
			}

			ctx.Request = request

			respond := h.DeleteAlert(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("ExternalAPIDeleteAlert() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestAnalyticsAlerts_DeleteManyHandler(t *testing.T) {
	validBody := `{"ids": ["123", "1234"]}`
	invalidBody := `{"ids" []}`
	emptyBody := `{"ids": []}`

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	email := "requester@example.com"
	customerID := "test_customer_id"

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            serviceMock.AlertsService
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name        string
		args        args
		on          func(*fields)
		requestBody io.ReadCloser
		wantErr     bool
	}{
		{
			name: "successfully deleted alerts",
			args: args{
				ctx: ctx,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.
					On(
						"DeleteMany",
						mock.AnythingOfType("*gin.Context"),
						email,
						[]string{"123", "1234"},
					).
					Return(nil).
					Once()
			},
			requestBody: io.NopCloser(bytes.NewBufferString(validBody)),
		},
		{
			name: "error alert ids is empty",
			args: args{
				ctx: ctx,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
			},
			requestBody: io.NopCloser(bytes.NewBufferString(emptyBody)),
		},
		{
			name: "error invalid json",
			args: args{
				ctx: ctx,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
			},
			requestBody: io.NopCloser(bytes.NewBufferString(invalidBody)),
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

			ctx.Request = httptest.NewRequest(http.MethodDelete, "/someRequest", tt.requestBody)
			ctx.Set("email", email)
			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
			}

			respond := h.DeleteManyHandler(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("ExternalAPIDeleteAlert() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

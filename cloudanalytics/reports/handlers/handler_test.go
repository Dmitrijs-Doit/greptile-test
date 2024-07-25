package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service"
	reportServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier"
	reportTierServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestReportHandler_CreateReportExternalHandler(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"

	description := "some description"

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	type args struct {
		body externalreport.ExternalReport
	}

	validExternalReport := externalreport.ExternalReport{
		Name:        "some name",
		Description: &description,
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				body: validExternalReport,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"CreateReportWithExternal",
					mock.AnythingOfType("*gin.Context"),
					&validExternalReport,
					customerID,
					email,
				).
					Return(&externalreport.ExternalReport{}, nil, nil).
					Once()
			},
			wantedStatus: http.StatusCreated,
		},
		{
			name: "Requires higher tier",
			args: args{
				body: validExternalReport,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
			wantedStatus: http.StatusForbidden,
		},
		{
			name: "Error - did not pass validation",
			args: args{
				body: validExternalReport,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"CreateReportWithExternal",
					mock.AnythingOfType("*gin.Context"),
					&validExternalReport,
					customerID,
					email,
				).
					Return(nil, []errormsg.ErrorMsg{{Field: "metric", Message: "invalid metric: InvalidMetric"}}, errors.New("error creating report")).
					Once()
			},
			wantedStatus: http.StatusBadRequest,
		},

		{
			name: "Error - internal",
			args: args{
				body: validExternalReport,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"CreateReportWithExternal",
					mock.AnythingOfType("*gin.Context"),
					&validExternalReport,
					customerID,
					email,
				).
					Return(nil, nil, errors.New("internal error")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Set("email", email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

			ctx.Request = request

			respond := h.CreateReportExternalHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("Report.CreateReportExternalHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestReportHandler_UpdateReportExternalHandler(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"
	reportID := "11111"
	description := "some description"

	labelsNoReportID := map[string]string{"customerId": customerID, "email": email, "reportId": ""}

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	type args struct {
		reportID string
		body     externalreport.ExternalReport
	}

	validExternalReport := externalreport.ExternalReport{
		Name:        "some name",
		Description: &description,
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				body:     validExternalReport,
				reportID: reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"UpdateReportWithExternal",
					mock.AnythingOfType("*gin.Context"),
					reportID,
					&validExternalReport,
					customerID,
					email,
				).
					Return(&externalreport.ExternalReport{}, nil, nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Requires higher tier",
			args: args{
				body:     validExternalReport,
				reportID: reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
			wantedStatus: http.StatusForbidden,
		},
		{
			name: "Error - did not pass validation",
			args: args{
				body:     validExternalReport,
				reportID: reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"UpdateReportWithExternal",
					mock.AnythingOfType("*gin.Context"),
					reportID,
					&validExternalReport,
					customerID,
					email,
				).
					Return(nil, []errormsg.ErrorMsg{
						{
							Field: "metric", Message: "invalid metric: InvalidMetric",
						},
					}, errors.New("error updating report")).
					Once()
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "Error - internal",
			args: args{
				reportID: reportID,
				body:     validExternalReport,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&validExternalReport,
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"UpdateReportWithExternal",
					mock.AnythingOfType("*gin.Context"),
					reportID,
					&validExternalReport,
					customerID,
					email,
				).
					Return(nil, nil, errors.New("internal error")).
					Once()
			},
			wantErr: true,
		},
		{
			name: "error no report id",
			args: args{
				reportID: "",
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labelsNoReportID).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			bodyStr, err := json.Marshal(tt.args.body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPatch, "/someRequest", bodyReader)

			ctx.Set("email", email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.reportID},
			}

			ctx.Request = request

			respond := h.UpdateReportExternalHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("Report.UpdateReportExternalHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestReportHandler_DeleteReportExternalHandler(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"
	defaultReportID := "456"

	labels := map[string]string{"customerId": customerID, "email": email, "reportId": defaultReportID}
	labelsNoReportID := map[string]string{"customerId": customerID, "email": email, "reportId": ""}

	expectedDeleteReportError := errors.New("error deleting report")

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	type args struct {
		reportID string
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successful delete",
			args: args{
				reportID: defaultReportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"DeleteReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					email,
					defaultReportID,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "higher tier is required for delete",
			args: args{
				reportID: defaultReportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
			wantedStatus: http.StatusForbidden,
		},
		{
			name: "error no report id",
			args: args{
				reportID: "",
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labelsNoReportID).Once()
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "delete report error",
			args: args{
				reportID: defaultReportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"DeleteReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					email,
					defaultReportID,
				).
					Return(expectedDeleteReportError).
					Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "delete report unauthorized",
			args: args{
				reportID: defaultReportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"DeleteReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					email,
					defaultReportID,
				).
					Return(service.ErrUnauthorizedDelete).
					Once()
			},
			wantErr: true,
		},
		{
			name: "forbidden invalid customer id",
			args: args{
				reportID: defaultReportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", labels).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"DeleteReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					email,
					defaultReportID,
				).
					Return(service.ErrInvalidCustomerID).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodDelete, "/someRequest", nil)

			ctx.Set("email", email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)
			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.reportID},
			}

			ctx.Request = request

			respond := h.DeleteReportExternalHandler(ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("Report.DeleteReportExternalHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func getContext() *gin.Context {
	request := httptest.NewRequest(http.MethodDelete, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	return ctx
}

func TestReportHandler_DeleteManyHandler(t *testing.T) {
	validBody := `{"ids": ["123", "1234"]}`
	invalidBody := `{"ids" []}`
	emptyBody := `{"ids": []}`

	customerID := "customerID"
	email := "email"

	ctx := getContext()

	type args struct {
		ctx        *gin.Context
		customerID string
		email      string
	}

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		requestBody io.ReadCloser
		on          func(*fields)
		wantErr     bool
		expectedErr error
	}{
		{
			name: "successfully delete report",
			args: args{
				ctx,
				customerID,
				email,
			},
			requestBody: io.NopCloser(bytes.NewBufferString(validBody)),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On("DeleteMany", ctx, customerID, email, []string{"123", "1234"}).Return(nil)
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.reportTierService.On(
					"CheckAccessToPresetReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "tier upgrade required for delete report",
			args: args{
				ctx,
				customerID,
				email,
			},
			requestBody: io.NopCloser(bytes.NewBufferString(validBody)),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On("DeleteMany", ctx, customerID, email, []string{"123", "1234"}).Return(nil)
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
		},
		{
			name: "error empty ids list",
			args: args{
				ctx,
				customerID,
				email,
			},
			requestBody: io.NopCloser(bytes.NewBufferString(emptyBody)),
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.reportTierService.On(
					"CheckAccessToPresetReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.loggerProviderMock.On("Errorf", mock.Anything, mock.Anything).Once()
			},
			wantErr: true,
		},
		{
			name: "error no ids list",
			args: args{
				ctx,
				customerID,
				email,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.reportTierService.On(
					"CheckAccessToPresetReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.loggerProviderMock.On("Errorf", mock.Anything, mock.Anything).Once()
			},
			requestBody: io.NopCloser(bytes.NewBufferString(invalidBody)),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			ctx.Request.Body = tt.requestBody

			ctx.Set(common.CtxKeys.Email, tt.args.email)
			ctx.Params = []gin.Param{{Key: "customerID", Value: tt.args.customerID}}

			if response := h.DeleteManyHandler(tt.args.ctx); (response != nil) != tt.wantErr {
				t.Errorf("Reports.DeleteManyHandler() error = %v, wantErr %v", response, tt.wantErr)
			} else {
				if tt.expectedErr != nil {
					assert.ErrorIs(t, tt.expectedErr, response)
				}
			}
		})
	}
}

func TestReportHandler_ShareReportHandler(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"
	reportID := "report-id"
	userID := "user-id"
	name := "some name"

	var validCollabData = map[string]interface{}{
		"collaborators": []map[string]interface{}{
			{
				"Email": email,
				"role":  "owner",
			}},
	}

	var invalidCollabData = map[string]interface{}{
		"collaborators": []map[string]interface{}{},
	}

	validBody, _ := json.Marshal(validCollabData)
	invalidBody, _ := json.Marshal(invalidCollabData)

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	type args struct {
		customerID string
		reportID   string
	}

	tests := []struct {
		name         string
		args         args
		requestBody  []byte
		on           func(*fields)
		wantedStatus int
		wantErr      error
	}{
		{
			name:        "Happy path",
			requestBody: validBody,
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"ShareReport",
					mock.AnythingOfType("*gin.Context"),
					mock.AnythingOfType("report.ShareReportArgsReq"),
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name:        "Requires tier upgrade",
			requestBody: validBody,
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
		},
		{
			name: "Invalid Body",
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			requestBody: invalidBody,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      collab.ErrNoCollaborators,
		},
		{
			name: "Missing param report ID",
			args: args{
				customerID: customerID,
				reportID:   "",
			},
			requestBody: invalidBody,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      service.ErrInvalidReportID,
		},
		{
			name: "Missing params customer ID",
			args: args{
				customerID: "",
				reportID:   reportID,
			},
			requestBody: invalidBody,
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      service.ErrInvalidCustomerID,
		},
		{
			name:        "Interval Server Error",
			requestBody: validBody,
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToCustomReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"ShareReport",
					mock.AnythingOfType("*gin.Context"),
					mock.AnythingOfType("report.ShareReportArgsReq"),
				).
					Return(errors.New("some error")).
					Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			ctx.Set(common.CtxKeys.Email, email)
			ctx.Set(common.CtxKeys.UserID, userID)
			ctx.Set(common.CtxKeys.Name, name)

			ctx.Params = []gin.Param{
				{Key: "customerID", Value: tt.args.customerID},
				{Key: "reportID", Value: tt.args.reportID},
			}

			bodyReader := strings.NewReader(string(tt.requestBody))
			request := httptest.NewRequest(http.MethodPatch, "/someRequest", bodyReader)
			ctx.Request = request
			gotError := h.ShareReportHandler(ctx)

			if tt.wantErr != nil {
				assert.ErrorContains(t, gotError, tt.wantErr.Error())
			} else {
				assert.NoError(t, gotError)
			}
		})
	}
}

func TestReportHandler_GetReportConfigExternalHandler(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"
	reportID := "report-id"

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	type args struct {
		customerID string
		reportID   string
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      error
	}{
		{
			name: "Happy path",
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&externalreport.ExternalReport{},
					false,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"GetReportConfig",
					mock.AnythingOfType("*gin.Context"),
					reportID,
					customerID,
				).
					Return(&externalreport.ExternalReport{}, nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Requires tier upgrade",
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.service.On(
					"GetReportConfig",
					mock.AnythingOfType("*gin.Context"),
					reportID,
					customerID,
				).
					Return(&externalreport.ExternalReport{}, nil).
					Once()
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&externalreport.ExternalReport{},
					false,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
		},
		{
			name: "Missing param report ID",
			args: args{
				customerID: customerID,
				reportID:   "",
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      ErrMissingReportID,
		},
		{
			name: "Interval Server Error",
			args: args{
				customerID: customerID,
				reportID:   reportID,
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.service.On(
					"GetReportConfig",
					mock.AnythingOfType("*gin.Context"),
					reportID,
					customerID,
				).
					Return(nil, errors.New("some error")).
					Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      errors.New("some error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			ctx.Set(common.CtxKeys.Email, email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.reportID},
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)
			ctx.Request = request
			gotError := h.GetReportConfigExternalHandler(ctx)

			if tt.wantErr != nil {
				assert.ErrorContains(t, gotError, tt.wantErr.Error())
			} else {
				assert.NoError(t, gotError)
			}
		})
	}
}

func TestReportHandler_RunReportFromExternalConfig(t *testing.T) {
	email := "test@doit.com"
	customerID := "123"

	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            reportServiceMocks.IReportService
		reportTierService  reportTierServiceMocks.ReportTierService
	}

	testExternalConfig := externalreport.ExternalConfig{}
	testRunReportFromExternalConfigRequest := domainExternalAPI.RunReportFromExternalConfigRequest{
		Config: testExternalConfig,
	}

	tests := []struct {
		name         string
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On(
					"RunReportFromExternalConfig",
					mock.AnythingOfType("*gin.Context"),
					&testExternalConfig,
					customerID,
					email,
				).
					Return(&domainExternalAPI.RunReportResult{}, nil, nil).
					Once()
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&externalreport.ExternalReport{
						Config: &testExternalConfig,
					},
					true,
				).
					Return(nil, nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Requires tier upgrade",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.service.On(
					"RunReportFromExternalConfig",
					mock.AnythingOfType("*gin.Context"),
					&testExternalConfig,
					customerID,
					email,
				).
					Return(&domainExternalAPI.RunReportResult{}, nil, nil).
					Once()
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&externalreport.ExternalReport{
						Config: &testExternalConfig,
					},
					true,
				).
					Return(&reporttier.AccessDeniedCustomReports, nil).
					Once()
			},
		},
		{
			name: "Error - did not pass validation",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&externalreport.ExternalReport{
						Config: &testExternalConfig,
					},
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"RunReportFromExternalConfig",
					mock.AnythingOfType("*gin.Context"),
					&testExternalConfig,
					customerID,
					email,
				).
					Return(nil, []errormsg.ErrorMsg{{Field: "metric", Message: "invalid metric: InvalidMetric"}}, errors.New("error validating config")).
					Once()
			},
			wantedStatus: http.StatusBadRequest,
		},

		{
			name: "Error - internal",
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything).Once()
				f.reportTierService.On(
					"CheckAccessToExternalReport",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&externalreport.ExternalReport{
						Config: &testExternalConfig,
					},
					true,
				).
					Return(nil, nil).
					Once()
				f.service.On(
					"RunReportFromExternalConfig",
					mock.AnythingOfType("*gin.Context"),
					&testExternalConfig,
					customerID,
					email,
				).
					Return(nil, nil, errors.New("internal error")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:            reportServiceMocks.IReportService{},
				loggerProviderMock: loggerMocks.ILogger{},
				reportTierService:  reportTierServiceMocks.ReportTierService{},
			}

			h := &Report{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service:           &fields.service,
				reportTierService: &fields.reportTierService,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			bodyStr, err := json.Marshal(testRunReportFromExternalConfigRequest)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Set("email", email)
			ctx.Set(auth.CtxKeyVerifiedCustomerID, customerID)

			ctx.Request = request

			respond := h.RunReportFromExternalConfig(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("Report.RunReportFromExternalConfig() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

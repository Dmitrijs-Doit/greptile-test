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

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var ginContextMock = mock.AnythingOfType("*gin.Context")

func TestReportTemplateHandler_CreateReportTemplateHandler(t *testing.T) {
	email := "test@doit.com"

	reportTemplateWithVersion := &domain.ReportTemplateWithVersion{
		Template:    &domain.ReportTemplate{},
		LastVersion: &domain.ReportTemplateVersion{},
	}

	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.ReportTemplateService
	}

	type args struct {
		body *domain.ReportTemplateReq
	}

	reportTemplateReq := domain.ReportTemplateReq{
		Name:        "some name",
		Description: "some descr",
		Visibility:  domain.VisibilityPrivate,
		Categories:  []string{"some-category"},
		Cloud:       []string{"some-cloud"},
		Config:      &report.Config{},
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
			name: "successful create",
			args: args{
				body: &reportTemplateReq,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateReportTemplate",
					ginContextMock,
					email,
					&reportTemplateReq,
				).
					Return(reportTemplateWithVersion, nil, nil).
					Once()
			},
			wantedStatus: http.StatusCreated,
		},
		{
			name: "custom attribution error",
			args: args{
				body: &reportTemplateReq,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateReportTemplate",
					ginContextMock,
					email,
					&reportTemplateReq,
				).
					Return(nil, nil, domain.ValidationErr{
						Name: "some custom attribution name",
						Type: domain.CustomAttributionErrType,
					}).
					Once()
			},
			wantErr: true,
		},
		{
			name:    "ShouldBindJSON error",
			args:    args{},
			wantErr: true,
		},
		{
			name: "error invalid report template config",
			args: args{
				body: &reportTemplateReq,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateReportTemplate",
					ginContextMock,
					email,
					&reportTemplateReq,
				).
					Return(nil, []errormsg.ErrorMsg{{Field: "filters", Message: "some error"}}, domain.ErrInvalidReportTemplateConfig).
					Once()
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "error invalid report template",
			args: args{
				body: &reportTemplateReq,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateReportTemplate",
					ginContextMock,
					email,
					&reportTemplateReq,
				).
					Return(nil, []errormsg.ErrorMsg{{Field: "visibility", Message: "some error"}}, domain.ErrInvalidReportTemplate).
					Once()
			},
			wantedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.ReportTemplateService{},
			}

			h := &ReportTemplate{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
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

			ctx.Set("email", email)
			ctx.Request = request

			respond := h.CreateReportTemplateHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("ReportTemplateHandler.CreateReportTemplateHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestReportTemplateHandler_DeleteReportTemplateHandler(t *testing.T) {
	email := "test@doit.com"
	reportTemplateID := "123"

	errorDeletingReportTemplate := errors.New("error deleting report template")

	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.ReportTemplateService
	}

	type args struct {
		reportTemplateID string
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
			name: "successful delete",
			args: args{
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "missing report template id",
			args: args{
				reportTemplateID: "",
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "delete report template error",
			args: args{
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"DeleteReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
				).
					Return(errorDeletingReportTemplate).
					Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.ReportTemplateService{},
			}

			h := &ReportTemplate{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			request := httptest.NewRequest(http.MethodDelete, "/someRequest", nil)

			ctx.Set("email", email)
			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.reportTemplateID},
			}

			ctx.Request = request

			respond := h.DeleteReportTemplateHandler(ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("ReportTemplateHandler.DeleteReportTemplateHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestReportTemplateHandler_GetTemplateData(t *testing.T) {
	type fields struct {
		service *mocks.ReportTemplateService
	}

	tests := []struct {
		name           string
		fields         fields
		isDoitEmployee bool
		on             func(*fields)
		wantErr        bool
	}{
		{
			name: "successful retrieval for non doit employee",
			fields: fields{
				service: &mocks.ReportTemplateService{},
			},
			isDoitEmployee: false,
			on: func(f *fields) {
				f.service.On(
					"GetTemplateData",
					ginContextMock,
					false,
				).
					Return([]domain.ReportTemplate{}, []domain.ReportTemplateVersion{}, nil).
					Once()
			},
			wantErr: false,
		},
		{
			name: "successful retrieval for doit employee",
			fields: fields{
				service: &mocks.ReportTemplateService{},
			},
			isDoitEmployee: true,
			on: func(f *fields) {
				f.service.On(
					"GetTemplateData",
					ginContextMock,
					true,
				).
					Return([]domain.ReportTemplate{}, []domain.ReportTemplateVersion{}, nil).
					Once()
			},
			wantErr: false,
		},
		{
			name: "error retrieving template data",
			fields: fields{
				service: &mocks.ReportTemplateService{},
			},
			isDoitEmployee: false,
			on: func(f *fields) {
				f.service.On(
					"GetTemplateData",
					ginContextMock,
					false,
				).
					Return(nil, nil, errors.New("some error")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Set(common.DoitEmployee, tt.isDoitEmployee)

			h := &ReportTemplate{
				service: tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			tt.fields.service = &mocks.ReportTemplateService{}

			err := h.GetTemplateData(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReportTemplateHandler.GetTemplateData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestReportTemplateHandler_UpdateReportTemplateHandler(t *testing.T) {
	email := "test@doit.com"
	isDoitEmployee := true

	type fields struct {
		loggerProvider loggerMocks.ILogger
		service        *mocks.ReportTemplateService
	}

	type args struct {
		body             *domain.ReportTemplateReq
		reportTemplateID string
	}

	reportTemplateReq := domain.ReportTemplateReq{
		Name:        "some name",
		Description: "some descr",
		Visibility:  domain.VisibilityPrivate,
		Categories:  []string{"some-category"},
		Cloud:       []string{"some-cloud"},
		Config:      &report.Config{},
	}

	reportTemplateID := "123"

	tests := []struct {
		name         string
		args         args
		wantErr      bool
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful update",
			args: args{
				body:             &reportTemplateReq,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"UpdateReportTemplate",
					ginContextMock,
					email,
					isDoitEmployee,
					reportTemplateID,
					&reportTemplateReq,
				).
					Return(&domain.ReportTemplateWithVersion{}, nil, nil).
					Once()
			},
			wantedStatus: http.StatusCreated,
		},
		{
			name: "custom attribution error",
			args: args{
				body:             &reportTemplateReq,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"UpdateReportTemplate",
					ginContextMock,
					email,
					isDoitEmployee,
					reportTemplateID,
					&reportTemplateReq,
				).
					Return(nil, nil, domain.ValidationErr{
						Name: "some custom attribution name",
						Type: domain.CustomAGErrType,
					}).
					Once()
			},
			wantErr: true,
		},
		{
			name: "ShouldBindJSON error",
			args: args{
				reportTemplateID: reportTemplateID,
			},
			wantErr: true,
		},
		{
			name: "error invalid report template config",
			args: args{
				body:             &reportTemplateReq,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"UpdateReportTemplate",
					ginContextMock,
					email,
					isDoitEmployee,
					reportTemplateID,
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
				body:             &reportTemplateReq,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"UpdateReportTemplate",
					ginContextMock,
					email,
					isDoitEmployee,
					reportTemplateID,
					&reportTemplateReq,
				).
					Return(nil, []errormsg.ErrorMsg{{Field: "visibility", Message: "some error"}}, domain.ErrInvalidReportTemplate).
					Once()
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "error updating hidden report template",
			args: args{
				body:             &reportTemplateReq,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"UpdateReportTemplate",
					ginContextMock,
					email,
					isDoitEmployee,
					reportTemplateID,
					&reportTemplateReq,
				).
					Return(nil, nil, service.ErrTemplateIsHidden).
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
				service: mocks.NewReportTemplateService(t),
			}

			h := &ReportTemplate{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				service: fields.service,
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

			ctx.Params = []gin.Param{
				{
					Key:   "id",
					Value: tt.args.reportTemplateID,
				},
			}

			ctx.Set("email", email)
			ctx.Set(common.CtxKeys.DoitEmployee, isDoitEmployee)
			ctx.Request = request

			respond := h.UpdateReportTemplateHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("ReportTemplateHandler.UpdateReportTemplateHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

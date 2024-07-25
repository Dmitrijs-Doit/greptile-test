package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestReportTemplateHandler_RejectReportTemplateHandler(t *testing.T) {
	email := "test@doit.com"

	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.ReportTemplateService
	}

	type args struct {
		body             *domain.RejectReportTemplateRequest
		reportTemplateID string
	}

	comment := "some reject comment"

	rejectRequest := &domain.RejectReportTemplateRequest{
		Comment: comment,
	}

	reportTemplateID := "123"

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		wantedStatus int
		on           func(*fields)
	}{
		{
			name: "successful reject",
			args: args{
				body:             rejectRequest,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"RejectReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
					comment,
				).
					Return(&domain.ReportTemplateWithVersion{}, nil).
					Once()
			},
			wantedStatus: http.StatusCreated,
		},
		{
			name: "ok when rejecting already rejected version",
			args: args{
				body:             rejectRequest,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"RejectReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
					comment,
				).
					Return(nil, service.ErrVersionIsRejected).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "bad request when rejecting approved version",
			args: args{
				body:             rejectRequest,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"RejectReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
					comment,
				).
					Return(nil, service.ErrVersionIsApproved).
					Once()
			},
			wantErr: true,
		},
		{
			name: "bad request when rejecting canceled version",
			args: args{
				body:             rejectRequest,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"RejectReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
					comment,
				).
					Return(nil, service.ErrVersionIsCanceled).
					Once()
			},
			wantErr: true,
		},
		{
			name: "not found when rejecting hidden template",
			args: args{
				body:             rejectRequest,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"RejectReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
					comment,
				).
					Return(nil, service.ErrTemplateIsHidden).
					Once()
			},
			wantErr: true,
		},
		{
			name: "internal error when reject returns some generic error",
			args: args{
				body:             rejectRequest,
				reportTemplateID: reportTemplateID,
			},
			on: func(f *fields) {
				f.service.On(
					"RejectReportTemplate",
					ginContextMock,
					email,
					reportTemplateID,
					comment,
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

			ctx.Params = []gin.Param{
				{Key: "id", Value: tt.args.reportTemplateID},
			}

			ctx.Set("email", email)
			ctx.Request = request

			respond := h.RejectReportTemplateHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("ReportTemplateHandler.RejectReportTemplateHandler() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

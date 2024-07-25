package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/courier/domain"
	"github.com/doitintl/hello/scheduled-tasks/courier/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestCourierHandler_ExportNotificationsToBQHandler(t *testing.T) {
	email := "test@doit.com"

	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.Courier
	}

	type args struct {
		body *domain.ExportNotificationsReq
	}

	startDate := "2023-01-02"

	req := domain.ExportNotificationsReq{
		StartDate: &startDate,
		NotificationIDs: []domain.Notification{
			domain.DailyWeeklyDigestNotification,
			domain.NoClusterOnboardedNotification,
		},
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
			name: "successfully trigger 2 tasks for 2 notificationIDs",
			args: args{
				body: &req,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateExportNotificationsTasks",
					ginContextMock,
					time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
					req.NotificationIDs,
				).
					Return(nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error",
			args: args{
				body: &req,
			},
			on: func(f *fields) {
				f.service.On(
					"CreateExportNotificationsTasks",
					ginContextMock,
					time.Date(2023, time.January, 2, 0, 0, 0, 0, time.UTC),
					req.NotificationIDs,
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
				service:        mocks.NewCourier(t),
			}

			h := &Courier{
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
			request := httptest.NewRequest(http.MethodPost, "/export-url", bodyReader)

			ctx.Set("email", email)
			ctx.Request = request

			respond := h.ExportNotificationsToBQHandler(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
				return
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("Courier.ExportNotificationsToBQHandler() error = %v, wantErr %v", respond, tt.wantErr)
				return
			}
		})
	}
}

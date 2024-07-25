package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	backfillServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/stretchr/testify/mock"

	"github.com/gin-gonic/gin"
)

func TestBackfillHandler_HandleCustomer(t *testing.T) {
	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            backfillServiceMocks.IBackfillService
	}

	customerID := "test-customer-id"

	body := domainBackfill.TaskBodyHandlerCustomer{
		BillingAccountID: "test-billing-account-id",
		PartitionDate:    "2023-01-01",
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
				f.service.On(
					"BackfillCustomer",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&body,
				).Return(nil).Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Internal error",
			on: func(f *fields) {
				f.service.On(
					"BackfillCustomer",
					mock.AnythingOfType("*gin.Context"),
					customerID,
					&body,
				).Return(errors.New("internal error")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				loggerProviderMock: loggerMocks.ILogger{},
				service:            backfillServiceMocks.IBackfillService{},
			}

			h := &Backfill{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			bodyStr, err := json.Marshal(body)
			if err != nil {
				t.Error(err)
			}

			bodyReader := strings.NewReader(string(bodyStr))
			request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)

			ctx.Request = request
			ctx.AddParam("customerID", customerID)

			respond := h.HandleCustomer(ctx)
			status := ctx.Writer.Status()

			if tt.wantedStatus != 0 && tt.wantedStatus != status {
				t.Errorf("got %v, want %v", ctx.Writer.Status(), tt.wantedStatus)
			}

			if (respond != nil) != tt.wantErr {
				t.Errorf("Backfill.HandleCustomer() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

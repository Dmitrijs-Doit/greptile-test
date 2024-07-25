package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/framework/mid"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service/mocks"
)

func TestMarketplaceGCP_StandaloneApprove(t *testing.T) {
	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		service            mocks.MarketplaceIface
	}

	type args struct {
		body StandaloneApprovePayload
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantedBody   string
		wantErr      bool
	}{
		{
			name: "successful request with a valid payload",
			args: args{
				body: StandaloneApprovePayload{
					CustomerID:       "123",
					BillingAccountID: "456",
				},
			},
			on: func(f *fields) {
				f.loggerProviderMock.On("SetLabels", mock.Anything)
				f.service.On("StandaloneApprove", mock.AnythingOfType("*gin.Context"), "123", "456").Return(nil).Once()
			},
			wantedBody:   "",
			wantedStatus: http.StatusOK,
		},
		{
			name: "request with invalid payload missing customer id",
			args: args{
				body: StandaloneApprovePayload{
					CustomerID:       "",
					BillingAccountID: "456",
				},
			},
			wantedBody:   `{"error":"invalid payload"}`,
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "request with invalid payload missing billing account id",
			args: args{
				body: StandaloneApprovePayload{
					CustomerID:       "123",
					BillingAccountID: "",
				},
			},
			wantedBody:   `{"error":"invalid payload"}`,
			wantedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProviderMock: loggerMocks.ILogger{},
				service:            mocks.MarketplaceIface{},
			}
			h := &MarketplaceGCP{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			w := httptest.NewRecorder()
			errMx := mid.Errors()
			app := web.NewTestApp(w, errMx)
			app.Post("/marketplace/gcp/standalone-approve", h.StandaloneApprove)

			rawBody, _ := json.Marshal(tt.args.body)
			body := bytes.NewBuffer(rawBody)

			const url = "/marketplace/gcp/standalone-approve"
			req, _ := http.NewRequest(http.MethodPost, url, body)
			app.ServeHTTP(w, req)

			assert.Equal(t, tt.wantedStatus, w.Code)
			assert.Equal(t, tt.wantedBody, w.Body.String())
		})
	}
}

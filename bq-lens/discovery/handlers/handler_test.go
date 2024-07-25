package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	cloudResourceManagerDomain "github.com/doitintl/cloudresourcemanager/domain"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/service"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestAllCustomersTablesDiscovery(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	type fields struct {
		service     *mocks.DiscoveryService
		loggerMocks *loggerMocks.ILogger
	}

	multiError := []error{errors.New("schedule task creation error")}

	tests := []struct {
		name         string
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successful schedule",
			on: func(f *fields) {
				f.service.On("Schedule", ctx).
					Return(nil, nil).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "schedule failed, error",
			on: func(f *fields) {
				f.service.On("Schedule", ctx).
					Return(nil, errors.New("schedule error")).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "schedule failed, multi-error",
			on: func(f *fields) {
				f.service.On("Schedule", ctx).
					Return(multiError, nil).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Errorf", mock.AnythingOfType("string"), multiError)
			},
			wantedStatus: http.StatusMultiStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service:     mocks.NewDiscoveryService(t),
				loggerMocks: loggerMocks.NewILogger(t),
			}

			h := &TableDiscovery{
				service: fields.service,
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.loggerMocks
				},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Request = request

			respond := h.AllCustomersTablesDiscovery(ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("AllCustomersTablesDiscovery() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestSingleCustomerTablesDiscovery(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	customerID := "test-customer-1"
	validPayload := `{"projects": [
    {
      "id": "project-1",
      "number": 123456789,
      "name": "project-1",
      "lifecycleState": "ACTIVE",
      "labels": {
        "env": "production",
        "team": "team-a"
      }
    }]}`

	invalidPayload := `{"projects": []}`

	input := service.TablesDiscoveryPayload{
		Projects: []*cloudResourceManagerDomain.Project{
			{
				ID:             "project-1",
				Number:         123456789,
				Name:           "project-1",
				LifecycleState: "ACTIVE",
				Labels: map[string]string{
					"env":  "production",
					"team": "team-a",
				},
			},
		},
	}

	discoveryError := errors.New("discovery error")

	type fields struct {
		service     *mocks.DiscoveryService
		loggerMocks *loggerMocks.ILogger
	}

	type args struct {
		body io.Reader
	}

	tests := []struct {
		name         string
		args args
		on           func(*fields)
		wantedStatus int
	}{
		{
			name: "successful discovery",
			on: func(f *fields) {
				f.service.On("TablesDiscovery", ctx, customerID, input).
					Return(nil).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			args: args{body: strings.NewReader(validPayload)},
			wantedStatus: http.StatusOK,
		},
		{
			name: "no payload bad request",
			on: func(f *fields) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "field validation bad request",
			on: func(f *fields) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			args:         args{body: strings.NewReader(invalidPayload)},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "discovery failed",
			on: func(f *fields) {
				f.service.On("TablesDiscovery", ctx, customerID, input).
					Return(discoveryError).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Error", discoveryError).Once()
			},
			args: args{body: strings.NewReader(validPayload)},
			wantedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service:     mocks.NewDiscoveryService(t),
				loggerMocks: loggerMocks.NewILogger(t),
			}

			h := &TableDiscovery{
				service: fields.service,
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.loggerMocks
				},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", tt.args.body)

			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
			}
			ctx.Request = request

			err := h.SingleCustomerTablesDiscovery(ctx)
			if err == nil {
				assert.Equal(t, tt.wantedStatus, recorder.Code)
			} else {
				var reqErr *web.Error

				if errors.As(err, &reqErr) {
					assert.Equal(t, tt.wantedStatus, reqErr.Status)
				} else {
					t.Fatalf("Unexpected error type: %v", err)
				}
			}
		})
	}
}

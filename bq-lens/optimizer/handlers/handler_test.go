package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestSingleCustomerOptimizer(t *testing.T) {
	customerID := "test-customer-1"

	validPayload := `{
						"billingProjectWithReservation": [{"project": "doitintl-cmp-dev", "location": "us-central1"}],
						"discount": 1.0
					}`
	validPayloadStruct := domain.Payload{
		BillingProjectWithReservation: []domain.BillingProjectWithReservation{
			{
				Project:  "doitintl-cmp-dev",
				Location: "us-central1",
			}},
		Discount: 1,
	}
	type fields struct {
		service     *mocks.OptimizerService
		loggerMocks *loggerMocks.ILogger
	}

	optimizerError := errors.New("optimizer error")

	type args struct {
		body io.Reader
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields, *gin.Context)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successful pass with payload",
			args: args{body: strings.NewReader(validPayload)},
			on: func(f *fields, ctx *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Infof", "Optimizer for customer '%s' started", customerID).Once()

				f.service.On("SingleCustomerOptimizer",
					ctx,
					customerID,
					validPayloadStruct).
					Return(nil).Once()

				f.loggerMocks.On("Infof", "Optimizer for customer '%s' completed with total duration of '%v' seconds", customerID, mock.AnythingOfType("float64")).Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "optimizer failed",
			args: args{body: strings.NewReader(validPayload)},
			on: func(f *fields, ctx *gin.Context) {
				f.service.On("SingleCustomerOptimizer", ctx, customerID, validPayloadStruct).
					Return(optimizerError).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Infof", "Optimizer for customer '%s' started", customerID).Once()
				f.loggerMocks.On("Infof", "Optimizer for customer '%s' completed with total duration of '%v' seconds", customerID, mock.AnythingOfType("float64")).Once()
				f.loggerMocks.On("Error", optimizerError).Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "no payload bad request",
			on: func(f *fields, ctx *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "failed due to missing project",
			args: args{body: strings.NewReader(`
				{
					"billingProjectWithReservation": [{"project": "", "location": "us-central1"}],
					"discount": 1.0,
				}`)},
			on: func(f *fields, ctx *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "failed due to missing discount",
			args: args{body: strings.NewReader(`
				{"billingProjectWithReservation": [{"project": "", "location": "us-central1"}]}`)},
			on: func(f *fields, ctx *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
		},
		{
			name: "failed due to missing field on periodTotalPriceMapping",
			args: args{body: strings.NewReader(`
				{
					"billingProjectWithReservation": [{"project": "", "location": "us-central1"}],
					"discount": 1.0
				}`)},
			on: func(f *fields, ctx *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{
				service:     mocks.NewOptimizerService(t),
				loggerMocks: loggerMocks.NewILogger(t),
			}

			h := &Optimizer{
				service: fields.service,
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.loggerMocks
				},
			}

			if tt.on != nil {
				tt.on(&fields, ctx)
			}

			request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/optimizer/%s", customerID), tt.args.body)

			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
			}
			ctx.Request = request

			err := h.SingleCustomerOptimizer(ctx)
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

func TestAllCustomersOptimizer(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	type fields struct {
		service     *mocks.OptimizerService
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
				service:     mocks.NewOptimizerService(t),
				loggerMocks: loggerMocks.NewILogger(t),
			}

			h := &Optimizer{
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

			respond := h.AllCustomersOptimizer(ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("AllCustomersOptimizer() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

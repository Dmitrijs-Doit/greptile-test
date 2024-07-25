package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestPricebook_SetEditionPricebook(t *testing.T) {
	type fields struct {
		log     loggerMocks.ILogger
		service mocks.Pricebook
	}

	tests := []struct {
		name         string
		on           func(*fields, *gin.Context)
		wantedStatus int
	}{
		{
			name: "successful pass",
			on: func(f *fields, ctx *gin.Context) {
				f.service.On("SetEditionPrices", ctx).Return(nil)
				f.service.On("SetLegacyFlatRatePrices", ctx).Return(nil)
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error in edition prices",
			on: func(f *fields, ctx *gin.Context) {
				f.service.On("SetEditionPrices", ctx).Return(errors.New("some error"))
			},
			wantedStatus: http.StatusInternalServerError,
		},
		{
			name: "error in flat rate prices",
			on: func(f *fields, ctx *gin.Context) {
				f.service.On("SetEditionPrices", ctx).Return(nil)
				f.service.On("SetLegacyFlatRatePrices", ctx).Return(errors.New("some error"))
			},
			wantedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)

			fields := fields{}
			if tt.on != nil {
				tt.on(&fields, ctx)
			}

			h := &Pricebook{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.log
				},
				conn:    new(connection.Connection),
				service: &fields.service,
			}

			request := httptest.NewRequest(http.MethodPost, "/edition-pricebook", nil)
			ctx.Request = request

			err := h.SetEditionPricebook(ctx)
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

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"gotest.tools/assert"

	mockService "github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestOnboardHandler_Onboard(t *testing.T) {
	mockSinkID := "google-cloud-114075288177071352357"

	type fields struct {
		loggerMocks *loggerMocks.ILogger
		service     *mockService.OnboardService
	}

	tests := []struct {
		name         string
		on           func(*fields, *gin.Context)
		body         map[string]interface{}
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Test Onboard happy path",
			body: map[string]interface{}{
				"HandleSpecificSink": mockSinkID,
			},
			on: func(f *fields, _ *gin.Context) {
				f.service.On("HandleSpecificSink", mock.Anything, mockSinkID).Return(nil)
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name: "Test Onboard fail",
			body: map[string]interface{}{
				"HandleSpecificSink": mockSinkID,
			},
			on: func(f *fields, _ *gin.Context) {
				f.service.On("HandleSpecificSink", mock.Anything, mockSinkID).Return(fmt.Errorf("mockError"))
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
		{
			name: "Test Onboard missing HandleSpecificSink",
			body: map[string]interface{}{},
			on: func(f *fields, _ *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedStatus: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name: "Test Onboard DontRun",
			body: map[string]interface{}{
				"HandleSpecificSink": mockSinkID,
				"DontRun":            true,
			},
			on: func(f *fields, _ *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Info", "Onboarding did not run, dontRun flag set").Once()
			},
			wantedStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name: "Test Onboard RemoveData",
			body: map[string]interface{}{
				"HandleSpecificSink": mockSinkID,
				"RemoveData":         true,
			},
			on: func(f *fields, _ *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.service.On("RemoveData", mock.Anything, mockSinkID).Return(nil)
			},
			wantedStatus: http.StatusOK,
			wantErr:      false,
		},
		{
			name: "Test Onboard fail RemoveData",
			body: map[string]interface{}{
				"HandleSpecificSink": mockSinkID,
				"RemoveData":         true,
			},
			on: func(f *fields, _ *gin.Context) {
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.service.On("RemoveData", mock.Anything, mockSinkID).Return(fmt.Errorf("mockError"))
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest("POST", "/onboard", nil)

			jsonbytes, err := json.Marshal(tt.body)
			if err != nil {
				panic(err)
			}

			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(jsonbytes))

			fields := fields{
				loggerMocks: loggerMocks.NewILogger(t),
				service:     mockService.NewOnboardService(t),
			}

			h := &OnboardHandler{
				service: fields.service,
				loggerProvider: func(_ context.Context) logger.ILogger {
					return fields.loggerMocks
				},
			}

			if tt.on != nil {
				tt.on(&fields, ctx)
			}

			err = h.Onboard(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("OnboardHandler.Onboard() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				assert.Equal(t, tt.wantedStatus, recorder.Code)
			} else {
				var reqErr *web.Error

				if errors.As(err, &reqErr) {
					assert.Equal(t, tt.wantedStatus, reqErr.Status, "Unexpected status code")
				} else {
					t.Fatalf("Unexpected error type: %v", err)
				}
			}
		})
	}
}

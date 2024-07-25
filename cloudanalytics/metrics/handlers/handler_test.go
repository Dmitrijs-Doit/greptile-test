package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"github.com/zeebo/assert"
)

func getContext() *gin.Context {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	return ctx
}

func TestMetric_DeleteMetricsHandler(t *testing.T) {
	ctx := getContext()

	type fields struct {
		loggerProvider logger.Provider
		service        *serviceMock.IMetricsService
	}

	type args struct {
		ctx *gin.Context
	}

	var validRequestMap = map[string]interface{}{
		"ids": []string{"metric1", "metric2"},
	}

	validBody, err := json.Marshal(validRequestMap)
	if err != nil {
		t.Fatal(err)
	}

	var invalidRequestMapEmptyArray = map[string]interface{}{
		"ids": []string{},
	}

	invalidBodyEmptyArray, err := json.Marshal(invalidRequestMapEmptyArray)
	if err != nil {
		t.Fatal(err)
	}

	var invalidRequestMapNoArray = map[string]interface{}{}

	invalidBodyNoArray, err := json.Marshal(invalidRequestMapNoArray)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		requestBody io.ReadCloser
		on          func(*fields)
		expectedErr error
	}{
		{
			name: "successfully delete metrics",
			args: args{
				ctx,
			},
			wantErr:     false,
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			on: func(f *fields) {
				f.service.On("DeleteMany", ctx, service.DeleteMetricsRequest{IDs: []string{"metric1", "metric2"}}).Return(nil)
			},
		},
		{
			name: "error empty ids list",
			args: args{
				ctx,
			},
			wantErr:     true,
			requestBody: io.NopCloser(bytes.NewReader(invalidBodyEmptyArray)),
		},
		{
			name: "error no ids list",
			args: args{
				ctx,
			},
			wantErr:     true,
			requestBody: io.NopCloser(bytes.NewReader(invalidBodyNoArray)),
		},
		{
			name: "error metric not found",
			args: args{
				ctx,
			},
			wantErr:     true,
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			expectedErr: web.NewRequestError(service.CustomMetricNotFoundError{ID: "metric1"}, http.StatusNotFound),
			on: func(f *fields) {
				f.service.On("DeleteMany", ctx, service.DeleteMetricsRequest{IDs: []string{"metric1", "metric2"}}).
					Return(service.CustomMetricNotFoundError{ID: "metric1"})
			},
		},
		{
			name: "error cannot delete preset",
			args: args{
				ctx,
			},
			wantErr:     true,
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			expectedErr: web.NewRequestError(service.PresetMetricsCannotBeDeletedError{ID: "metric1"}, http.StatusForbidden),
			on: func(f *fields) {
				f.service.On("DeleteMany", ctx, service.DeleteMetricsRequest{IDs: []string{"metric1", "metric2"}}).
					Return(service.PresetMetricsCannotBeDeletedError{ID: "metric1"})
			},
		},
		{
			name: "error cannot delete used metric",
			args: args{
				ctx,
			},
			wantErr:     true,
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			expectedErr: web.NewRequestError(service.MetricIsInUseError{ID: "metric1"}, http.StatusForbidden),
			on: func(f *fields) {
				f.service.On("DeleteMany", ctx, service.DeleteMetricsRequest{IDs: []string{"metric1", "metric2"}}).
					Return(service.MetricIsInUseError{ID: "metric1"})
			},
		},
		{
			name: "internal server error",
			args: args{
				ctx,
			},
			wantErr:     true,
			requestBody: io.NopCloser(bytes.NewReader(validBody)),
			expectedErr: web.NewRequestError(errors.New("error"), http.StatusInternalServerError),
			on: func(f *fields) {
				f.service.On("DeleteMany", ctx, service.DeleteMetricsRequest{IDs: []string{"metric1", "metric2"}}).
					Return(errors.New("error"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&serviceMock.IMetricsService{},
			}

			h := &Metric{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			ctx.Request.Body = tt.requestBody

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if response := h.DeleteMetricsHandler(tt.args.ctx); (response != nil) != tt.wantErr {
				t.Errorf("Metric.DeleteMetricsHandler() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				if tt.expectedErr != nil {
					assert.Equal(t, tt.expectedErr, response)
				}
			}
		})
	}
}

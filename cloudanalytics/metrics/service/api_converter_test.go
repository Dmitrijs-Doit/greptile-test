package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metricsDALMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/mocks"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestToInternal(t *testing.T) {
	type fields struct {
		metricsDAL         *metricsDALMocks.Metrics
		loggerProviderMock loggerMocks.ILogger
	}

	customerID := "some customer id"

	customMetricID := "CustomMetricID"
	dalError := errors.New("dal error")
	externalMetricOK := &metrics.ExternalMetric{
		Type:  metrics.ExternalMetricTypeCustom,
		Value: customMetricID,
	}

	tests := []struct {
		name                 string
		externalMetric       *metrics.ExternalMetric
		fields               fields
		on                   func(*fields)
		wantMetric           *metrics.InternalMetricParameters
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "Invalid metric type",
			externalMetric: &metrics.ExternalMetric{
				Type: "INVALID",
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric type: INVALID"}},
			wantErr:              true,
		},
		{
			name:                 "metric is not provided",
			externalMetric:       nil,
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "metric can not be null"}},
			wantErr:              true,
		},
		{
			name: "Convert basic metric - ok",
			externalMetric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeBasic,
				Value: string(metrics.ExternalBasicMetricCost),
			},
			wantMetric: &metrics.InternalMetricParameters{
				Metric: makeMetricPtr(report.MetricCost),
			},
		},
		{
			name: "Convert basic metric - error",
			externalMetric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeBasic,
				Value: "INVALID",
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid basic metric value: INVALID"}},
			wantErr:              true,
		},
		{
			name:           "Convert custom metric - not found",
			externalMetric: externalMetricOK,
			on: func(f *fields) {
				f.metricsDAL.
					On("GetRef", testutils.ContextBackgroundMock, externalMetricOK.Value).
					Return(&firestore.DocumentRef{ID: customMetricID}).
					Once()
				f.metricsDAL.
					On("Exists", testutils.ContextBackgroundMock, externalMetricOK.Value).
					Return(false, nil).
					Once()
				f.loggerProviderMock.
					On("Warning", CustomMetricNotFoundError{customMetricID}.Error()).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "custom metric not found: CustomMetricID"}},
			wantErr:              true,
		},
		{
			name:           "Convert custom metric - internal error",
			externalMetric: externalMetricOK,
			on: func(f *fields) {
				f.metricsDAL.
					On("GetRef", testutils.ContextBackgroundMock, externalMetricOK.Value).
					Return(&firestore.DocumentRef{ID: customMetricID}).
					Once()
				f.metricsDAL.
					On("Exists", testutils.ContextBackgroundMock, externalMetricOK.Value).
					Return(false, dalError).
					Once()
				f.loggerProviderMock.
					On("Error",
						CheckCustomMetricExistsError{customMetricID, dalError}.Error()).Once()
			},
			wantErr: true,
		},
		{
			name:           "Convert custom metric - ok",
			externalMetric: externalMetricOK,
			on: func(f *fields) {
				f.metricsDAL.
					On("GetRef", testutils.ContextBackgroundMock, externalMetricOK.Value).
					Return(&firestore.DocumentRef{ID: "CustomMetricID"}).
					Once()
				f.metricsDAL.
					On("Exists", testutils.ContextBackgroundMock, externalMetricOK.Value).
					Return(true, nil).
					Once()
			},
			wantMetric: &metrics.InternalMetricParameters{
				Metric:       makeMetricPtr(report.MetricCustom),
				CustomMetric: &firestore.DocumentRef{ID: "CustomMetricID"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				metricsDAL:         &metricsDALMocks.Metrics{},
				loggerProviderMock: loggerMocks.ILogger{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &MetricsService{
				metricsDAL: tt.fields.metricsDAL,
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &tt.fields.loggerProviderMock
				},
			}

			gotMetric, gotValidationErrors, gotErr := s.ToInternal(ctx, customerID, tt.externalMetric)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ToInternal() error = %v, wantErr %v", gotErr, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, gotValidationErrors)

			assert.Equal(t, tt.wantMetric, gotMetric)
		})
	}
}

func TestToExternal(t *testing.T) {
	tests := []struct {
		name                 string
		params               *metrics.InternalMetricParameters
		wantMetric           *metrics.ExternalMetric
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "Invalid metric type",
			params: &metrics.InternalMetricParameters{
				Metric: makeMetricPtr(999999),
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric type: 999999"}},
			wantErr:              true,
		},
		{
			name: "Convert basic metric - ok",
			params: &metrics.InternalMetricParameters{
				Metric: makeMetricPtr(report.MetricCost),
			},
			wantMetric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeBasic,
				Value: "cost",
			},
		},
		{
			name: "Convert custom metric - ok",
			params: &metrics.InternalMetricParameters{
				Metric:       makeMetricPtr(report.MetricCustom),
				CustomMetric: &firestore.DocumentRef{ID: "MyCustomMetricID"},
			},
			wantMetric: &metrics.ExternalMetric{
				Type:  metrics.ExternalMetricTypeCustom,
				Value: "MyCustomMetricID",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &MetricsService{}

			gotMetric, gotValidationErrors, gotErr := s.ToExternal(tt.params)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("ToExternal() error = %v, wantErr %v", gotErr, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, gotValidationErrors)

			assert.Equal(t, tt.wantMetric, gotMetric)
		})
	}
}

func makeMetricPtr(m report.Metric) *report.Metric {
	return &m
}

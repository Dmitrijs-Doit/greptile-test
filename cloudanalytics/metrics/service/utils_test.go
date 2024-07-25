package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	metricsDALMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/mocks"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	reportsDALMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMetricsService_checkMetricsNotPreset(t *testing.T) {
	type fields struct {
		metricsDAL *metricsDALMocks.Metrics
		reportsDAL *reportsDALMocks.Reports
	}

	type args struct {
		ctx context.Context
		IDs []string
	}

	ctx := context.Background()

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		on          func(f *fields)
		expectedErr error
	}{
		{
			name: "success check metric not preset",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: false,
			on: func(f *fields) {
				f.metricsDAL.On("GetCustomMetric", ctx, "metric1").Return(&metrics.CalculatedMetric{}, nil)
				f.metricsDAL.On("GetCustomMetric", ctx, "metric2").Return(&metrics.CalculatedMetric{}, nil)
			},
		},
		{
			name: "error metric not found",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.metricsDAL.On("GetCustomMetric", ctx, "metric1").Return(nil, status.Error(codes.NotFound, "error"))
			},
			expectedErr: CustomMetricNotFoundError{"metric1"},
		},
		{
			name: "error getting metric",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.metricsDAL.On("GetCustomMetric", ctx, "metric1").Return(nil, errors.New("error"))
			},
			expectedErr: errors.New("error"),
		},
		{
			name: "error metric is preset",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.metricsDAL.On("GetCustomMetric", ctx, "metric1").Return(&metrics.CalculatedMetric{Type: "preset"}, nil)
			},
			expectedErr: PresetMetricsCannotBeDeletedError{"metric1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				&metricsDALMocks.Metrics{},
				&reportsDALMocks.Reports{},
			}
			s := &MetricsService{
				metricsDAL: tt.fields.metricsDAL,
				reportsDAL: tt.fields.reportsDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.checkMetricsNotPreset(tt.args.ctx, tt.args.IDs); (err != nil) != tt.wantErr {
				t.Errorf("MetricsService.checkMetricsNotPreset() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestMetricsService_checkMetricsNotInUse(t *testing.T) {
	type fields struct {
		metricsDAL *metricsDALMocks.Metrics
		reportsDAL *reportsDALMocks.Reports
	}

	type args struct {
		ctx context.Context
		IDs []string
	}

	ctx := context.Background()

	var (
		metric1Ref = &firestore.DocumentRef{ID: "metric1"}
		metric2Ref = &firestore.DocumentRef{ID: "metric2"}
	)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		on          func(f *fields)
		expectedErr error
	}{
		{
			name: "success check metric not in use",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: false,
			on: func(f *fields) {
				f.metricsDAL.On("GetRef", ctx, "metric1").Return(metric1Ref)
				f.reportsDAL.On("GetByMetricRef", ctx, metric1Ref).Return([]*report.Report{}, nil)
				f.metricsDAL.On("GetRef", ctx, "metric2").Return(metric2Ref)
				f.reportsDAL.On("GetByMetricRef", ctx, metric2Ref).Return([]*report.Report{}, nil)
			},
		},
		{
			name: "error getting reports",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.metricsDAL.On("GetRef", ctx, "metric1").Return(metric1Ref)
				f.reportsDAL.On("GetByMetricRef", ctx, metric1Ref).Return(nil, errors.New("error"))
			},
			expectedErr: errors.New("error"),
		},
		{
			name: "error metric in use",
			args: args{
				ctx,
				[]string{"metric1", "metric2"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.metricsDAL.On("GetRef", ctx, "metric1").Return(metric1Ref)
				f.reportsDAL.On("GetByMetricRef", ctx, metric1Ref).Return([]*report.Report{{}}, nil)
			},
			expectedErr: MetricIsInUseError{"metric1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				&metricsDALMocks.Metrics{},
				&reportsDALMocks.Reports{},
			}
			s := &MetricsService{
				metricsDAL: tt.fields.metricsDAL,
				reportsDAL: tt.fields.reportsDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.checkMetricsNotInUse(tt.args.ctx, tt.args.IDs); (err != nil) != tt.wantErr {
				t.Errorf("MetricsService.checkMetricsNotInUse() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

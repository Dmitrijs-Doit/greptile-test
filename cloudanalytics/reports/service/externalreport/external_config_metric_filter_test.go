package externalreport

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	metricsServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/mocks"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestExternalReport_NewExternalMetricFilterFromInternal(t *testing.T) {
	type fields struct {
		metricsService *metricsServiceMocks.IMetricsService
	}

	invalidMetric := domainReport.Metric(999999)
	validMetric := domainReport.MetricCost

	type args struct {
		configMetricFilter []*domainReport.ConfigMetricFilter
		customMetric       *firestore.DocumentRef
		extendedMetric     string
	}

	extendedMetric := "flexsave"

	emptyExtendedMetric := ""

	customMetric := firestore.DocumentRef{
		ID: "123",
	}

	tests := []struct {
		name                 string
		fields               fields
		args                 args
		on                   func(*fields)
		want                 *domainExternalReport.ExternalConfigMetricFilter
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "Too many filters",
			args: args{
				configMetricFilter: []*domainReport.ConfigMetricFilter{
					{}, {},
				},
				extendedMetric: "",
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.ExternalConfigFilterField, Message: "unsupported multiple metric filters. Number of filters: 2"}},
		},
		{
			name: "Invalid metric",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(invalidMetric).(*domainReport.Metric),
						ExtendedMetric: &emptyExtendedMetric,
					},
				).Return(nil, []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: metrics.ErrInvalidMetricMsg}}, nil).Once()
			},
			args: args{
				configMetricFilter: []*domainReport.ConfigMetricFilter{
					{
						Metric: invalidMetric,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: metrics.ErrInvalidMetricMsg}},
		},
		{
			name: "Invalid operator",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(validMetric).(*domainReport.Metric),
						ExtendedMetric: &emptyExtendedMetric,
					},
				).Return(&metrics.ExternalMetric{Type: metrics.ExternalMetricTypeBasic, Value: string(metrics.ExternalBasicMetricCost)}, nil, nil).Once()
			},
			args: args{
				configMetricFilter: []*domainReport.ConfigMetricFilter{
					{
						Metric:   validMetric,
						Operator: domainReport.MetricFilter("INVALID"),
					},
				},
				extendedMetric: "",
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.MetricFilterField, Message: "unsupported metric filter operation: INVALID"}},
		},
		{
			name: "Happy path",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(validMetric).(*domainReport.Metric),
						ExtendedMetric: &emptyExtendedMetric,
					},
				).Return(&metrics.ExternalMetric{Type: metrics.ExternalMetricTypeBasic, Value: string(metrics.ExternalBasicMetricCost)}, nil, nil).Once()
			},
			args: args{
				configMetricFilter: []*domainReport.ConfigMetricFilter{
					{
						Metric:   validMetric,
						Operator: domainReport.MetricFilterGreaterThan,
						Values:   []float64{100},
					},
				},
			},
			want: &domainExternalReport.ExternalConfigMetricFilter{
				Metric: metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeBasic,
					Value: string(metrics.ExternalBasicMetricCost),
				},
				Operator: domainExternalReport.ExternalMetricFilterGreaterThan,
				Values:   []float64{100},
			},
		},
		{
			name: "Custom metric filter",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(domainReport.MetricCustom).(*domainReport.Metric),
						CustomMetric:   &customMetric,
						ExtendedMetric: &emptyExtendedMetric,
					},
				).Return(&metrics.ExternalMetric{Type: metrics.ExternalMetricTypeCustom, Value: string(customMetric.ID)}, nil, nil).Once()
			},
			args: args{
				configMetricFilter: []*domainReport.ConfigMetricFilter{
					{
						Metric:   domainReport.MetricCustom,
						Operator: domainReport.MetricFilterGreaterThan,
						Values:   []float64{100},
					},
				},
				customMetric: &customMetric,
			},
			want: &domainExternalReport.ExternalConfigMetricFilter{
				Metric: metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeCustom,
					Value: "123",
				},
				Operator: domainExternalReport.ExternalMetricFilterGreaterThan,
				Values:   []float64{100},
			},
		},
		{
			name: "Extended metric filter",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric:         toPointer(domainReport.MetricExtended).(*domainReport.Metric),
						CustomMetric:   nil,
						ExtendedMetric: &extendedMetric,
					},
				).Return(&metrics.ExternalMetric{Type: metrics.ExternalMetricTypeExtended, Value: string(extendedMetric)}, nil, nil).Once()
			},
			args: args{
				configMetricFilter: []*domainReport.ConfigMetricFilter{
					{
						Metric:   domainReport.MetricExtended,
						Operator: domainReport.MetricFilterGreaterThan,
						Values:   []float64{100},
					},
				},
				extendedMetric: extendedMetric,
			},
			want: &domainExternalReport.ExternalConfigMetricFilter{
				Metric: metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeExtended,
					Value: extendedMetric,
				},
				Operator: domainExternalReport.ExternalMetricFilterGreaterThan,
				Values:   []float64{100},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricsService: &metricsServiceMocks.IMetricsService{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				metricsService: tt.fields.metricsService,
			}

			got, validationErrors, err := s.NewExternalMetricFilterFromInternal(
				tt.args.configMetricFilter,
				tt.args.customMetric,
				&tt.args.extendedMetric,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.NewExternalMetricFilterFromInternal() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExternalReport_ExternalConfigMetricsFilterToInternal(t *testing.T) {
	type fields struct {
		metricsService *metricsServiceMocks.IMetricsService
	}

	ctx := context.Background()

	customerID := "some customer id"

	tests := []struct {
		name                       string
		fields                     fields
		on                         func(*fields)
		externalConfigMetricFilter *domainExternalReport.ExternalConfigMetricFilter
		want                       *domainReport.ConfigMetricFilter
		wantValidationErrors       []errormsg.ErrorMsg
		wantErr                    bool
	}{
		{
			name: "Metrics DAL error",
			externalConfigMetricFilter: &domainExternalReport.ExternalConfigMetricFilter{
				Metric: metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeCustom,
					Value: "MyCustomMetricID",
				},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					&metrics.ExternalMetric{Type: metrics.ExternalMetricTypeCustom, Value: "MyCustomMetricID"},
				).Return(nil, nil, errors.New("DAL error")).Once()
			},
			wantErr: true,
		},
		{
			name: "Multiple fields are invalid",
			externalConfigMetricFilter: &domainExternalReport.ExternalConfigMetricFilter{
				Metric: metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeCustom,
					Value: "MyCustomMetricID",
				},
				Operator: domainExternalReport.ExternalMetricFilter("INVALID-OPERATOR"),
				Values:   []float64{1, 2, 3, 4, 5},
			},
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					&metrics.ExternalMetric{Type: metrics.ExternalMetricTypeCustom, Value: "MyCustomMetricID"},
				).Return(nil, []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "metric not found"}}, nil).Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{Field: metrics.MetricField, Message: "metric not found"},
				{Field: domainExternalReport.ExternalConfigMetricFilterField, Message: "unsupported metric filter operation: INVALID-OPERATOR"},
				{Field: domainExternalReport.ExternalConfigMetricFilterField, Message: "invalid number of values: 5"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricsService: &metricsServiceMocks.IMetricsService{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &Service{
				metricsService: tt.fields.metricsService,
			}

			got, validationErrors, err := s.ExternalConfigMetricsFilterToInternal(
				ctx,
				customerID,
				tt.externalConfigMetricFilter,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("external_report.NewExternalMetricFilterFromInternal() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			assert.Equal(t, tt.want, got)
		})
	}
}

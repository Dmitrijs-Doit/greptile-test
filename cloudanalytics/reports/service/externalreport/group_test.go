package externalreport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	metricsServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/mocks"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestGroupToFilter(t *testing.T) {
	type fields struct {
		metricsService *metricsServiceMocks.IMetricsService
	}

	ctx := context.Background()

	customerID := "some customer id"

	tests := []struct {
		name                 string
		fields               fields
		on                   func(*fields)
		group                *domainExternalReport.Group
		filter               *domainReport.ConfigFilter
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "The metric is invalid",
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					&metrics.ExternalMetric{Type: "INVALID", Value: ""},
				).Return(nil, []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric type: INVALID"}}, nil).Once()
			},
			group: &domainExternalReport.Group{
				Limit: &domainExternalReport.Limit{
					Value: 10,
					Sort:  toPointer(domainReport.SortDesc).(*domainReport.Sort),
					Metric: &metrics.ExternalMetric{
						Type: metrics.ExternalMetricType("INVALID"),
					},
				},
			},
			filter:               &domainReport.ConfigFilter{},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric type: INVALID"}},
		},
		{
			name: "The sorting order is invalid",
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					&metrics.ExternalMetric{Type: "basic", Value: "cost"},
				).Return(&metrics.InternalMetricParameters{
					Metric: toPointer(domainReport.MetricCost).(*domainReport.Metric),
				}, nil, nil).Once()
			},
			group: &domainExternalReport.Group{
				Limit: &domainExternalReport.Limit{
					Value: 10,
					Sort:  toPointer(domainReport.Sort("INVALID")).(*domainReport.Sort),
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					},
				},
			},
			filter:               &domainReport.ConfigFilter{},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.SortField, Message: "invalid limit sort: INVALID"}},
		},
		{
			name: "Happy path",
			on: func(f *fields) {
				f.metricsService.On("ToInternal",
					ctx,
					customerID,
					&metrics.ExternalMetric{Type: "basic", Value: "cost"},
				).Return(&metrics.InternalMetricParameters{
					Metric: toPointer(domainReport.MetricCost).(*domainReport.Metric),
				}, nil, nil).Once()
			},
			group: &domainExternalReport.Group{
				Limit: &domainExternalReport.Limit{
					Value: 10,
					Sort:  toPointer(domainReport.SortAtoZ).(*domainReport.Sort),
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					},
				},
			},
			filter: &domainReport.ConfigFilter{},
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

			validationErrors, err := s.GroupToFilter(
				ctx,
				customerID,
				tt.group,
				tt.filter,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupToFilter error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestGroupLoadFilter(t *testing.T) {
	type fields struct {
		metricsService *metricsServiceMocks.IMetricsService
	}

	tests := []struct {
		name                 string
		fields               fields
		on                   func(*fields)
		group                *domainExternalReport.Group
		filter               *domainReport.ConfigFilter
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name:   "Empty filter",
			group:  &domainExternalReport.Group{},
			filter: &domainReport.ConfigFilter{},
		},
		{
			name: "The metric conversion fails",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric: toPointer(domainReport.Metric(99999999)).(*domainReport.Metric),
					},
				).Return(nil, []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric type: 99999999"}}, nil).Once()
			},
			group: &domainExternalReport.Group{
				Limit: &domainExternalReport.Limit{
					Value: 10,
					Sort:  toPointer(domainReport.SortAsc).(*domainReport.Sort),
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					},
				},
			},
			filter: &domainReport.ConfigFilter{
				LimitMetric: toPointer(99999999).(*int),
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: metrics.MetricField, Message: "invalid metric type: 99999999"}},
		},
		{
			name: "Invalid limit order",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric: toPointer(domainReport.MetricUsage).(*domainReport.Metric),
					},
				).Return(&metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeBasic,
					Value: string(metrics.ExternalBasicMetricUsage),
				}, nil, nil).Once()
			},
			group: &domainExternalReport.Group{
				Limit: &domainExternalReport.Limit{
					Value: 10,
					Sort:  toPointer(domainReport.SortAtoZ).(*domainReport.Sort),
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					},
				},
			},
			filter: &domainReport.ConfigFilter{
				Limit:       100,
				LimitOrder:  toPointer("INVALID").(*string),
				LimitMetric: toPointer(int(domainReport.MetricUsage)).(*int),
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: domainExternalReport.SortField, Message: "invalid limit sort: INVALID"}},
		},
		{
			name: "Happy path",
			on: func(f *fields) {
				f.metricsService.On("ToExternal",
					&metrics.InternalMetricParameters{
						Metric: toPointer(domainReport.MetricUsage).(*domainReport.Metric),
					},
				).Return(&metrics.ExternalMetric{
					Type:  metrics.ExternalMetricTypeBasic,
					Value: string(metrics.ExternalBasicMetricUsage),
				}, nil, nil).Once()
			},
			group: &domainExternalReport.Group{
				Limit: &domainExternalReport.Limit{
					Value: 10,
					Sort:  toPointer(domainReport.SortAtoZ).(*domainReport.Sort),
					Metric: &metrics.ExternalMetric{
						Type:  metrics.ExternalMetricTypeBasic,
						Value: string(metrics.ExternalBasicMetricCost),
					},
				},
			},
			filter: &domainReport.ConfigFilter{
				Limit:       100,
				LimitOrder:  toPointer(string(domainReport.SortAtoZ)).(*string),
				LimitMetric: toPointer(int(domainReport.MetricUsage)).(*int),
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

			validationErrors, err := s.GroupLoadFilter(tt.group, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupToFilter error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

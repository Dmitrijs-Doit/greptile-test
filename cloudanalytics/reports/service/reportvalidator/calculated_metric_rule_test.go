package reportvalidator

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metricDALMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/mocks"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
)

func TestCalculatedMetricRule_validateFiltersWithAttributions(t *testing.T) {
	type args struct {
		calculatedMetricRef *firestore.DocumentRef
		filters             []*domainReport.ConfigFilter
	}

	calculatedMetricID := "calculated_metric_ref_id"

	calculatedMetricRef := &firestore.DocumentRef{
		ID: calculatedMetricID,
	}

	filterAttributionID := fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeAttribution)
	attributionID1 := "attrID1"
	attributionID2 := "attrID2"

	calculatedMetric := metrics.CalculatedMetric{
		Variables: []*metrics.CalculatedMetricVariable{
			{
				Attribution: &firestore.DocumentRef{
					ID: attributionID1,
				},
			},
			{
				Attribution: &firestore.DocumentRef{
					ID: attributionID2,
				},
			},
		},
	}

	type fields struct {
		metricDAL *metricDALMocks.Metrics
	}

	ctx := context.Background()

	tests := []struct {
		name                 string
		args                 args
		fields               fields
		on                   func(*fields)
		wantValidationErrors []errormsg.ErrorMsg
		wantError            error
	}{
		{
			name: "no error when calculated metric is being used and attributions are present in the filters",
			args: args{
				calculatedMetricRef: calculatedMetricRef,
				filters: []*domainReport.ConfigFilter{
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID:     filterAttributionID,
							Values: &[]string{attributionID1, attributionID2},
						},
					},
				},
			},
			on: func(f *fields) {
				f.metricDAL.On("GetCustomMetric",
					ctx,
					calculatedMetricID,
				).Return(&calculatedMetric, nil).
					Once()
			},
		},
		{
			name: "no error when calculated metric is not being used",
			args: args{
				calculatedMetricRef: nil,
			},
		},
		{
			name: "error when calculated metric is being used and one attribution is missing in the filters",
			args: args{
				calculatedMetricRef: calculatedMetricRef,
				filters: []*domainReport.ConfigFilter{
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID:     filterAttributionID,
							Values: &[]string{attributionID1},
						},
					},
				},
			},
			on: func(f *fields) {
				f.metricDAL.On("GetCustomMetric",
					ctx,
					calculatedMetricID,
				).Return(&calculatedMetric, nil).
					Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigFilterField,
					Message: "custom metric must filter attribution for: attrID1,attrID2",
				},
			},
			wantError: externalReportService.ErrValidation,
		},
		{
			name: "error when calculated metric is being used and two attributions are missing in the filters",
			args: args{
				calculatedMetricRef: calculatedMetricRef,
				filters: []*domainReport.ConfigFilter{
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID:     filterAttributionID,
							Values: &[]string{},
						},
					},
				},
			},
			on: func(f *fields) {
				f.metricDAL.On("GetCustomMetric",
					ctx,
					calculatedMetricID,
				).Return(&calculatedMetric, nil).
					Once()
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigFilterField,
					Message: "custom metric must filter attribution for: attrID1,attrID2",
				},
			},
			wantError: externalReportService.ErrValidation,
		},
		{
			name: "error when fail to read calculated metric from dal",
			args: args{
				calculatedMetricRef: calculatedMetricRef,
				filters: []*domainReport.ConfigFilter{
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID:     filterAttributionID,
							Values: &[]string{},
						},
					},
				},
			},
			on: func(f *fields) {
				f.metricDAL.On("GetCustomMetric",
					ctx,
					calculatedMetricID,
				).Return(nil, errors.New("error reading metric")).
					Once()
			},
			wantError: errors.New("error reading metric"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricDAL: &metricDALMocks.Metrics{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			r := NewCalculatedMetricRule(tt.fields.metricDAL)

			validationErrors, err := r.validateFiltersWithAttributions(ctx, tt.args.calculatedMetricRef, tt.args.filters)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

func TestCalculatedMetricRule_validateLimitByValue(t *testing.T) {
	type args struct {
		calculatedMetricRef *firestore.DocumentRef
		metricFilters       []*domainReport.ConfigMetricFilter
	}

	type fields struct {
		metricDAL *metricDALMocks.Metrics
	}

	calculatedMetricID := "calculated_metric_ref_id"

	calculatedMetricRef := &firestore.DocumentRef{
		ID: calculatedMetricID,
	}

	tests := []struct {
		name                 string
		args                 args
		fields               fields
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "success when metric filter uses calculated metric as limit and calculated metric is present",
			args: args{
				calculatedMetricRef: calculatedMetricRef,
				metricFilters: []*domainReport.ConfigMetricFilter{
					{
						Metric: domainReport.MetricCustom,
					},
				},
			},
		},
		{
			name: "error when metric filter uses calculated metric as limit and calculated metric is not present",
			args: args{
				calculatedMetricRef: nil,
				metricFilters: []*domainReport.ConfigMetricFilter{
					{
						Metric: domainReport.MetricCustom,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigMetricFilterField,
					Message: "can only limit by a custom metric if the metric itself is selected as the report metric",
				},
			},
		},
		{
			name: "success when metric filter does not use calculated metric as limit and calculated metric is not present",
			args: args{
				calculatedMetricRef: nil,
				metricFilters: []*domainReport.ConfigMetricFilter{
					{
						Metric: domainReport.MetricCost,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricDAL: &metricDALMocks.Metrics{},
			}

			r := NewCalculatedMetricRule(tt.fields.metricDAL)

			validationErrors := r.validateLimitByValue(tt.args.calculatedMetricRef, tt.args.metricFilters)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestCalculatedMetricRule_validateLimitTopBottom(t *testing.T) {
	type args struct {
		calculatedMetricRef *firestore.DocumentRef
		filters             []*domainReport.ConfigFilter
	}

	type fields struct {
		metricDAL *metricDALMocks.Metrics
	}

	calculatedMetricID := "calculated_metric_ref_id"

	calculatedMetricRef := &firestore.DocumentRef{
		ID: calculatedMetricID,
	}

	customMetric := 4

	tests := []struct {
		name                 string
		args                 args
		fields               fields
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "success when filter uses calculated metric as limit and calculated metric is present",
			args: args{
				calculatedMetricRef: calculatedMetricRef,
				filters: []*domainReport.ConfigFilter{
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID: "filterid",
						},
						LimitMetric: &customMetric,
					},
				},
			},
		},
		{
			name: "error when filter uses calculated metric as limit and calculated metric is not present",
			args: args{
				calculatedMetricRef: nil,
				filters: []*domainReport.ConfigFilter{
					{
						BaseConfigFilter: domainReport.BaseConfigFilter{
							ID: "filterid",
						},
						LimitMetric: &customMetric,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigFilterField,
					Message: "can only limit by a custom metric if the metric itself is selected as the report metric: filterid",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				metricDAL: &metricDALMocks.Metrics{},
			}

			r := NewCalculatedMetricRule(tt.fields.metricDAL)

			validationErrors := r.validateLimitTopBottom(tt.args.calculatedMetricRef, tt.args.filters)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

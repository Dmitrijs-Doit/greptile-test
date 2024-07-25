package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestExternalMetricFilter_ToInternal(t *testing.T) {
	tests := []struct {
		name                 string
		externalMetricFilter ExternalMetricFilter
		want                 *report.MetricFilter
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to internal, invalid",
			externalMetricFilter: ExternalMetricFilter("INVALID"),
			wantValidationErrors: []errormsg.ErrorMsg{{Field: MetricFilterField, Message: "unsupported metric filter operation: INVALID"}},
		},
		{
			name:                 "Conversion to internal, valid",
			externalMetricFilter: ExternalMetricFilterEquals,
			want:                 toPointer(report.MetricFilterEquals).(*report.MetricFilter),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.externalMetricFilter.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestNewExternalMetricFilterFromInternal(t *testing.T) {
	tests := []struct {
		name                 string
		metricFilter         *report.MetricFilter
		want                 *ExternalMetricFilter
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to external, invalid",
			metricFilter:         toPointer(report.MetricFilter("INVALID")).(*report.MetricFilter),
			wantValidationErrors: []errormsg.ErrorMsg{{Field: MetricFilterField, Message: "unsupported metric filter operation: INVALID"}},
		},
		{
			name:         "Conversion to external, ok",
			metricFilter: toPointer(report.MetricFilterEquals).(*report.MetricFilter),
			want:         toPointer(ExternalMetricFilterEquals).(*ExternalMetricFilter),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalMetricFilterFromInternal(tt.metricFilter)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

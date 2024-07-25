package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestExternalComparative_ToInternal(t *testing.T) {
	tests := []struct {
		name                 string
		externalComparative  ExternalComparative
		want                 *string
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to internal, invalid",
			externalComparative:  ExternalComparative("INVALID"),
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalComparativeField, Message: "invalid displayVaues: INVALID"}},
		},
		{
			name:                "Conversion to internal, actuals only",
			externalComparative: ExternalComparativeActualsOnly,
		},
		{
			name:                "Conversion to internal, absolute change",
			externalComparative: ExternalComparativeAbsoluteChange,
			want:                toPointer(report.ComparativeAbsoluteChange).(*string),
		},
		{
			name:                "Conversion to internal, percentage change",
			externalComparative: ExternalComparativePercentageChange,
			want:                toPointer(report.ComparativePercentageChange).(*string),
		},
		{
			name:                "Conversion to internal, absolute and percentage change",
			externalComparative: ExternalComparativeAbsoluteAndPercentage,
			want:                toPointer(report.ComparativeAbsoluteAndPercentage).(*string),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.externalComparative.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestNewExternalComparativeFromInternal(t *testing.T) {
	tests := []struct {
		name                 string
		comparative          *string
		want                 *ExternalComparative
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to external, invalid",
			comparative:          toPointer("INVALID").(*string),
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalComparativeField, Message: "invalid displayVaues: INVALID"}},
		},
		{
			name: "Conversion to external, nil",
			want: toPointer(ExternalComparativeActualsOnly).(*ExternalComparative),
		},
		{
			name:        "Conversion to external, absolute change",
			comparative: toPointer(report.ComparativeAbsoluteChange).(*string),
			want:        toPointer(ExternalComparativeAbsoluteChange).(*ExternalComparative),
		},
		{
			name:        "Conversion to external, percentage change",
			comparative: toPointer(report.ComparativePercentageChange).(*string),
			want:        toPointer(ExternalComparativePercentageChange).(*ExternalComparative),
		},
		{
			name:        "Conversion to external, absolute and percentage change",
			comparative: toPointer(report.ComparativeAbsoluteAndPercentage).(*string),
			want:        toPointer(ExternalComparativeAbsoluteAndPercentage).(*ExternalComparative),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalComparativeFromInternal(tt.comparative)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func toPointer(i interface{}) interface{} {
	switch i := i.(type) {
	case string:
		return &i
	case ExternalComparative:
		return &i
	case []string:
		return &i
	case ExternalMetricFilter:
		return &i
	case report.MetricFilter:
		return &i
	case report.Renderer:
		return &i
	default:
		return i
	}
}

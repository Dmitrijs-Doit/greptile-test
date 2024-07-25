package externalreport

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestExternalCustomTimeRange_ToInternal(t *testing.T) {
	from := time.Date(2023, 10, 1, 2, 3, 4, 0, time.UTC)
	to := time.Date(2023, 11, 1, 2, 3, 4, 0, time.UTC)

	tests := []struct {
		name                    string
		externalCustomTimeRange ExternalCustomTimeRange
		want                    *report.ConfigCustomTimeRange
		wantValidationErrors    []errormsg.ErrorMsg
	}{
		{
			name:                    "Conversion to internal, time is zero",
			externalCustomTimeRange: ExternalCustomTimeRange{},
			wantValidationErrors:    []errormsg.ErrorMsg{{Field: ExternalCustomTimeRangeField, Message: ErrInvalidCustomTimeRangeZero}},
		},
		{
			name: "Conversion to internal, negagive interval",
			externalCustomTimeRange: ExternalCustomTimeRange{
				From: to,
				To:   from,
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalCustomTimeRangeField, Message: ErrInvalidCustomTimeRangeNegativeRange}},
		},
		{
			name: "Conversion to internal, valid",
			externalCustomTimeRange: ExternalCustomTimeRange{
				From: from,
				To:   to,
			},
			want: &report.ConfigCustomTimeRange{
				From: from,
				To:   to,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.externalCustomTimeRange.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestNewExternalCustomTimeRangeFromInternal(t *testing.T) {
	from := time.Date(2023, 10, 1, 2, 3, 4, 0, time.UTC)
	to := time.Date(2023, 11, 1, 2, 3, 4, 0, time.UTC)

	tests := []struct {
		name                 string
		customTimeRange      *report.ConfigCustomTimeRange
		want                 *ExternalCustomTimeRange
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to external, time is zero",
			customTimeRange:      &report.ConfigCustomTimeRange{},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalCustomTimeRangeField, Message: ErrInvalidCustomTimeRangeZero}},
		},
		{
			name: "Conversion to external, negative range",
			customTimeRange: &report.ConfigCustomTimeRange{
				From: to,
				To:   from,
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalCustomTimeRangeField, Message: ErrInvalidCustomTimeRangeNegativeRange}},
		},
		{
			name: "Conversion to external, valid",
			customTimeRange: &report.ConfigCustomTimeRange{
				From: from,
				To:   to,
			},
			want: &ExternalCustomTimeRange{
				From: from,
				To:   to,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalCustomTimeRangeFromInternal(tt.customTimeRange)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

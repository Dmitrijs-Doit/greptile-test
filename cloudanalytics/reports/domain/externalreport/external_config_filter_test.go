package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

var (
	someRegexp = "some regexp .*"
)

func TestExternalConfigFilter_ToInternal(t *testing.T) {
	tests := []struct {
		name                 string
		externalConfigFilter ExternalConfigFilter
		want                 *report.ConfigFilter
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "Conversion to internal, invalid type",
			externalConfigFilter: ExternalConfigFilter{
				ID:     "INVALID",
				Type:   metadata.MetadataFieldType("INVALID"),
				Values: &[]string{"amazon-web-services"},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalConfigFilterField, Message: "invalid config filter type: INVALID"}},
		},
		{
			name: "Conversion to internal, invalid type and missing value",
			externalConfigFilter: ExternalConfigFilter{
				ID:   "INVALID",
				Type: metadata.MetadataFieldType("INVALID"),
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   ExternalConfigFilterField,
					Message: "invalid config filter type: INVALID",
				},
				{
					Field:   ExternalConfigFilterField,
					Message: ErrConfigFilterRequiresValuesOrRegexp,
				},
			},
		},
		{
			name: "Conversion to internal, invalid when both values & regexp are specified",
			externalConfigFilter: ExternalConfigFilter{
				ID:     "cloud_provider",
				Type:   metadata.MetadataFieldTypeFixed,
				Values: &[]string{"amazon-web-services", "google-cloud"},
				Regexp: &someRegexp,
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   ExternalConfigFilterField,
					Message: ErrConfigFilterRequiresValuesOrRegexp,
				},
			},
		},
		{
			name: "Conversion to internal with values, ok",
			externalConfigFilter: ExternalConfigFilter{
				ID:     "cloud_provider",
				Type:   metadata.MetadataFieldTypeFixed,
				Values: &[]string{"amazon-web-services", "google-cloud"},
			},
			want: &report.ConfigFilter{
				BaseConfigFilter: report.BaseConfigFilter{
					ID:     "fixed:cloud_provider",
					Values: toPointer([]string{"amazon-web-services", "google-cloud"}).(*[]string),
					Type:   metadata.MetadataFieldTypeFixed,
				},
			},
		},
		{
			name: "Conversion to internal with regexp, ok",
			externalConfigFilter: ExternalConfigFilter{
				ID:     "cloud_provider",
				Type:   metadata.MetadataFieldTypeFixed,
				Regexp: &someRegexp,
			},
			want: &report.ConfigFilter{
				BaseConfigFilter: report.BaseConfigFilter{
					ID:     "fixed:cloud_provider",
					Regexp: &someRegexp,
					Type:   metadata.MetadataFieldTypeFixed,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.externalConfigFilter.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestNewExternalConfigFilterFromInternal(t *testing.T) {
	tests := []struct {
		name                 string
		configFilter         *report.ConfigFilter
		want                 *ExternalConfigFilter
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to external, no fields",
			configFilter:         &report.ConfigFilter{},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalConfigFilterField, Message: "internal config filter is used for limits"}},
		},
		{
			name: "Conversion to external, invalid ID",
			configFilter: &report.ConfigFilter{
				BaseConfigFilter: report.BaseConfigFilter{
					Values: toPointer([]string{"value"}).(*[]string),
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalConfigFilterField, Message: "invalid id: "}},
		},
		{
			name: "Conversion to external, invalid type",
			configFilter: &report.ConfigFilter{
				BaseConfigFilter: report.BaseConfigFilter{
					ID:     "INVALID:cloud_provider",
					Values: toPointer([]string{"amazon-web-services"}).(*[]string),
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: ExternalConfigFilterField, Message: "invalid config filter type: INVALID"}},
		},
		{
			name: "Conversion to external with values, ok",
			configFilter: &report.ConfigFilter{
				BaseConfigFilter: report.BaseConfigFilter{
					ID:     "fixed:cloud_provider",
					Values: toPointer([]string{"amazon-web-services"}).(*[]string),
				},
			},
			want: &ExternalConfigFilter{
				ID:     "cloud_provider",
				Type:   metadata.MetadataFieldTypeFixed,
				Values: &[]string{"amazon-web-services"},
			},
		},
		{
			name: "Conversion to external with regexp, ok",
			configFilter: &report.ConfigFilter{
				BaseConfigFilter: report.BaseConfigFilter{
					ID:     "fixed:cloud_provider",
					Regexp: &someRegexp,
				},
			},
			want: &ExternalConfigFilter{
				ID:     "cloud_provider",
				Type:   metadata.MetadataFieldTypeFixed,
				Regexp: &someRegexp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalConfigFilterFromInternal(tt.configFilter)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

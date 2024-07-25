package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

func TestDimension_NewExternalDimensionFromInternal(t *testing.T) {
	tests := []struct {
		name                 string
		col                  string
		want                 *Dimension
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Conversion to external, invalid",
			col:                  "INVALID",
			wantValidationErrors: []errormsg.ErrorMsg{{Field: DimensionsField, Message: "invalid id: INVALID"}},
		},
		{
			name:                 "Conversion to external, invalid type",
			col:                  "INVALID:year",
			wantValidationErrors: []errormsg.ErrorMsg{{Field: DimensionsField, Message: "invalid metadata field type: INVALID"}},
		},
		{
			name: "Conversion to external, datetime",
			col:  "datetime:year",
			want: &Dimension{
				ID:   "year",
				Type: metadata.MetadataFieldTypeDatetime,
			},
		},
		{
			name: "Conversion to external, project_label, organization tag",
			col:  "project_label:YXdzLW9yZy90ZWFt",
			want: &Dimension{
				ID:   "YXdzLW9yZy90ZWFt",
				Type: metadata.MetadataFieldTypeOrganizationTagExternal,
			},
		},
		{
			name: "Conversion to external, project_label, regular value",
			col:  "project_label:Y29udGFjdA==",
			want: &Dimension{
				ID:   "Y29udGFjdA==",
				Type: metadata.MetadataFieldTypeProjectLabel,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalDimensionFromInternal(tt.col)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestDimension_ToInternal(t *testing.T) {
	tests := []struct {
		name                 string
		dimension            Dimension
		want                 string
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "Conversion to internal, invalid",
			dimension: Dimension{
				Type: metadata.MetadataFieldType("INVALID"),
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: DimensionsField, Message: "invalid metadata field type: INVALID"}},
		},
		{
			name: "Conversion to internal, ok",
			dimension: Dimension{
				Type: metadata.MetadataFieldTypeDatetime,
				ID:   "year",
			},
			want: "datetime:year",
		},
		{
			name: "Conversion to internal from project_label, organization tag",
			dimension: Dimension{
				Type: metadata.MetadataFieldTypeOrganizationTagExternal,
				ID:   "YXdzLW9yZy90ZWFt",
			},
			want: "project_label:YXdzLW9yZy90ZWFt",
		},
		{
			name: "Conversion to internal from project_label, regular value",
			dimension: Dimension{
				Type: metadata.MetadataFieldTypeProjectLabel,
				ID:   "Y29udGFjdA==",
			},
			want: "project_label:Y29udGFjdA==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.dimension.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

func TestExternalConfig_LoadCols(t *testing.T) {
	tests := []struct {
		name                 string
		cols                 []string
		want                 *ExternalConfig
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "Invalid cols",
			cols: []string{"INVALID1", "INVALID2"},
			want: &ExternalConfig{},
			wantValidationErrors: []errormsg.ErrorMsg{
				{Field: DimensionsField, Message: "invalid id: INVALID1"},
				{Field: DimensionsField, Message: "invalid id: INVALID2"},
			},
		},
		{
			name: "Happy path",
			cols: []string{"fixed:year", "fixed:month"},
			want: &ExternalConfig{
				Dimensions: []*Dimension{
					{
						ID:   "year",
						Type: metadata.MetadataFieldTypeFixed,
					},
					{
						ID:   "month",
						Type: metadata.MetadataFieldTypeFixed,
					},
				},
			},
		},
		{
			name: "load organizational tag and the regular project label",
			cols: []string{"project_label:YXdzLW9yZy90ZWFt", "project_label:Y29udGFjdA=="},
			want: &ExternalConfig{
				Dimensions: []*Dimension{
					{
						ID:   "YXdzLW9yZy90ZWFt",
						Type: metadata.MetadataFieldTypeOrganizationTagExternal,
					},
					{
						ID:   "Y29udGFjdA==",
						Type: metadata.MetadataFieldTypeProjectLabel,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			externalConfig := &ExternalConfig{}
			validationErrors := externalConfig.LoadCols(tt.cols)

			assert.Equal(t, tt.want, externalConfig)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestExternalConfig_LoadRows(t *testing.T) {
	tests := []struct {
		name                 string
		rows                 []string
		want                 *ExternalConfig
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "Invalid rows",
			rows: []string{"INVALID1", "INVALID2"},
			want: &ExternalConfig{},
			wantValidationErrors: []errormsg.ErrorMsg{
				{Field: GroupField, Message: "invalid id: INVALID1"},
				{Field: GroupField, Message: "invalid id: INVALID2"},
			},
		},
		{
			name: "Happy path",
			rows: []string{"datetime:year", "datetime:month"},
			want: &ExternalConfig{
				Groups: []*Group{
					{
						ID:   "year",
						Type: metadata.MetadataFieldTypeDatetime,
					},
					{
						ID:   "month",
						Type: metadata.MetadataFieldTypeDatetime,
					},
				},
			},
		},
		{
			name: "load organizational tag and the regular project label",
			rows: []string{"project_label:YXdzLW9yZy90ZWFt", "project_label:Y29udGFjdA=="},
			want: &ExternalConfig{
				Groups: []*Group{
					{
						ID:   "aws-org/team",
						Type: metadata.MetadataFieldTypeOrganizationTagExternal,
					},
					{
						ID:   "contact",
						Type: metadata.MetadataFieldTypeProjectLabel,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			externalConfig := &ExternalConfig{}
			validationErrors := externalConfig.LoadRows(tt.rows)

			assert.Equal(t, tt.want, externalConfig)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

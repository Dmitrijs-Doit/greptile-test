package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
)

func TestGroup_LoadRow(t *testing.T) {
	tests := []struct {
		name                 string
		group                *Group
		wantGroup            *Group
		row                  string
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:                 "Not enough fields",
			group:                &Group{},
			row:                  "INVALID",
			wantValidationErrors: []errormsg.ErrorMsg{{Field: GroupField, Message: "invalid id: INVALID"}},
		},
		{
			name:                 "Invalid type",
			group:                &Group{},
			row:                  "INVALID:INVALID",
			wantValidationErrors: []errormsg.ErrorMsg{{Field: GroupField, Message: "invalid metadata field type: INVALID"}},
		},
		{
			name:  "LoadRows ok",
			group: &Group{},
			wantGroup: &Group{
				ID:   "team",
				Type: metadata.MetadataFieldTypeLabel,
			},
			row: "label:dGVhbQ==",
		},
		{
			name:  "LoadRow ok",
			group: &Group{},
			row:   "datetime:year",
		},
		{
			name:  "LoadRow from project_label, organization tag",
			group: &Group{},
			wantGroup: &Group{
				ID:   "aws-org/team",
				Type: metadata.MetadataFieldTypeOrganizationTagExternal,
			},
			row: "project_label:YXdzLW9yZy90ZWFt",
		},
		{
			name:  "LoadRow from project_label, regular value",
			group: &Group{},
			wantGroup: &Group{
				ID:   "contact",
				Type: metadata.MetadataFieldTypeProjectLabel,
			},
			row: "project_label:Y29udGFjdA==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validationErrors := tt.group.LoadRow(tt.row)
			assert.Equal(t, tt.wantValidationErrors, validationErrors)

			if tt.wantGroup != nil {
				assert.Equal(t, tt.wantGroup, tt.group)
			}
		})
	}
}

func TestGroup_ToInternal(t *testing.T) {
	tests := []struct {
		name                 string
		group                Group
		want                 string
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "Invalid type",
			group: Group{
				Type: metadata.MetadataFieldType("INVALID"),
			},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: GroupField, Message: "invalid metadata field type: INVALID"}},
		},
		{
			name: "Ok",
			group: Group{
				ID:   "year",
				Type: metadata.MetadataFieldTypeDatetime,
			},
			want: "datetime:year",
		},
		{
			name: "from organization_tag",
			group: Group{
				ID:   "aws-org/team",
				Type: metadata.MetadataFieldTypeOrganizationTagExternal,
			},
			want: "project_label:YXdzLW9yZy90ZWFt",
		},
		{
			name: "from project_label",
			group: Group{
				ID:   "contact",
				Type: metadata.MetadataFieldTypeProjectLabel,
			},
			want: "project_label:Y29udGFjdA==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.group.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

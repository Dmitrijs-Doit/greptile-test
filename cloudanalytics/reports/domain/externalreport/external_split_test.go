package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
)

func TestExternalSplit_ToInternal(t *testing.T) {
	targetVal := 0.7
	targetVal2 := 0.3

	targetValTooSmall := 0.1

	tests := []struct {
		name                 string
		externalSplit        ExternalSplit
		want                 *split.Split
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "conversion to internal, valid",
			externalSplit: ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: split.ModeCustom,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets: []ExternalSplitTarget{
					{
						ID:    "attr2",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetVal,
					},
					{
						ID:    "attr3",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetVal2,
					},
				},
			},
			want: &split.Split{
				ID:     "attribution_group:attrgroup111",
				Key:    "",
				Type:   "attribution_group",
				Origin: "attribution:attr1",
				Mode:   "custom",
				Targets: []split.SplitTarget{
					{
						ID:    "attribution:attr2",
						Value: targetVal,
					},
					{
						ID:    "attribution:attr3",
						Value: targetVal2,
					},
				},
				IncludeOrigin: true,
			},
		},
		{
			name: "error when target total does not total to 1 when mode is custom",
			externalSplit: ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: split.ModeCustom,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets: []ExternalSplitTarget{
					{
						ID:    "attr2",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetVal,
					},
					{
						ID:    "attr3",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetValTooSmall,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "split",
					Message: "invalid target total for custom mode: 0.80",
				},
			},
		},
		{
			name: "invalid split type",
			externalSplit: ExternalSplit{
				ID:   "attrroup111",
				Type: "some-random-value",
				Mode: split.ModeCustom,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets:       []ExternalSplitTarget{},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "split",
					Message: "invalid split type: some-random-value",
				},
			},
		},
		{
			name: "invalid split mode",
			externalSplit: ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: "some-random-mode",
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets:       []ExternalSplitTarget{},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "split",
					Message: "invalid split mode: some-random-mode",
				},
			},
		},
		{
			name: "fail when mode is custom and target does not have a value",
			externalSplit: ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: split.ModeCustom,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets: []ExternalSplitTarget{
					{
						ID:   "attr2",
						Type: metadata.MetadataFieldTypeAttribution,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "target",
					Message: "invalid target value of target id: attr2. Not compatible with mode: custom",
				},
			},
		},
		{
			name: "fail when mode is proportional and target does have a value",
			externalSplit: ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: split.ModeProportional,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets: []ExternalSplitTarget{
					{
						ID:    "attr2",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetVal,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "target",
					Message: "invalid target value of target id: attr2. Not compatible with mode: proportional",
				},
			},
		},
		{
			name: "fail when mode is even and target does have a value",
			externalSplit: ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: split.ModeEven,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets: []ExternalSplitTarget{
					{
						ID:    "attr2",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetVal,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "target",
					Message: "invalid target value of target id: attr2. Not compatible with mode: even",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := tt.externalSplit.ToInternal()

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

func TestNewExternalSplitFromInternal(t *testing.T) {
	targetVal := 14.2

	tests := []struct {
		name                 string
		split                *split.Split
		want                 *ExternalSplit
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name: "conversion to external, valid",
			split: &split.Split{
				ID:     "attribution_group:attrgroup111",
				Key:    "some key",
				Type:   "attribution_group",
				Origin: "attribution:attr1",
				Mode:   "custom",
				Targets: []split.SplitTarget{
					{
						ID:    "attribution:attr2",
						Value: 14.2,
					},
				},
				IncludeOrigin: true,
			},
			want: &ExternalSplit{
				ID:   "attrgroup111",
				Type: metadata.MetadataFieldTypeAttributionGroup,
				Mode: split.ModeCustom,
				Origin: ExternalOrigin{
					ID:   "attr1",
					Type: metadata.MetadataFieldTypeAttribution,
				},
				IncludeOrigin: true,
				Targets: []ExternalSplitTarget{
					{
						ID:    "attr2",
						Type:  metadata.MetadataFieldTypeAttribution,
						Value: &targetVal,
					},
				},
			},
		},
		{
			name: "fail conversion to external",
			split: &split.Split{
				ID:     "some-strange-id:attrroup111",
				Key:    "some key",
				Type:   "attribution_group",
				Origin: "attribution:attr1",
				Mode:   "custom",
				Targets: []split.SplitTarget{
					{
						ID:    "attribution:attr2",
						Value: 14.2,
					},
				},
				IncludeOrigin: true,
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "split",
					Message: "invalid split id: some-strange-id:attrroup111",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalSplitFromInternal(tt.split)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

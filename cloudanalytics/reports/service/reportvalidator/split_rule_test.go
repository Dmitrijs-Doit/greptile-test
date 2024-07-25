package reportvalidator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestSplitRule(t *testing.T) {
	targetVal := 0.3
	targetVal2 := 0.7

	tests := []struct {
		name                 string
		report               *domainReport.Report
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name: "split exists in the rows",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Rows: []string{"attribution_group:attrgroup111"},
					Splits: []split.Split{
						{
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
				},
			},
			wantValidationErrors: nil,
			wantErr:              false,
		},
		{
			name: "fail when split does not exist in the rows",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Rows: []string{"some-other-row"},
					Splits: []split.Split{
						{
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
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   "split",
					Message: "split element must be present in group: attribution_group:attrgroup111",
				},
			},
			wantErr: true,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSplitRule()

			validationErrors, err := r.Validate(ctx, tt.report)
			if (err != nil) != tt.wantErr {
				t.Errorf("split.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

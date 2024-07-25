package reportvalidator

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestTreemapsRule(t *testing.T) {
	comparativeStr := "values"
	allowedDimension := fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeDatetime, domainReport.TimeIntervalYear)

	tests := []struct {
		name                 string
		report               *domainReport.Report
		wantValidationErrors []errormsg.ErrorMsg
		wantErr              bool
	}{
		{
			name:   "Config is nil",
			report: &domainReport.Report{},
		},
		{
			name: "Renderer is not treemap",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Renderer: domainReport.RendererAreaChart,
				},
			},
		},
		{
			name: "Aggregator is not total",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Aggregator: domainReport.AggregatorPercentCol,
					Renderer:   domainReport.RendererTreemapChart,
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{Field: domainReport.ConfigRendererField,
					Message: ErrInvalidTreemapsAggregator},
			},
			wantErr: true,
		},
		{
			name: "Forecast is set",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Renderer:   domainReport.RendererTreemapChart,
					Aggregator: domainReport.AggregatorTotal,
					Features: []domainReport.Feature{
						domainReport.FeatureForecast,
					},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigRendererField,
					Message: ErrInvalidTreemapsFeatures,
				},
			},
			wantErr: true,
		},
		{
			name: "A dimension is set",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Renderer:   domainReport.RendererTreemapChart,
					Aggregator: domainReport.AggregatorTotal,
					Cols:       []string{"fixed:kubernetes_cluster_name"},
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigRendererField,
					Message: "treemaps renderer cannot be used with dimension fixed:kubernetes_cluster_name",
				},
			},
			wantErr: true,
		},
		{
			name: "Comparative is set",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Renderer:    domainReport.RendererTreemapChart,
					Aggregator:  domainReport.AggregatorTotal,
					Comparative: &comparativeStr,
				},
			},
			wantValidationErrors: []errormsg.ErrorMsg{
				{
					Field:   domainReport.ConfigRendererField,
					Message: ErrInvalidTreemapsDisplayValues,
				},
			},
			wantErr: true,
		},
		{
			name: "Happy path",
			report: &domainReport.Report{
				Config: &domainReport.Config{
					Renderer:   domainReport.RendererTreemapChart,
					Aggregator: domainReport.AggregatorTotal,
					Cols:       []string{allowedDimension},
				},
			},
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewTreemapsRule()

			validationErrors, err := r.Validate(ctx, tt.report)
			if (err != nil) != tt.wantErr {
				t.Errorf("treemaps.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

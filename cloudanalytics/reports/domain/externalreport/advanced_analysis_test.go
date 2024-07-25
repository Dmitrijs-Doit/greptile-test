package externalreport

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

func TestAdvancedAnalysis_ToInternal(t *testing.T) {
	tests := []struct {
		name             string
		advancedAnalysis AdvancedAnalysis
		want             []report.Feature
	}{
		{
			name: "Conversion to internal, trending up",
			advancedAnalysis: AdvancedAnalysis{
				TrendingUp: true,
			},
			want: []report.Feature{report.FeatureTrendingUp},
		},
		{
			name: "Conversion to internal, trending down",
			advancedAnalysis: AdvancedAnalysis{
				TrendingDown: true,
			},
			want: []report.Feature{report.FeatureTrendingDown},
		},
		{
			name: "Conversion to internal, not trending",
			advancedAnalysis: AdvancedAnalysis{
				NotTrending: true,
			},
			want: []report.Feature{report.FeatureTrendingNone},
		},
		{
			name: "Conversion to internal, forecast",
			advancedAnalysis: AdvancedAnalysis{
				Forecast: true,
			},
			want: []report.Feature{report.FeatureForecast},
		},
		{
			name: "Conversion to internal, multiple options",
			advancedAnalysis: AdvancedAnalysis{
				TrendingUp:   true,
				TrendingDown: true,
				Forecast:     true,
			},
			want: []report.Feature{report.FeatureTrendingUp, report.FeatureTrendingDown, report.FeatureForecast},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.advancedAnalysis.ToInternal()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAdvancedAnalysis_NewExternalAdvancedAnalysisFromInternal(t *testing.T) {
	tests := []struct {
		name                 string
		features             []report.Feature
		want                 *AdvancedAnalysis
		wantValidationErrors []errormsg.ErrorMsg
	}{
		{
			name:     "Conversion to external, trending up",
			features: []report.Feature{report.FeatureTrendingUp},
			want: &AdvancedAnalysis{
				TrendingUp: true,
			},
		},
		{
			name:     "Conversion to external, trending down",
			features: []report.Feature{report.FeatureTrendingDown},
			want: &AdvancedAnalysis{
				TrendingDown: true,
			},
		},
		{
			name:     "Conversion to external, not trending",
			features: []report.Feature{report.FeatureTrendingNone},
			want: &AdvancedAnalysis{
				NotTrending: true,
			},
		},
		{
			name:     "Conversion to external, forecast",
			features: []report.Feature{report.FeatureForecast},
			want: &AdvancedAnalysis{
				Forecast: true,
			},
		},
		{
			name:                 "Conversion to external, invalid",
			features:             []report.Feature{report.Feature("INVALID")},
			wantValidationErrors: []errormsg.ErrorMsg{{Field: AdvancedAnalysisField, Message: "unsupported advanced analysis: INVALID"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, validationErrors := NewExternalAdvancedAnalysisFromInternal(tt.features)

			assert.Equal(t, tt.want, got)

			assert.Equal(t, tt.wantValidationErrors, validationErrors)
		})
	}
}

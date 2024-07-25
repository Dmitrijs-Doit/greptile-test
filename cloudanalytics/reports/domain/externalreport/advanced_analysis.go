package externalreport

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// Advanced analysis toggles. Each of these can be set independently
type AdvancedAnalysis struct {
	TrendingUp   bool `json:"trendingUp"`
	TrendingDown bool `json:"trendingDown"`
	NotTrending  bool `json:"notTrending"`
	Forecast     bool `json:"forecast"`
}

func (advancedAnalysis AdvancedAnalysis) ToInternal() []report.Feature {
	var features []report.Feature

	if advancedAnalysis.TrendingUp {
		features = append(features, report.FeatureTrendingUp)
	}

	if advancedAnalysis.TrendingDown {
		features = append(features, report.FeatureTrendingDown)
	}

	if advancedAnalysis.NotTrending {
		features = append(features, report.FeatureTrendingNone)
	}

	if advancedAnalysis.Forecast {
		features = append(features, report.FeatureForecast)
	}

	return features
}

func NewExternalAdvancedAnalysisFromInternal(features []report.Feature) (*AdvancedAnalysis, []errormsg.ErrorMsg) {
	var advancedAnalysis AdvancedAnalysis

	for _, feature := range features {
		switch feature {
		case report.FeatureTrendingUp:
			advancedAnalysis.TrendingUp = true
		case report.FeatureTrendingDown:
			advancedAnalysis.TrendingDown = true
		case report.FeatureTrendingNone:
			advancedAnalysis.NotTrending = true
		case report.FeatureForecast:
			advancedAnalysis.Forecast = true
		default:
			return nil, []errormsg.ErrorMsg{
				{
					Field:   AdvancedAnalysisField,
					Message: fmt.Sprintf("%s: %s", report.ErrUnsupportedAdvancedAnalysisMsg, feature),
				},
			}
		}
	}

	return &advancedAnalysis, nil
}

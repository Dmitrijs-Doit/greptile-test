package reportvalidator

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
)

var (
	treemapsRuleDisallowedFeatures = []domainReport.Feature{
		domainReport.FeatureForecast,
		domainReport.FeatureTrendingDown,
		domainReport.FeatureTrendingNone,
		domainReport.FeatureTrendingUp,
	}

	treemapsRuleAllowedDimensions = []string{
		fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeDatetime, domainReport.TimeIntervalYear),
		fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeDatetime, domainReport.TimeIntervalMonth),
		fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeDatetime, domainReport.TimeIntervalDay),
	}
)

type TreemapsRule struct{}

func NewTreemapsRule() *TreemapsRule {
	return &TreemapsRule{}
}

// Validate ensures that the following conditions are met if Treemap is selected:
// - Must use aggregation “Total”
// - Must not use trends or forecast features
// - Must not have any “dimensions”
// - “displayValues” must be “actuals only”
func (r *TreemapsRule) Validate(_ context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if report.Config == nil {
		return nil, nil
	}

	if report.Config.Renderer != domainReport.RendererTreemapChart {
		return nil, nil
	}

	if report.Config.Aggregator != domainReport.AggregatorTotal {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigRendererField,
			Message: ErrInvalidTreemapsAggregator,
		})
	}

	for _, feature := range report.Config.Features {
		if slices.Contains(treemapsRuleDisallowedFeatures, feature) {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigRendererField,
				Message: ErrInvalidTreemapsFeatures,
			})

			break
		}
	}

	for _, col := range report.Config.Cols {
		if !slices.Contains(treemapsRuleAllowedDimensions, col) {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigRendererField,
				Message: fmt.Sprintf("%s %s", ErrInvalidTreemapsDimension, col),
			})
		}
	}

	if report.Config.Comparative != nil {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigRendererField,
			Message: ErrInvalidTreemapsDisplayValues,
		})
	}

	if validationErrors != nil {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

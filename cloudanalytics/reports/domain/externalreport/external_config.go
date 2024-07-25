package externalreport

import (
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

// Report configuration
type ExternalConfig struct {
	Metric           *metrics.ExternalMetric     `json:"metric,omitempty"`
	MetricFilter     *ExternalConfigMetricFilter `json:"metricFilter,omitempty"`
	Aggregator       *report.Aggregator          `json:"aggregation,omitempty"`
	AdvancedAnalysis *AdvancedAnalysis           `json:"advancedAnalysis,omitempty"`
	TimeInterval     *report.TimeInterval        `json:"timeInterval,omitempty"`
	// The dimensions to apply to the report. If not set, the following dimensions will be used by default: "datetime:year", "datetime:month", "datetime:day"
	Dimensions   []*Dimension         `json:"dimensions,omitempty"`
	TimeSettings *report.TimeSettings `json:"timeRange,omitempty"`
	// Whether to include credits or not.
	// If set, the report must use time interval “month”/”quarter”/”year”
	IncludeCredits *bool `json:"includePromotionalCredits,omitempty"`
	// Whether to include subtotals or not. This configuration has no impact when reading report's data via API
	IncludeSubtotals *bool `json:"includeSubtotals,omitempty"`
	// The filters to use in this report
	Filters []*ExternalConfigFilter `json:"filters,omitempty"`
	// The groups to use in the report.
	Groups          []*Group                 `json:"group,omitempty"`
	Renderer        *ExternalRenderer        `json:"layout,omitempty"`
	Comparative     *ExternalComparative     `json:"displayValues,omitempty"`
	Currency        *fixer.Currency          `json:"currency,omitempty"`
	CustomTimeRange *ExternalCustomTimeRange `json:"customTimeRange,omitempty"`
	// The splits to use in the report.
	Splits []*ExternalSplit `json:"splits,omitempty"`
	// SortGroups, possible values: "asc"/"a_to_z"/"desc". If not set, the following value will be used by default: "asc"
	// This configuration has no impact when reading data from the report via API."
	SortGroups *report.Sort `json:"sortGroups,omitempty"`
	// SortDimensions, possible values: "asc"/"a_to_z"/"desc". If not set, the following value will be used by default: "desc"
	// This configuration has no impact when reading data from the report via API."
	SortDimensions *report.Sort `json:"sortDimensions,omitempty"`
	// Datasource to use in the report.
	DataSource *ExternalDataSource `json:"dataSource,omitempty"`
}

func (externalConfig *ExternalConfig) LoadCols(cols []string) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	for _, col := range cols {
		dimension, dimensionValidationErrors := NewExternalDimensionFromInternal(col)
		if dimensionValidationErrors != nil {
			validationErrors = append(validationErrors, dimensionValidationErrors...)
			continue
		}

		externalConfig.Dimensions = append(externalConfig.Dimensions, dimension)
	}

	return validationErrors
}

func (externalConfig *ExternalConfig) LoadRows(rows []string) []errormsg.ErrorMsg {
	var validationErrors []errormsg.ErrorMsg

	for _, row := range rows {
		var group Group

		if validationError := group.LoadRow(row); validationError != nil {
			validationErrors = append(validationErrors, validationError...)
			continue
		}

		externalConfig.Groups = append(externalConfig.Groups, &group)
	}

	if len(validationErrors) > 0 {
		return validationErrors
	}

	return nil
}

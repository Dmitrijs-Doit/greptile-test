package externalreport

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// Enables comparative mode.
// swagger:enum ExternalComparative
// default : "actuals_only"
// To use comparative mode the following criteria must be met:
//
// - Must use aggregation “total”
// - Must have a valid time series dimension configuration and dimensions must have default sorting (alphabetically)
// - Must not use “forecast”
// - “absolute_and_percentage” mode is only compatible with Table renderers (table or heatmap)
type ExternalComparative string

const (
	ExternalComparativeActualsOnly           ExternalComparative = "actuals_only"
	ExternalComparativeAbsoluteChange        ExternalComparative = "absolute_change"
	ExternalComparativePercentageChange      ExternalComparative = "percentage_change"
	ExternalComparativeAbsoluteAndPercentage ExternalComparative = "absolute_and_percentage"
)

func (externalComparative ExternalComparative) ToInternal() (*string, []errormsg.ErrorMsg) {
	switch externalComparative {
	case ExternalComparativeActualsOnly:
		return nil, nil
	case ExternalComparativeAbsoluteChange:
		res := report.ComparativeAbsoluteChange
		return &res, nil
	case ExternalComparativePercentageChange:
		res := report.ComparativePercentageChange
		return &res, nil
	case ExternalComparativeAbsoluteAndPercentage:
		res := report.ComparativeAbsoluteAndPercentage
		return &res, nil
	}

	return nil, []errormsg.ErrorMsg{
		{
			Field:   ExternalComparativeField,
			Message: fmt.Sprintf("%s: %s", report.ErrInvalidComparativeMsg, externalComparative),
		},
	}
}

func NewExternalComparativeFromInternal(comparative *string) (*ExternalComparative, []errormsg.ErrorMsg) {
	var externalComparative ExternalComparative

	if comparative == nil {
		externalComparative = ExternalComparativeActualsOnly
		return &externalComparative, nil
	}

	switch *comparative {
	case report.ComparativeAbsoluteChange:
		externalComparative = ExternalComparativeAbsoluteChange
	case report.ComparativePercentageChange:
		externalComparative = ExternalComparativePercentageChange
	case report.ComparativeAbsoluteAndPercentage:
		externalComparative = ExternalComparativeAbsoluteAndPercentage
	default:
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalComparativeField,
				Message: fmt.Sprintf("%s: %s", report.ErrInvalidComparativeMsg, *comparative),
			},
		}
	}

	return &externalComparative, nil
}

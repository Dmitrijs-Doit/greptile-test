package externalreport

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// swagger:enum ExternalMetricFilter
type ExternalMetricFilter string

const (
	ExternalMetricFilterGreaterThan   ExternalMetricFilter = "gt"
	ExternalMetricFilterLessThan      ExternalMetricFilter = "lt"
	ExternalMetricFilterLessEqThan    ExternalMetricFilter = "lte"
	ExternalMetricFilterGreaterEqThan ExternalMetricFilter = "gte"
	ExternalMetricFilterBetween       ExternalMetricFilter = "b"
	ExternalMetricFilterNotBetween    ExternalMetricFilter = "nb"
	ExternalMetricFilterEquals        ExternalMetricFilter = "e"
	ExternalMetricFilterNotEquals     ExternalMetricFilter = "ne"
)

func (externalMetricFilter ExternalMetricFilter) Validate() bool {
	switch externalMetricFilter {
	case ExternalMetricFilterGreaterThan,
		ExternalMetricFilterLessThan,
		ExternalMetricFilterLessEqThan,
		ExternalMetricFilterGreaterEqThan,
		ExternalMetricFilterBetween,
		ExternalMetricFilterNotBetween,
		ExternalMetricFilterEquals,
		ExternalMetricFilterNotEquals:
		return true
	default:
		return false
	}
}

var externalToInternalMetricFilterMap = map[ExternalMetricFilter]report.MetricFilter{
	ExternalMetricFilterGreaterThan:   report.MetricFilterGreaterThan,
	ExternalMetricFilterLessThan:      report.MetricFilterLessThan,
	ExternalMetricFilterLessEqThan:    report.MetricFilterLessEqThan,
	ExternalMetricFilterGreaterEqThan: report.MetricFilterGreaterEqThan,
	ExternalMetricFilterBetween:       report.MetricFilterBetween,
	ExternalMetricFilterNotBetween:    report.MetricFilterNotBetween,
	ExternalMetricFilterEquals:        report.MetricFilterEquals,
	ExternalMetricFilterNotEquals:     report.MetricFilterNotEquals,
}

func (externalMetricFilter ExternalMetricFilter) ToInternal() (*report.MetricFilter, []errormsg.ErrorMsg) {
	if metricFilter, ok := externalToInternalMetricFilterMap[externalMetricFilter]; ok {
		return &metricFilter, nil
	}

	return nil, []errormsg.ErrorMsg{
		{
			Field:   MetricFilterField,
			Message: fmt.Sprintf("%s: %s", report.ErrUnsupportedMetricFilterMsg, externalMetricFilter),
		},
	}
}

var internalToExternalMetricFilterMap = map[report.MetricFilter]ExternalMetricFilter{
	report.MetricFilterGreaterThan:   ExternalMetricFilterGreaterThan,
	report.MetricFilterLessThan:      ExternalMetricFilterLessThan,
	report.MetricFilterLessEqThan:    ExternalMetricFilterLessEqThan,
	report.MetricFilterGreaterEqThan: ExternalMetricFilterGreaterEqThan,
	report.MetricFilterBetween:       ExternalMetricFilterBetween,
	report.MetricFilterNotBetween:    ExternalMetricFilterNotBetween,
	report.MetricFilterEquals:        ExternalMetricFilterEquals,
	report.MetricFilterNotEquals:     ExternalMetricFilterNotEquals,
}

func NewExternalMetricFilterFromInternal(metricFilter *report.MetricFilter) (*ExternalMetricFilter, []errormsg.ErrorMsg) {
	if externalMetricFilter, ok := internalToExternalMetricFilterMap[*metricFilter]; ok {
		return &externalMetricFilter, nil
	}

	return nil, []errormsg.ErrorMsg{
		{
			Field:   MetricFilterField,
			Message: fmt.Sprintf("%s: %s", report.ErrUnsupportedMetricFilterMsg, *metricFilter),
		},
	}
}

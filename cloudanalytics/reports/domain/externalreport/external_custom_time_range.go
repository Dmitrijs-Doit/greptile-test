package externalreport

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

// Set the specific start and end times for the report in rfc3339 format.
// If present, the timeSettings mode must be set to "custom".
type ExternalCustomTimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

func (externalCustomTimeRange ExternalCustomTimeRange) ToInternal() (*report.ConfigCustomTimeRange, []errormsg.ErrorMsg) {
	if externalCustomTimeRange.From.IsZero() || externalCustomTimeRange.To.IsZero() {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalCustomTimeRangeField,
				Message: ErrInvalidCustomTimeRangeZero,
			},
		}
	}

	if externalCustomTimeRange.To.Before(externalCustomTimeRange.From) {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalCustomTimeRangeField,
				Message: ErrInvalidCustomTimeRangeNegativeRange,
			},
		}
	}

	return &report.ConfigCustomTimeRange{
		From: externalCustomTimeRange.From,
		To:   externalCustomTimeRange.To,
	}, nil
}

func NewExternalCustomTimeRangeFromInternal(customTimeRange *report.ConfigCustomTimeRange) (*ExternalCustomTimeRange, []errormsg.ErrorMsg) {
	if customTimeRange.From.IsZero() || customTimeRange.To.IsZero() {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalCustomTimeRangeField,
				Message: ErrInvalidCustomTimeRangeZero,
			},
		}
	}

	if customTimeRange.To.Before(customTimeRange.From) {
		return nil, []errormsg.ErrorMsg{
			{
				Field:   ExternalCustomTimeRangeField,
				Message: ErrInvalidCustomTimeRangeNegativeRange,
			},
		}
	}

	return &ExternalCustomTimeRange{
		From: customTimeRange.From,
		To:   customTimeRange.To,
	}, nil
}

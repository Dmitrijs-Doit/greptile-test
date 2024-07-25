package reportvalidator

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type LimitTopBottomRule struct{}

func NewLimitTopBottomRule() *LimitTopBottomRule {
	return &LimitTopBottomRule{}
}

// Validate ensures that the limit only uses
// the fields that are specified in “group”.
func (r *LimitTopBottomRule) Validate(_ context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	rows := report.Config.Rows

	for _, filter := range report.Config.Filters {
		if filter.LimitMetric == nil {
			continue
		}

		id := filter.BaseConfigFilter.ID

		if !slice.Contains(rows, id) {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigFilterField,
				Message: fmt.Sprintf("%s: %s", ErrInvalidLimitTopBottom, id),
			})
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

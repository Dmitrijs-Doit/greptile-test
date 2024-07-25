package reportvalidator

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
)

type SplitRule struct{}

func NewSplitRule() *SplitRule {
	return &SplitRule{}
}

func (r *SplitRule) Validate(_ context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	type void struct{}

	var member void

	rows := make(map[string]void)
	for _, row := range report.Config.Rows {
		rows[row] = member
	}

	for _, split := range report.Config.Splits {
		if _, ok := rows[split.ID]; !ok {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigSplitField,
				Message: fmt.Sprintf(ErrMsgFormat, ErrInvalidSplit, split.ID),
			})
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

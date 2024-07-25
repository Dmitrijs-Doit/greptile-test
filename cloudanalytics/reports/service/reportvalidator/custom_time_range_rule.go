package reportvalidator

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
)

type CustomTimeRangeRule struct{}

func NewCustomTimeRangeRule() *CustomTimeRangeRule {
	return &CustomTimeRangeRule{}
}

func (r *CustomTimeRangeRule) Validate(_ context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	if report.Config.CustomTimeRange == nil {
		return nil, nil
	}

	if report.Config.TimeSettings == nil {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigCustomTimeRangeField,
			Message: ErrInvalidCustomTimeRangeModeNotSet,
		})
	} else {
		if report.Config.TimeSettings.Mode != domainReport.TimeSettingsModeCustom {
			validationErrors = append(validationErrors, errormsg.ErrorMsg{
				Field:   domainReport.ConfigCustomTimeRangeField,
				Message: ErrInvalidCustomTimeRangeModeNotSet,
			})
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

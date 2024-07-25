package reportvalidator

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metricsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reportvalidator/iface"
)

type ReportValidator struct {
	rules []iface.IReportValidatorRule
}

func New(rules []iface.IReportValidatorRule) *ReportValidator {
	return &ReportValidator{
		rules: rules,
	}
}

func NewWithAllRules(metricDAL *metricsDAL.MetricsFirestore) *ReportValidator {
	return &ReportValidator{
		rules: []iface.IReportValidatorRule{
			NewLimitTopBottomRule(),
			NewPromotionalCreditRule(),
			NewCalculatedMetricRule(metricDAL),
			NewTreemapsRule(),
			NewComparativeRule(),
			NewSplitRule(),
			NewCustomTimeRangeRule(),
		},
	}
}

func (rv *ReportValidator) Validate(ctx context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	for _, rule := range rv.rules {
		if ruleValidationErrors, err := rule.Validate(ctx, report); ruleValidationErrors != nil {
			if err != externalReportService.ErrValidation {
				return ruleValidationErrors, err
			}

			validationErrors = append(validationErrors, ruleValidationErrors...)
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

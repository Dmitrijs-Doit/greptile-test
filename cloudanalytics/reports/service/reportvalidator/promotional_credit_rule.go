package reportvalidator

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	externalReportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/externalreport"
)

var (
	validPromotionalCreditTimeIntervals = []domainReport.TimeInterval{
		domainReport.TimeIntervalMonth,
		domainReport.TimeIntervalQuarter,
		domainReport.TimeIntervalYear,
	}
)

type PromotionalCreditRule struct{}

func NewPromotionalCreditRule() *PromotionalCreditRule {
	return &PromotionalCreditRule{}
}

// Validate ensures that the promotionalCredit is only used
// together with valid time interval value
func (r *PromotionalCreditRule) Validate(_ context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error) {
	var validationErrors []errormsg.ErrorMsg

	promotionalCredit := report.Config.IncludeCredits
	interval := report.Config.TimeInterval

	if promotionalCredit && !slices.Contains(validPromotionalCreditTimeIntervals, interval) {
		validationErrors = append(validationErrors, errormsg.ErrorMsg{
			Field:   domainReport.ConfigTimeIntervalField,
			Message: fmt.Sprintf("%s: %s", ErrInvalidPromotionalCreditTimeInterval, interval),
		})
	}

	if len(validationErrors) > 0 {
		return validationErrors, externalReportService.ErrValidation
	}

	return nil, nil
}

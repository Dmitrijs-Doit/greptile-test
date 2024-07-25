//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type IReportValidatorService interface {
	Validate(ctx context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error)
}

type IReportValidatorRule interface {
	Validate(ctx context.Context, report *domainReport.Report) ([]errormsg.ErrorMsg, error)
}

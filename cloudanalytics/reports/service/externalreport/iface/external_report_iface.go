//go:generate mockery --output=../mocks --all

package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type IExternalReportService interface {
	UpdateReportWithExternalReport(
		ctx context.Context,
		customerID string,
		report *domainReport.Report,
		externalReport *domainExternalReport.ExternalReport,
	) (*domainReport.Report, []errormsg.ErrorMsg, error)
	NewExternalReportFromInternal(
		ctx context.Context,
		customerID string,
		report *domainReport.Report,
	) (*domainExternalReport.ExternalReport, []errormsg.ErrorMsg, error)
	MergeConfigWithExternalConfig(
		ctx context.Context,
		customerID string,
		config *domainReport.Config,
		externalConfig *domainExternalReport.ExternalConfig,
	) (*domainReport.Config, []errormsg.ErrorMsg, error)
}

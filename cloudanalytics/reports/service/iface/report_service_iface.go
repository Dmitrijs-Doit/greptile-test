//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	domainExternalAPI "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type IReportService interface {
	CreateReportWithExternal(
		ctx context.Context,
		externalReport *externalreport.ExternalReport,
		customerID string,
		email string,
	) (*externalreport.ExternalReport, []errormsg.ErrorMsg, error)
	GetReportConfig(
		ctx context.Context,
		reportID string,
		customerID string,
	) (*externalreport.ExternalReport, error)
	DeleteReport(ctx context.Context, customerID, requesterEmail, reportID string) error
	ShareReport(ctx context.Context, args report.ShareReportArgsReq) error
	UpdateReportWithExternal(
		ctx context.Context,
		reportID string,
		externalReportPayload *externalreport.ExternalReport,
		customerID string,
		email string,
	) (*externalreport.ExternalReport, []errormsg.ErrorMsg, error)
	RunReportFromExternalConfig(
		ctx context.Context,
		externalConfig *externalreport.ExternalConfig,
		customerID string,
		requesterEmail string,
	) (*domainExternalAPI.RunReportResult, []errormsg.ErrorMsg, error)
	DeleteMany(
		ctx context.Context,
		customerID string,
		email string,
		reportIDs []string,
	) error
}

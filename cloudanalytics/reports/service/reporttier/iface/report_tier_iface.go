//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
)

type ReportTierService interface {
	CheckAccessToQueryRequest(
		ctx context.Context,
		customerID string,
		qr *cloudanalytics.QueryRequest,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToReport(
		ctx context.Context,
		customerID string,
		report *report.Report,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToExternalReport(
		ctx context.Context,
		customerID string,
		externalReport *externalreport.ExternalReport,
		checkFeaturesAccess bool,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToReportType(
		ctx context.Context,
		customerID string,
		reportType string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToPresetReport(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToCustomReport(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToReportID(
		ctx context.Context,
		customerID string,
		reportID string,
	) (*domainTier.AccessDeniedError, error)
	GetCustomerEntitlementIDs(
		ctx context.Context,
		customerID string,
	) ([]string, error)
	CheckAccessToExtendedMetric(
		ctx context.Context,
		customerID string,
		extendedMetric string,
	) (*domainTier.AccessDeniedError, error)
}

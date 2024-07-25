//go:generate mockery --output=./mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
)

type WidgetService interface {
	DeleteReportWidget(
		ctx context.Context,
		customerID string,
		reportID string,
	) error
	DeleteReportsWidgets(
		ctx context.Context,
		customerID string,
		reportIDs []string,
	) error
	Save(
		ctx context.Context,
		minUpdateDelayMinutes int,
		request domain.ReportWidgetRequest,
	) error
	BuildWidgetDocID(customerID, orgID, reportID string) string
	RefreshReportWidget(ctx context.Context, requestParams *domain.ReportWidgetRequest) error
	UpdateReportWidget(ctx context.Context, requestParams *domain.ReportWidgetRequest) error
}

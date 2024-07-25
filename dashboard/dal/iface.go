//go:generate mockery --output=../mocks --all
package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/domain"
)

type Dashboards interface {
	GetDashboardsWithCloudReportsCustomerIDs(ctx context.Context) ([]string, error)
	GetCustomerDashboardsWithCloudReports(ctx context.Context, customerID string) ([]*dashboard.Dashboard, error)
	RemoveDashboardWidget(ctx context.Context, dashboardRef *firestore.DocumentRef, widget dashboard.DashboardWidget) error
	GetCustomerTicketStatistics(ctx context.Context, customerID string) ([]*dashboard.TicketSummary, error)
	GetDashboardsWithPaths(ctx context.Context, paths []string) ([]*dashboard.Dashboard, error)
}

type PublicDashboards interface {
	GetDashboardsWithCloudReportsCustomerIDs(ctx context.Context) ([]string, error)
	GetCustomerDashboardsWithCloudReports(ctx context.Context, customerID string) ([]*dashboard.Dashboard, error)
	UpdateReportWidgetDashboardsWidgetState(ctx context.Context, customerID string, reportID string, state dashboard.WidgetRefreshState) error
}

type DashboardAccessMetadata interface {
	ListCustomerDashboardAccessMetadata(ctx context.Context, customerID string) ([]*domain.DashboardAccessMetadata, error)
	GetDashboardAccessMetadata(ctx context.Context, customerID, orgID, dashboardID string) (*domain.DashboardAccessMetadata, error)
	SaveDashboardAccessMetadata(ctx context.Context, accessMetadata *domain.DashboardAccessMetadata) error
	UpdateTimeLastRefreshed(ctx context.Context, customerID, orgID, dashboardID string) error
	ShouldRefreshDashboard(
		ctx context.Context,
		customerID, orgID, dashboardID string,
		timeRefreshThreshold time.Duration,
	) (bool, *domain.DashboardAccessMetadata, error)
}

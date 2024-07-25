package widget

import (
	"context"
	"strings"
	"time"

	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"

	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func (s *ScheduledWidgetUpdateService) UpdateDashboards(ctx context.Context, request domainWidget.DashboardsWidgetUpdateRequest) error {
	l := s.loggerProvider(ctx)

	customerID := request.CustomerID

	var organizationID string

	if s.doitEmployeesService.IsDoitEmployee(ctx) {
		organizationID = organizations.PresetDoitOrgID
	} else if userID, ok := ctx.Value("userId").(string); ok && userID != "" {
		user, err := s.userDAL.Get(ctx, userID)
		if err != nil {
			return err
		}

		if len(user.Organizations) > 0 {
			organizationID = user.Organizations[0].ID
		} else {
			organizationID = organizations.RootOrgID
		}
	}

	uniqueDashboardPaths := slice.Unique(request.DashboardPaths)

	dashboards, err := s.dashboardsDAL.GetDashboardsWithPaths(ctx, uniqueDashboardPaths)
	if err != nil {
		return err
	}

	timeRefreshThreshold := getWidgetUpdateDashboardTimeRefreshThreshold(customerID)

	uniqueReportWidgets := make(map[string]bool)

	for _, dashboard := range dashboards {
		shouldRefreshDashboard, md, err := s.dashboardAccessMetadataDAL.ShouldRefreshDashboard(
			ctx,
			customerID,
			organizationID,
			dashboard.ID,
			timeRefreshThreshold,
		)
		if err != nil {
			return err
		}

		if !shouldRefreshDashboard {
			l.Infof("Skipping dashboard '%s', timeLastRefreshed is %s", dashboard.DocPath, md.TimeLastRefreshed)
			continue
		}

		var timeLastAccessed *time.Time

		if md != nil {
			timeLastAccessed = md.TimeLastAccessed
		}

		for _, widget := range dashboard.Widgets {
			if !strings.HasPrefix(widget.Name, widgetPrefix) {
				continue
			}

			widgetID, _, reportID, err := widget.ExtractInfoFromName()
			if err != nil {
				return err
			}

			if uniqueReportWidgets[widgetID] {
				continue
			}

			uniqueReportWidgets[widgetID] = true

			body := domainWidget.ReportWidgetRequest{
				CustomerID:       customerID,
				ReportID:         reportID,
				OrgID:            organizationID,
				DashboardPath:    dashboard.DocPath,
				TimeLastAccessed: timeLastAccessed,
				IsScheduled:      false,
			}

			if err := s.buildWidgetUpdateTask(ctx, body, true); err != nil {
				return err
			}
		}
	}

	return nil
}

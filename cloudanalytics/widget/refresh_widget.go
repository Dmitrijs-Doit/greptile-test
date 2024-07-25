package widget

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
)

const (
	minUpdateDelayMinutes = 10
)

func (s *WidgetService) RefreshReportWidget(ctx context.Context, requestParams *domain.ReportWidgetRequest) error {
	l := s.loggerProvider(ctx)

	if s.doitEmployeesService.IsDoitEmployee(ctx) {
		requestParams.OrgID = organizations.PresetDoitOrgID
	} else if userID, ok := ctx.Value("userId").(string); ok && userID != "" {
		user, err := s.userDAL.Get(ctx, userID)
		if err != nil {
			return err
		}

		if len(user.Organizations) > 0 {
			requestParams.OrgID = user.Organizations[0].ID
		} else {
			requestParams.OrgID = organizations.RootOrgID
		}
	}

	customerID := requestParams.CustomerID
	reportID := requestParams.ReportID

	err := s.publicDashboardsDAL.UpdateReportWidgetDashboardsWidgetState(ctx, customerID, reportID, dashboard.WidgetRefreshStateProcessing)
	if err != nil {
		return err
	}

	widgetState := dashboard.WidgetRefreshStateFailed

	defer func() {
		err := s.publicDashboardsDAL.UpdateReportWidgetDashboardsWidgetState(ctx, customerID, reportID, widgetState)
		if err != nil {
			l.Errorf("Failed to update widget state with error: %s", err)
		}
	}()

	if err := s.Save(ctx, minUpdateDelayMinutes, *requestParams); err != nil {
		return err
	}

	if requestParams.CustomerOrPresentationID != requestParams.CustomerID {
		requestParams.CustomerOrPresentationID = requestParams.CustomerID

		if err := s.Save(ctx, minUpdateDelayMinutes, *requestParams); err != nil {
			return err
		}
	}

	widgetState = dashboard.WidgetRefreshStateSuccess

	return nil
}

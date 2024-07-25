package widget

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	domainDashboard "github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/domain"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/publicdashboards"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	widgetPrefix    = "cloudReports::"
	widgetPrefixLen = len(widgetPrefix)

	widgetUpdateDashboardTimeRefreshThreshold = 6 * time.Hour
	widgetUpdateDashboardAccessThreshold      = 8 * times.WeekDuration

	disableDoitWidgetUpdates = true
)

func getWidgetUpdateDashboardTimeRefreshThreshold(customerID string) time.Duration {
	// CSP does not need to update the dashboard as frequently as other customers
	if customerID == domainQuery.CSPCustomerID {
		return 4 * widgetUpdateDashboardTimeRefreshThreshold
	}

	return widgetUpdateDashboardTimeRefreshThreshold

}

func (s *ScheduledWidgetUpdateService) UpdateAllCustomerDashboardReportWidgets(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	customersIDs, err := s.getDashboardCustomers(ctx)
	if err != nil {
		return err
	}

	customers, err := s.customersDAL.GetCustomersByIDs(ctx, customersIDs)
	if err != nil {
		return err
	}

	for _, customer := range customers {
		if customer.Terminated() {
			continue
		}

		customerID := customer.Snapshot.Ref.ID

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/analytics/widgets/customers/%s", customerID),
			Queue:  common.TaskQueueCloudAnalyticsCustomerDashboards,
		}

		if _, err = s.cloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil)); err != nil {
			l.Errorf("UpdateAllCustomerDashboardReportWidgets, failed to create task for customer: %s, error: %v", customerID, err)
			return err
		}
	}

	return nil
}

func (s *ScheduledWidgetUpdateService) getDashboardCustomers(ctx context.Context) ([]string, error) {
	privateDashboardCustomers, err := s.dashboardsDAL.GetDashboardsWithCloudReportsCustomerIDs(ctx)
	if err != nil {
		return nil, err
	}

	publicDashboardCustomers, err := s.publicDashboardsDAL.GetDashboardsWithCloudReportsCustomerIDs(ctx)
	if err != nil {
		return nil, err
	}

	uniqCustomers := slice.Unique(append(privateDashboardCustomers, publicDashboardCustomers...))

	return uniqCustomers, nil
}

func (s *ScheduledWidgetUpdateService) UpdateCustomerDashboardReportWidgetsHandler(ctx context.Context, customerID, orgID string) error {
	usePrioritizedQueue := orgID != ""

	privateDashboards, err := s.dashboardsDAL.GetCustomerDashboardsWithCloudReports(ctx, customerID)
	if err != nil {
		return err
	}

	publicDashboards, err := s.publicDashboardsDAL.GetCustomerDashboardsWithCloudReports(ctx, customerID)
	if err != nil {
		return err
	}

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if customer.Terminated() {
		return ErrTerminatedCustomer
	}

	orgs, err := s.customersDAL.GetCustomerOrgs(ctx, customerID, orgID)
	if err != nil {
		return err
	}

	var (
		customerOrgs        []string
		uniqueReportWidgets = make(map[string]bool)
	)

	for _, org := range orgs {
		// Skip preset orgs that are not relevant for the customer assets
		if organizations.IsPresetOrg(org.Snapshot.Ref.ID) {
			// skip partner orgs from scheduled widget update
			if !usePrioritizedQueue && organizations.IsPartnerOrg(org.Snapshot.Ref.ID) {
				continue
			}

			orgAssetRestrictions := organizations.GetPresetOrgCloudProviderRestriction(org.Snapshot.Ref.ID)
			if len(orgAssetRestrictions) > 0 && !slice.ContainsAny(customer.Assets, orgAssetRestrictions) {
				continue
			}
		}

		customerOrgs = append(customerOrgs, org.Snapshot.Ref.ID)
	}

	dashboardAccessMetadata, err := s.dashboardAccessMetadataDAL.ListCustomerDashboardAccessMetadata(ctx, customerID)
	if err != nil {
		return err
	}

	dashboardAccessMetadataMap := make(map[string]*domain.DashboardAccessMetadata)

	for _, v := range dashboardAccessMetadata {
		k := getDashboardAccessMetadataKey(v.OrganizationID, v.DashboardID)
		if _, ok := dashboardAccessMetadataMap[k]; !ok {
			dashboardAccessMetadataMap[k] = v
		}
	}

	s.processDashboards(ctx, customerID, privateDashboards, customerOrgs, uniqueReportWidgets, dashboardAccessMetadataMap, usePrioritizedQueue)
	s.processDashboards(ctx, customerID, publicDashboards, customerOrgs, uniqueReportWidgets, dashboardAccessMetadataMap, usePrioritizedQueue)

	return nil
}

func (s *ScheduledWidgetUpdateService) processDashboards(
	ctx context.Context,
	customerID string,
	dashboards []*domainDashboard.Dashboard,
	customerOrgs []string,
	uniqueReportWidgets map[string]bool,
	dashboardOrgAccessMap map[string]*domain.DashboardAccessMetadata,
	usePrioritizedQueue bool,
) {
	defaultTimeLastAccessed := times.CurrentDayUTC().Add(-widgetUpdateDashboardAccessThreshold).Add(times.DayDuration * 3)

	for _, dashboard := range dashboards {
		cloudProvider := publicdashboards.GetPublicDashboardCloudProvider(dashboard.DashboardType)

		for _, orgID := range customerOrgs {
			// do not update doit widgets for non-pulse dashboards
			if disableDoitWidgetUpdates && orgID == organizations.PresetDoitOrgID && dashboard.DashboardType != publicdashboards.Pulse {
				continue
			}
			// Skip dashboards and preset orgs that are not compatible with each other
			if !organizations.IsOrgAllowedAssetType(orgID, cloudProvider) {
				continue
			}

			mdKey := getDashboardAccessMetadataKey(orgID, dashboard.ID)

			var timeLastAccessed *time.Time

			if md, ok := dashboardOrgAccessMap[mdKey]; ok {
				// If the organization dashboard was refreshed recently, skip it
				if md.TimeLastRefreshed != nil && time.Since(*md.TimeLastRefreshed) < getWidgetUpdateDashboardTimeRefreshThreshold(customerID) {
					continue
				}

				timeLastAccessed = md.TimeLastAccessed
			} else {
				// If dashboard access metadata does not exist for this org then create it with a default time
				dashboardAccessMetadata := &domain.DashboardAccessMetadata{
					CustomerID:        dashboard.CustomerID,
					OrganizationID:    orgID,
					DashboardID:       dashboard.ID,
					TimeLastAccessed:  &defaultTimeLastAccessed,
					TimeLastRefreshed: &defaultTimeLastAccessed,
				}

				if err := s.dashboardAccessMetadataDAL.SaveDashboardAccessMetadata(ctx, dashboardAccessMetadata); err != nil {
					s.loggerProvider(ctx).Errorf("failed to save dashboard access metadata for dashboard %s with error: %s", dashboard.ID, err)
				}
			}

			s.processWidgets(ctx, uniqueReportWidgets, orgID, dashboard, timeLastAccessed, usePrioritizedQueue)
		}
	}
}

func (s *ScheduledWidgetUpdateService) processWidgets(
	ctx context.Context,
	uniqueReportWidgets map[string]bool,
	orgID string,
	dashboard *domainDashboard.Dashboard,
	timeLastAccessed *time.Time,
	usePrioritizedQueue bool,
) {
	l := s.loggerProvider(ctx)

	for _, widget := range dashboard.Widgets {
		if disableDoitWidgetUpdates && orgID == organizations.PresetDoitOrgID {
			if !strings.Contains(widget.Name, MonthVsLastReport) {
				continue
			}
		}

		err := s.processWidget(
			ctx,
			uniqueReportWidgets,
			orgID,
			dashboard,
			widget,
			timeLastAccessed,
			usePrioritizedQueue,
		)
		if err != nil {
			switch err {
			case ErrCustomerWidgetMismatch:
				l.Warningf("dashboard [%s] is different than widget customer %s", dashboard.Ref.Path, widget.Name)

				if err := s.dashboardsDAL.RemoveDashboardWidget(ctx, dashboard.Ref, widget); err != nil {
					l.Errorf("failed to remove dashboard widget path %s with error: %s", dashboard.Ref.Path, err.Error())
				}
			default:
				l.Errorf("dashboard [%s] process widget %s failed with error: %s", dashboard.Ref.Path, widget.Name, err)
			}
		}
	}
}

func (s *ScheduledWidgetUpdateService) processWidget(
	ctx context.Context,
	uniqueReportWidgets map[string]bool,
	orgID string,
	dashboard *domainDashboard.Dashboard,
	widget domainDashboard.DashboardWidget,
	timeLastAccessed *time.Time,
	usePrioritizedQueue bool,
) error {
	if !strings.HasPrefix(widget.Name, widgetPrefix) {
		return nil
	}

	widgetID, customerID, reportID, err := widget.ExtractInfoFromName()
	if err != nil {
		return err
	}

	if len(orgID) > 0 && orgID != organizations.RootOrgID {
		widgetID = fmt.Sprintf("%s_%s_%s", customerID, orgID, reportID)
	}

	if _, ok := uniqueReportWidgets[widgetID]; ok {
		return nil
	}

	uniqueReportWidgets[widgetID] = true

	if customerID == "" || dashboard.CustomerID == "" {
		return ErrMissingCustomerID
	}

	if dashboard.CustomerID != customerID {
		return ErrCustomerWidgetMismatch
	}

	body := domainWidget.ReportWidgetRequest{
		DashboardPath:    dashboard.DocPath,
		CustomerID:       customerID,
		ReportID:         reportID,
		OrgID:            orgID,
		TimeLastAccessed: timeLastAccessed,
		IsScheduled:      true,
	}

	return s.buildWidgetUpdateTask(ctx, body, usePrioritizedQueue)
}

func (s *ScheduledWidgetUpdateService) buildWidgetUpdateTask(ctx context.Context, body domainWidget.ReportWidgetRequest, usePrioritizedQueue bool) error {
	if body.TimeLastAccessed != nil {
		body.TimeLastAccessedString = body.TimeLastAccessed.Format(time.RFC3339)
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   fmt.Sprintf("/tasks/analytics/widgets/customers/%s/singleWidget", body.CustomerID),
		Queue:  common.TaskQueueCloudAnalyticsWidgets,
	}

	isPrioritized, _ := ctx.Value(domainWidget.IsPrioritized).(string)

	if usePrioritizedQueue || isPrioritized != "" {
		config.Queue = common.TaskQueueCloudAnalyticsWidgetsPrioritized
		if isPrioritized == "delayed" {
			config.ScheduleTime = common.TimeToTimestamp(time.Now().UTC().Add(10 * time.Minute))
		}
	}

	if _, err := s.cloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(body)); err != nil {
		return fmt.Errorf("failed to create task for widget: %s, error: %v", body.ReportID, err)
	}

	return nil
}

// getDashboardAccessMetadataKey returns the key for the dashboard access metadata map
func getDashboardAccessMetadataKey(orgID, dashboardID string) string {
	return fmt.Sprintf("%s_%s", orgID, dashboardID)
}

package widget

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	domainReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	domainDashboard "github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	expireWidgetMonths          = 3
	minScheduleHoursRefreshRate = 12
)

const (
	AWSLensReport     = "mkzMt3cHTPC14WRH0eER"
	GCPLensReport     = "ss2m7rGY0OjuPDyJub6g"
	AzureLensReport   = "pnjzUHb3FNleMJMpcOcG"
	MonthVsLastReport = "Anl2FHDAgyxR4GFellrA"
)

// highPriorityWidgets are the widgets that should be updated more frequently
// those include the Spend history reports for the AWS,GCP,Azure lenses and This Month vs. Last report
var highPriorityWidgets = []string{
	AWSLensReport,
	GCPLensReport,
	AzureLensReport,
	MonthVsLastReport,
}

func (s *WidgetService) Save(
	ctx context.Context,
	minUpdateDelayMinutes int,
	request domain.ReportWidgetRequest,
) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	if err := s.analyticsConfigs.initialize(ctx); err != nil {
		return err
	}

	customerID := request.CustomerID
	reportID := request.ReportID
	orgID := request.OrgID
	email := request.Email
	isScheduled := request.IsScheduled
	timeLastAccessed := request.TimeLastAccessed

	l.Infof("Refreshing widget for report '%s' on dashboard '%s' for organization '%s'", reportID, request.DashboardPath, orgID)

	report, err := s.cloudAnalytics.GetReport(ctx, customerID, reportID, false)
	if err != nil {
		if codes.NotFound == status.Code(err) {
			l.Infof("report on dashboard '%s' was not found", request.DashboardPath)
			return s.handleNotFoundError(ctx, request)
		}

		return err
	}

	// If a report has an organization, we should only create a widget for that organization.
	// otherwise, create widget based on the orgID supplied to the function.
	var orgRef *firestore.DocumentRef
	if report.Organization != nil {
		orgRef = report.Organization
	} else if orgID == "" {
		orgRef = fs.Collection("customers").Doc(customerID).Collection("customerOrgs").Doc(organizations.RootOrgID)
	} else if organizations.IsPresetOrg(orgID) {
		orgRef = fs.Collection("organizations").Doc(orgID)
	} else {
		orgRef = fs.Collection("customers").Doc(customerID).Collection("customerOrgs").Doc(orgID)
	}

	organization, err := common.GetOrganization(ctx, orgRef)
	if err != nil {
		return err
	}

	orgID = orgRef.ID

	customerIDForWidgetDocID := request.CustomerID
	if report.Type == domainReport.ReportTypePreset {
		customerIDForWidgetDocID = request.CustomerOrPresentationID
	}

	widgetDocID := s.BuildWidgetDocID(customerIDForWidgetDocID, orgID, reportID)
	widgetReportRef := fs.Collection("cloudAnalytics").
		Doc("widgets").
		Collection("cloudAnalyticsWidgets").
		Doc(widgetDocID)

	timeRefreshed, err := s.getWidgetReportTimeRefreshed(ctx, widgetReportRef)
	if err != nil {
		return err
	}

	if ok := s.shouldUpdateWidget(timeRefreshed, organization, report, minUpdateDelayMinutes); !ok {
		l.Info("widget is up to date")
		return nil
	}

	wasUpdatedInTheLastWeek := time.Since(timeRefreshed) < times.WeekDuration
	isHighPriorityWidget := slice.Contains(highPriorityWidgets, reportID)

	// If this is a scheduled widget update and the widget was updated in the last week
	if isScheduled && !isHighPriorityWidget && wasUpdatedInTheLastWeek {
		if timeLastAccessed != nil {
			// Do not update the widget if the dashboard was not accessed according to the threshold
			if time.Since(*timeLastAccessed) > widgetUpdateDashboardAccessThreshold {
				l.Infof("dashboard was not accessed in the last %v, skipping widget update", widgetUpdateDashboardAccessThreshold)
				return nil
			}
		} else {
			// Do not update the widget if there is no dashboard access time
			l.Infof("dashboard access time is not provided, skipping widget update")
			return nil
		}
	}

	queryRequest, err := s.cloudAnalytics.NewQueryRequestFromFirestoreReport(ctx, customerID, report)
	if err != nil {
		return err
	}

	if err := s.reportDAL.UpdateTimeLastRun(ctx, reportID, domainOrigin.QueryOriginWidgets); err != nil {
		l.Errorf("failed to update last time run for report %v; %s", reportID, err)
	}

	queryRequest.Organization = orgRef

	queryResults, err := s.cloudAnalytics.GetQueryResult(ctx, queryRequest, request.CustomerOrPresentationID, email)

	if err != nil {
		// if no tables found, it means the customer has no assets, do not retry task in case of this error
		if errors.As(err, &service.ErrNoTablesFound{}) {
			l.Infof("no tables found for customer %s, skipping widget update", customerID)
			return nil
		}

		// If the query started but one or more tables are not found, log a warning and exit
		// without retrying. This is to prevent error log spamming.
		var gapiError *googleapi.Error

		if errors.As(err, &gapiError) {
			if gapiError.Code == http.StatusNotFound {
				l.Warningf("failed to run query result with error %s", err)
				return nil
			}
		}

		return err
	}

	if queryRequest.Type == "report" {
		if err := s.reportStatsService.UpdateReportStats(ctx, queryRequest.ID, queryRequest.Origin, queryResults.Details); err != nil {
			l.Errorf("failed to update report stats for report %s with error %s", queryRequest.ID, err)
		}
	}

	wr := s.newWidgetReport(ctx, report, queryRequest, queryResults)
	if _, err := widgetReportRef.Set(ctx, wr); err != nil {
		return err
	}

	l.Info("widget updated successfully")

	return nil
}

func (s *WidgetService) getWidgetReportTimeRefreshed(ctx context.Context, widgetReportRef *firestore.DocumentRef) (time.Time, error) {
	widgetReportTimeRefreshed := struct {
		TimeRefreshed time.Time `firestore:"timeRefreshed,serverTimestamp"`
	}{}

	widgetReportSnap, err := widgetReportRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			err = nil
		}

		return widgetReportTimeRefreshed.TimeRefreshed, err
	}

	err = widgetReportSnap.DataTo(&widgetReportTimeRefreshed)

	return widgetReportTimeRefreshed.TimeRefreshed, err
}

func (s *WidgetService) shouldUpdateWidget(widgetReportTimeRefreshed time.Time, organization *common.Organization, r *domainReport.Report, minUpdateDelayMinutes int) bool {
	if widgetReportTimeRefreshed.IsZero() {
		return true
	} else if widgetReportTimeRefreshed.Before(r.TimeModified) {
		return true
	} else if minUpdateDelayMinutes != 0 {
		minutes := time.Since(widgetReportTimeRefreshed).Minutes()
		return minUpdateDelayMinutes < int(minutes)
	}

	orgID := organization.Snapshot.Ref.ID

	// Root org and doit org refresh rate
	if orgID == organizations.RootOrgID || orgID == organizations.PresetDoitOrgID {
		// If the report was not modified after it was last refreshed
		// ,and it was refreshed in the last hour then don't refresh widget again
		return time.Since(widgetReportTimeRefreshed).Hours() > minScheduleHoursRefreshRate
	}

	// AWS/GCP and partners preset orgs refresh rate
	if organizations.IsPresetOrg(orgID) {
		return time.Since(widgetReportTimeRefreshed).Hours() > 2*minScheduleHoursRefreshRate
	}

	return s.shouldUpdateWidgetForAccessAndRefreshTime(organization.LastAccessed, widgetReportTimeRefreshed)
}

func (s *WidgetService) shouldUpdateWidgetForAccessAndRefreshTime(orgLastAccessed time.Time, timeRefreshed time.Time) bool {
	if orgLastAccessed.IsZero() {
		return false
	}

	var maxWidgetUpdateHours int

	switch days := int(time.Since(orgLastAccessed).Hours() / 24); {
	case days < 3:
		maxWidgetUpdateHours = minScheduleHoursRefreshRate
	case days < 7:
		maxWidgetUpdateHours = 3 * 24
	case days < 14:
		maxWidgetUpdateHours = 7 * 24
	case days < 30:
		maxWidgetUpdateHours = 14 * 24
	case days > 30:
		maxWidgetUpdateHours = days / 2 * 24
	}

	timeRefreshedHours := int(time.Since(timeRefreshed).Hours())

	return timeRefreshedHours > maxWidgetUpdateHours
}

func (s *WidgetService) getRowsMap(rows *[][]bigquery.Value) (*map[string]interface{}, int64) {
	var size int64

	rowsMap := make(map[string]interface{})
	for i, row := range *rows {
		rowsMap[strconv.Itoa(i)] = row
		size += int64(len(row))
	}

	return &rowsMap, size
}

func (s *WidgetService) newWidgetReport(ctx context.Context, r *domainReport.Report, queryRequest *cloudanalytics.QueryRequest, queryResults cloudanalytics.QueryResult) *domain.WidgetReport {
	var wr domain.WidgetReport

	rowsMap, size := s.getRowsMap(&queryResults.Rows)
	forecastRowsMap, forecastSize := s.getRowsMap(&queryResults.ForecastRows)

	// General
	wr.Customer = r.Customer
	wr.CustomerID = r.Customer.ID
	wr.Organization = queryRequest.Organization
	wr.Report = s.reportDAL.GetRef(ctx, r.ID)
	wr.Name = r.Name
	wr.Description = s.getWidgetDescription(r.Description, r.Config.TimeSettings)
	wr.TimeRefreshed = time.Time{}
	wr.IsPublic = r.Public != nil

	wr.Collaborators = r.Collaborators
	wr.Type = r.Type
	wr.Size = size + forecastSize

	// Data
	wr.Data.Rows = *rowsMap
	wr.Data.ForecastRows = *forecastRowsMap

	// Config
	wr.Config.Rows = queryRequest.Rows
	wr.Config.Cols = queryRequest.Cols
	wr.Config.Splits = r.Config.Splits
	wr.Config.Metric = queryRequest.Metric
	wr.Config.ExtendedMetric = queryRequest.ExtendedMetric
	wr.Config.CalculatedMetric = s.getWidgetCalculatedMetric(queryRequest.CalculatedMetric)
	wr.Config.Count = queryRequest.Count
	wr.Config.ExtendedMetricType = s.getExtendedMetricType(r.Config.ExtendedMetric)
	wr.Config.Currency = queryRequest.Currency
	wr.Config.Timezone = queryRequest.Timezone
	wr.Config.RowOrder = r.Config.RowOrder
	wr.Config.ColOrder = r.Config.ColOrder
	wr.Config.Features = r.Config.Features
	wr.Config.Aggregator = r.Config.Aggregator
	wr.Config.Comparative = r.Config.Comparative
	wr.Config.Renderer = s.getWidgetRenderer(r.Config.Renderer)
	wr.Config.ExcludePartialData = r.Config.ExcludePartialData
	wr.Config.LogScale = r.Config.LogScale
	wr.Config.IncludeSubtotals = r.Config.IncludeSubtotals
	wr.Config.DataSource = r.Config.DataSource

	s.SetExpireBy(&wr)

	return &wr
}

var e2eTestCustomers = []string{
	"LcgELbXV21Imef3utMoh",
	"bi4Ehb5WS1LOgXmtW8L5",
	"bA3mtbUwaouS3dzZeg6T",
	"L4gRv3ufVS3e7mXOeBbl",
	"U9nLdJCQfGZkCupDfNcE",
	"brrhJu7TOI0o640tSONA",
	"JhV7WydpOlW8DeVRVVNf",
	"EE8CtpzYiKp0dVAESVrB",
}

// SetExpireBy sets the expiration date for the widgets. Test customers have a longer expiration date.
func (s *WidgetService) SetExpireBy(wr *domain.WidgetReport) {
	today := times.CurrentDayUTC()

	if slice.Contains(e2eTestCustomers, wr.CustomerID) {
		wr.ExpireBy = today.AddDate(0, 8*expireWidgetMonths, 0)
	} else {
		wr.ExpireBy = today.AddDate(0, expireWidgetMonths, 0)
	}
}

func (s *WidgetService) GetWidgetReport(ctx context.Context, customerID, orgID, reportID string) (*domain.WidgetReport, error) {
	fs := s.conn.Firestore(ctx)

	widgetDocID := s.BuildWidgetDocID(customerID, orgID, reportID)
	widgetReportRef := fs.Collection("cloudAnalytics").
		Doc("widgets").
		Collection("cloudAnalyticsWidgets").
		Doc(widgetDocID)

	widgetReportSnap, err := widgetReportRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var wr domain.WidgetReport
	if err := widgetReportSnap.DataTo(&wr); err != nil {
		return nil, err
	}

	return &wr, nil
}

// getWidgetCalculatedMetric saves a lean version of the calculated metric definition with
// only the fields required for formatting (format and the variable's metric type).
// The formula and variables definition are not required.
func (s *WidgetService) getWidgetCalculatedMetric(cm *cloudanalytics.QueryRequestCalculatedMetric) *cloudanalytics.QueryRequestCalculatedMetric {
	var r *cloudanalytics.QueryRequestCalculatedMetric
	if cm != nil {
		r = &cloudanalytics.QueryRequestCalculatedMetric{
			Format:    cm.Format,
			Variables: make([]*cloudanalytics.QueryRequestMetricAttribution, len(cm.Variables)),
		}
		for i := 0; i < len(cm.Variables); i++ {
			r.Variables[i] = &cloudanalytics.QueryRequestMetricAttribution{Metric: cm.Variables[i].Metric}
		}
	}

	return r
}

func (s *WidgetService) getWidgetRenderer(renderer domainReport.Renderer) domainReport.Renderer {
	switch renderer {
	// Unsupported widget renderers
	case
		domainReport.RendererTableSheetsExport, domainReport.RendererTableCSVExport:
		return domainReport.RendererStackedColumnChart
	default:
		return renderer
	}
}

func (s *WidgetService) BuildWidgetDocID(customerID, orgID, reportID string) string {
	if orgID == "" || organizations.IsRootOrg(orgID) {
		return fmt.Sprintf("%s_%s", customerID, reportID)
	}

	return fmt.Sprintf("%s_%s_%s", customerID, orgID, reportID)
}

func (s *WidgetService) getExtendedMetricType(extendedMetric string) string {
	if extendedMetric != "" {
		for _, m := range *s.analyticsConfigs.extendedMetrics {
			if m.Key == extendedMetric {
				return m.Type
			}
		}
	}

	return ""
}

func (s *WidgetService) getWidgetDescription(description string, timeSettings *domainReport.TimeSettings) string {
	if description == "" {
		return timeSettings.String()
	}

	return description
}

func (s *WidgetService) handleNotFoundError(ctx context.Context, request domain.ReportWidgetRequest) error {
	l := s.loggerProvider(ctx)

	l.Infof("attemptig to remove widget from dashboard: '%s'", request.DashboardPath)

	if request.DashboardPath == "" {
		l.Warningf("dashboard path is empty, cannot remove widget")
		return nil
	}

	dashboards, err := s.dashboardsDAL.GetDashboardsWithPaths(ctx, []string{request.DashboardPath})
	if err != nil {
		return err
	}

	if len(dashboards) != 1 {
		return fmt.Errorf("failed to get dashboard with path '%s'", request.DashboardPath)
	}

	dashboard := dashboards[0]

	// check for public dashboards, do not remove widgets from public dashboards
	if dashboard.DashboardType != "" {
		l.Info("dashboard is public, not removing widget")
		return nil
	}

	widgetID := s.BuildWidgetDocID(request.CustomerID, request.OrgID, request.ReportID)
	widgetName := fmt.Sprintf("%s%s", widgetPrefix, widgetID)

	var widget domainDashboard.DashboardWidget

	for _, w := range dashboard.Widgets {
		if w.Name == widgetName {
			widget = w
			break
		}
	}

	if err := s.dashboardsDAL.RemoveDashboardWidget(ctx, dashboard.Ref, widget); err != nil {
		l.Errorf("failed to remove dashboard widget path %s with error: %s", dashboard.Ref.Path, err)
		return err
	}

	l.Info("widget removed from dashboard")

	return nil
}

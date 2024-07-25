package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	cloudTasksIface "github.com/doitintl/cloudtasks/iface"
	budgets "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/domain"
	domainHighCharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/domain"
	highcharts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service"
	widget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	domainWidget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	dashboardDomain "github.com/doitintl/hello/scheduled-tasks/dashboard"
	dashboardDal "github.com/doitintl/hello/scheduled-tasks/dashboard/dal"
	dashboardAccessDomain "github.com/doitintl/hello/scheduled-tasks/dashboard/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	ncDomain "github.com/doitintl/notificationcenter/domain"
	nc "github.com/doitintl/notificationcenter/pkg"
	ncService "github.com/doitintl/notificationcenter/service"
)

const (
	processWidgetsConcurrencyLimit = 3
	dashboardRefreshThreshold      = time.Hour
	notificationTemplate           = "3HDR59WFW94QGWQXC3YQJWE7C43M"
)

type getDashboardsFunc func(ctx context.Context, paths []string) ([]*dashboardDomain.Dashboard, error)
type getReportImageFunc func(ctx context.Context, reportID, customerID string, highchartsFontSettings *domainHighCharts.HighchartsFontSettings) (string, error)
type createSendTaskFunc func(ctx context.Context, notification nc.Notification, opts ...nc.SendTaskOption) (cloudTasksIface.Task, error)
type shouldRefreshDashboardFunc func(
	ctx context.Context,
	customerID, orgID, dashboardID string,
	timeRefreshThreshold time.Duration,
) (bool, *dashboardAccessDomain.DashboardAccessMetadata, error)

type notificationService interface {
	GetCustomerNotificationsConfig(ctx context.Context, customerID string, configID string) (*ncDomain.NotificationConfig, error)
	GetCustomersNotifications(configs []ncDomain.NotificationConfig) map[string]*nc.Notification
}

type widgetService interface {
	RefreshReportWidget(ctx context.Context, requestParams *domainWidget.ReportWidgetRequest) error
	GetWidgetReport(ctx context.Context, customerID, orgID, reportID string) (*domainWidget.WidgetReport, error)
}

type Service struct {
	l logger.ILogger
	notificationService
	widgetService
	getDashboards          getDashboardsFunc
	shouldRefreshDashboard shouldRefreshDashboardFunc
	getReportImage         getReportImageFunc
	createSendTask         createSendTaskFunc
}

func NewService(conn *connection.Connection, lp logger.Provider) *Service {
	ctx := context.Background()

	bService, err := budgets.NewBudgetsService(lp, conn)
	if err != nil {
		panic(err)
	}

	cg, err := highcharts.NewHighcharts(lp, conn, bService)
	if err != nil {
		panic(err)
	}

	ws, err := widget.NewWidgetService(lp, conn)
	if err != nil {
		panic(err)
	}

	ncc, err := nc.NewClient(ctx, common.ProjectID)
	if err != nil {
		panic(err)
	}

	return &Service{
		lp(ctx),
		ncService.NewRecipientsService(conn.Firestore(ctx)),
		ws,
		dashboardDal.NewDashboardsFirestoreWithClient(conn.Firestore).GetDashboardsWithPaths,
		dashboardDal.NewDashboardAccessMetadataFirestoreWithClient(conn.Firestore).ShouldRefreshDashboard,
		cg.GetReportImage,
		ncc.CreateSendTask,
	}
}

func (s *Service) HandleDashboardSubscription(ctx context.Context, request domain.HandleReportSubscriptionRequest) error {
	s.l.Debugf("Handling dashboard subscription %v", request)

	subscriptionData, err := s.getSubscriptionData(ctx, request)
	if err != nil {
		return err
	}

	widgets, err := s.processWidgets(ctx, subscriptionData)
	if err != nil {
		return err
	}

	return s.sendNotification(ctx, widgets, subscriptionData)
}

func (s *Service) getSubscriptionData(ctx context.Context, request domain.HandleReportSubscriptionRequest) (domain.SubscriptionData, error) {
	config, err := s.GetCustomerNotificationsConfig(ctx, request.CustomerID, request.ConfigID)
	if err != nil {
		return domain.SubscriptionData{}, err
	}

	configSettings, ok := config.SelectedNotifications[fmt.Sprint(ncDomain.NotificationDashboardSubscription)]
	if !ok {
		return domain.SubscriptionData{}, fmt.Errorf("dashboard subscription not found in notification config")
	}

	if configSettings == nil ||
		configSettings.DashboardSubscription == nil ||
		configSettings.DashboardSubscription.DashboardPath == "" ||
		configSettings.DashboardSubscription.OrganizationID == "" ||
		configSettings.DashboardSubscription.NextAt == nil {
		return domain.SubscriptionData{}, fmt.Errorf("invalid notification config %v", configSettings)
	}

	notification, ok := s.GetCustomersNotifications([]ncDomain.NotificationConfig{*config})[request.CustomerID]
	if !ok {
		return domain.SubscriptionData{}, fmt.Errorf("notification not found for customer %s", request.CustomerID)
	}

	customerID := request.CustomerID

	dashboards, err := s.getDashboards(ctx, []string{configSettings.DashboardSubscription.DashboardPath})
	if err != nil {
		return domain.SubscriptionData{}, err
	}

	dashboard := dashboards[0]

	shouldRefreshDashboard, md, err := s.shouldRefreshDashboard(
		ctx,
		customerID,
		configSettings.DashboardSubscription.OrganizationID,
		dashboard.ID,
		dashboardRefreshThreshold,
	)
	if err != nil {
		return domain.SubscriptionData{}, err
	}

	var timeLastAccessed *time.Time

	if md != nil {
		timeLastAccessed = md.TimeLastAccessed
	}

	s.l.Debugf("shouldRefreshDashboard: %v timeLastAccessed: %v, timeRefreshThreshold: %v", shouldRefreshDashboard, timeLastAccessed, dashboardRefreshThreshold)

	return domain.SubscriptionData{
		Notification:           notification,
		Dashboard:              dashboard,
		ScheduleTime:           *configSettings.DashboardSubscription.NextAt,
		OrganizationID:         configSettings.DashboardSubscription.OrganizationID,
		ShouldRefreshDashboard: shouldRefreshDashboard,
		TimeLastAccessed:       timeLastAccessed,
		CustomerID:             customerID,
	}, nil
}

// processWidgets processes widgets in the dashboard concurrently
func (s *Service) processWidgets(ctx context.Context, subscriptionData domain.SubscriptionData) ([]domain.NotificationWidgetItem, error) {
	var wg sync.WaitGroup

	dashboardWidgets := subscriptionData.Dashboard.Widgets
	errCh := make(chan error, len(dashboardWidgets))
	resultCh := make(chan domain.NotificationWidgetItem, len(dashboardWidgets))
	sem := make(chan struct{}, processWidgetsConcurrencyLimit)

	for _, widget := range dashboardWidgets {
		if !strings.HasPrefix(widget.Name, "cloudReports::") {
			s.l.Warningf("Skipping widget %s", widget.Name)
			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(widget dashboardDomain.DashboardWidget) {
			defer wg.Done()
			defer func() { <-sem }()

			widgetItem, err := s.processWidget(
				ctx,
				widget,
				subscriptionData,
			)

			if err != nil {
				errCh <- err
				return
			}

			resultCh <- widgetItem
		}(widget)
	}

	wg.Wait()
	close(errCh)
	close(resultCh)

	errors := make([]error, 0, len(dashboardWidgets))

	for err := range errCh {
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("errors processing widgets: %v", errors)
	}

	var result []domain.NotificationWidgetItem
	for widgetItem := range resultCh {
		result = append(result, widgetItem)
	}

	return result, nil
}

// processWidget processes a single widget,
// gets widget data, refreshes it if needed, generates a chart image and returns the widget item
func (s *Service) processWidget(
	ctx context.Context,
	widget dashboardDomain.DashboardWidget,
	subscriptionData domain.SubscriptionData,
) (domain.NotificationWidgetItem, error) {

	_, _, reportID, err := widget.ExtractInfoFromName()
	if err != nil {
		return domain.NotificationWidgetItem{}, err
	}

	report, err := s.GetWidgetReport(ctx, subscriptionData.CustomerID, subscriptionData.OrganizationID, reportID)
	if err != nil {
		return domain.NotificationWidgetItem{}, err
	}

	widgetItem := domain.NotificationWidgetItem{
		Name:        report.Name,
		Description: report.Description,
		ReportID:    reportID,
	}

	if subscriptionData.ShouldRefreshDashboard {
		s.l.Debugf("Requesting refresh for widget report %s", reportID)

		req := domainWidget.ReportWidgetRequest{
			CustomerID:               subscriptionData.CustomerID,
			CustomerOrPresentationID: subscriptionData.CustomerID,
			ReportID:                 reportID,
			OrgID:                    subscriptionData.OrganizationID,
			DashboardPath:            subscriptionData.Dashboard.DocPath,
			TimeLastAccessed:         subscriptionData.TimeLastAccessed,
			IsScheduled:              false,
		}
		if err := s.RefreshReportWidget(ctx, &req); err != nil {
			return domain.NotificationWidgetItem{}, err
		}
	}

	imageURL, err := s.getReportImage(ctx, reportID, subscriptionData.CustomerID, &domainHighCharts.DashboardSubscriptionSettings)
	if err != nil {
		return domain.NotificationWidgetItem{}, err
	}

	widgetItem.ImageURL = imageURL

	return widgetItem, nil
}

func (s *Service) sendNotification(ctx context.Context, widgets []domain.NotificationWidgetItem, subscriptionData domain.SubscriptionData) error {
	notification := subscriptionData.Notification
	notification.Data = map[string]interface{}{
		"widgets":       widgets,
		"dashboardName": subscriptionData.Dashboard.Name,
		"dashboardURL":  fmt.Sprintf("https://%s/customers/%s/dashboards/%s", common.Domain, subscriptionData.CustomerID, subscriptionData.Dashboard.Name),
		"consoleURL":    fmt.Sprintf("https://%s", common.Domain),
		"date":          subscriptionData.ScheduleTime.Format("Monday, _2 Jan 2006"),
	}
	notification.Template = notificationTemplate

	s.l.Debugf("Sending notification %v", notification)

	var options []nc.SendTaskOption

	now := time.Now()
	if now.After(subscriptionData.ScheduleTime) {
		delay := now.Sub(subscriptionData.ScheduleTime)
		s.l.Warningf("Sending notification after schedule for customer %s, dashboard %s, delay %v", subscriptionData.CustomerID, subscriptionData.Dashboard.Name, delay)
	} else {
		options = append(options, nc.WithScheduleTime(subscriptionData.ScheduleTime))
	}

	_, err := s.createSendTask(ctx, *notification, options...)
	if err != nil {
		return err
	}

	return nil
}

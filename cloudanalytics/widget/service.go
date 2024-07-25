//go:generate mockery --output=./mocks --all
package widget

import (
	"context"
	"fmt"
	"sync"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config"
	configsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config/dal"
	reportDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	reportIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats"
	reportStatsServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
)

//go:generate mockery --name ReportWidgetWriter --output=./mocks
type ReportWidgetWriter interface {
	Save(
		ctx context.Context,
		minUpdateDelayMinutes int,
		request domain.ReportWidgetRequest,
	) error
}

type WidgetService struct {
	loggerProvider       logger.Provider
	conn                 *connection.Connection
	cloudAnalytics       cloudanalytics.CloudAnalytics
	analyticsConfigs     analyticsConfigs
	reportDAL            reportIface.Reports
	publicDashboardsDAL  dal.PublicDashboards
	dashboardsDAL        dal.Dashboards
	userDAL              *userDal.UserFirestoreDAL
	reportStatsService   reportStatsServiceIface.ReportStatsService
	doitEmployeesService doitemployees.ServiceInterface
}

func NewWidgetService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
) (*WidgetService, error) {
	cfg := analyticsConfigs{
		&sync.Mutex{},
		configsDal.NewConfigsFirestoreWithClient(conn.Firestore),
		nil,
	}

	customerDal := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDal.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	reportStatsService, err := stats.NewReportStatsService(loggerProvider, reportDal)
	if err != nil {
		return nil, err
	}

	return &WidgetService{
		loggerProvider,
		conn,
		cloudAnalytics,
		cfg,
		reportDal,
		dal.NewPublicDashboardsFirestoreWithClient(conn.Firestore),
		dal.NewDashboardsFirestoreWithClient(conn.Firestore),
		userDal.NewUserFirestoreDALWithClient(conn.Firestore),
		reportStatsService,
		doitemployees.NewService(conn),
	}, nil
}

type ScheduledWidgetUpdateService struct {
	loggerProvider             logger.Provider
	widgetService              ReportWidgetWriter
	cloudTaskClient            iface.CloudTaskClient
	dashboardsDAL              dal.Dashboards
	publicDashboardsDAL        dal.PublicDashboards
	customersDAL               customerDal.Customers
	dashboardAccessMetadataDAL dal.DashboardAccessMetadata
	userDAL                    *userDal.UserFirestoreDAL
	reportStatsService         reportStatsServiceIface.ReportStatsService
	doitEmployeesService       doitemployees.ServiceInterface
}

func NewScheduledWidgetUpdateService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	widgetService *WidgetService,
	dashboardAccessMetadataDAL dal.DashboardAccessMetadata,
	reportStatsService reportStatsServiceIface.ReportStatsService,
) (*ScheduledWidgetUpdateService, error) {
	return &ScheduledWidgetUpdateService{
		loggerProvider,
		widgetService,
		conn.CloudTaskClient,
		dal.NewDashboardsFirestoreWithClient(conn.Firestore),
		dal.NewPublicDashboardsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		dashboardAccessMetadataDAL,
		userDal.NewUserFirestoreDALWithClient(conn.Firestore),
		reportStatsService,
		doitemployees.NewService(conn),
	}, nil
}

func NewScheduledWidgetUpdateServiceWithAll(
	loggingProvider logger.Provider,
	ws ReportWidgetWriter,
	cloudTaskClient iface.CloudTaskClient,
	dashboardsDAL dal.Dashboards,
	publicDashboardsDAL dal.PublicDashboards,
	customersDAL customerDal.Customers,
	dashboardAccessMetadataDAL dal.DashboardAccessMetadata,
	doitEmployeesService doitemployees.ServiceInterface,
) *ScheduledWidgetUpdateService {
	return &ScheduledWidgetUpdateService{
		loggerProvider:             loggingProvider,
		widgetService:              ws,
		cloudTaskClient:            cloudTaskClient,
		dashboardsDAL:              dashboardsDAL,
		publicDashboardsDAL:        publicDashboardsDAL,
		customersDAL:               customersDAL,
		dashboardAccessMetadataDAL: dashboardAccessMetadataDAL,
		doitEmployeesService:       doitEmployeesService,
	}
}

func NewWidgetServiceWithAll(loggingProvider logger.Provider, conn *connection.Connection) *WidgetService {
	return &WidgetService{loggerProvider: loggingProvider, conn: conn}
}

type analyticsConfigs struct {
	mutex           *sync.Mutex
	dal             configsDal.Configs
	extendedMetrics *[]config.ExtendedMetric
}

func (s *analyticsConfigs) initialize(ctx context.Context) error {
	if s.extendedMetrics == nil {
		s.mutex.Lock()
		defer s.mutex.Unlock()

		extendedMetrics, err := s.dal.GetExtendedMetrics(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch extended metrics config with error: %s", err)
		}

		s.extendedMetrics = &extendedMetrics
	}

	return nil
}

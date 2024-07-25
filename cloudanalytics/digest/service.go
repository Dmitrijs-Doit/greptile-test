package digest

import (
	"context"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/announcekit"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	highchartsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service/iface"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/dal"
	"github.com/doitintl/hello/scheduled-tasks/fixer/converter"
	"github.com/doitintl/hello/scheduled-tasks/fixer/converter/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	notificationCenter "github.com/doitintl/notificationcenter/pkg"
	notificationService "github.com/doitintl/notificationcenter/service"
)

type IDigestService interface {
	ScheduleDaily(ctx context.Context) error
	ScheduleWeekly(ctx context.Context) error
	Generate(ctx context.Context, dayParam int, req *GenerateTaskRequest) error
	Send(ctx context.Context, req *Data) error
	GetMonthlyDigest(ctx context.Context) error
}

type DigestService struct {
	loggerProvider    logger.Provider
	conn              *connection.Connection
	cloudAnalytics    cloudanalytics.CloudAnalytics
	dashboardDAL      dal.Dashboards
	IntegrationsDAL   doitFirestore.Integrations
	converter         iface.Converter
	widgetService     *widget.WidgetService
	announceKit       announcekit.AnnounceKit
	ncClient          notificationCenter.NotificationSender
	recipientsService notificationService.NotificationRecipientService
	highchartsService highchartsIface.IHighcharts
}

func NewDigestService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	highchartsService highchartsIface.IHighcharts,
) (*DigestService, error) {
	ctx := context.Background()

	widgetService, err := widget.NewWidgetService(logger.FromContext, conn)
	if err != nil {
		return nil, err
	}

	announceKitService, err := announcekit.NewAnnounceKitService(loggerProvider)
	if err != nil {
		return nil, err
	}

	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	ncClient, err := notificationCenter.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	recipientsService := notificationService.NewRecipientsService(conn.Firestore(ctx))

	return &DigestService{
		loggerProvider,
		conn,
		cloudAnalytics,
		dal.NewDashboardsFirestoreWithClient(conn.Firestore),
		doitFirestore.NewIntegrationsDALWithClient(conn.Firestore(ctx)),
		converter.NewCurrencyConverterService(),
		widgetService,
		announceKitService,
		ncClient,
		recipientsService,
		highchartsService,
	}, nil
}

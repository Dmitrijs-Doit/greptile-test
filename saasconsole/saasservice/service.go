package saasservice

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	metadataDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal"
	reportDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contract "github.com/doitintl/hello/scheduled-tasks/contract/service"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	dashboardDAL "github.com/doitintl/hello/scheduled-tasks/dashboard/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/billingpipeline"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts"
	sharedDal "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared/dal"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/saasservice/validator"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
	tiers "github.com/doitintl/tiers/service"
)

type SaaSConsoleService struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	serviceAccounts        *service_accounts.GCPSaaSConsoleServiceAccountsService
	billingImportStatus    sharedDal.BillingImportStatus
	saasConsoleDAL         fsdal.SaaSConsoleOnboard
	cloudConnectDAL        fsdal.CloudConnect
	customersDAL           customerDal.Customers
	metadataDAL            *metadataDAL.MetadataFirestore
	userDal                userDal.UserFirestoreDAL
	rolesDal               fsdal.Roles
	widgetService          *widget.ScheduledWidgetUpdateService
	tiersService           *tiers.TiersService
	notificationClient     notificationcenter.NotificationSender
	contractService        *contract.ContractService
	billingPipelineService billingpipeline.ServiceInterface
	validatorService       *validator.SaaSConsoleValidatorService
}

func NewSaaSConsoleService(log logger.Provider, conn *connection.Connection) (*SaaSConsoleService, error) {
	ctx := context.Background()

	notificationClient, err := notificationcenter.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	widgetService, err := widget.NewWidgetService(logger.FromContext, conn)
	if err != nil {
		return nil, err
	}

	dashboardAccessMetadataDAL := dashboardDAL.NewDashboardAccessMetadataFirestoreWithClient(conn.Firestore)

	reportDAL := reportDal.NewReportsFirestoreWithClient(conn.Firestore)

	reportStatsService, err := stats.NewReportStatsService(log, reportDAL)
	if err != nil {
		return nil, err
	}

	widgetUpdateService, err := widget.NewScheduledWidgetUpdateService(
		log,
		conn,
		widgetService,
		dashboardAccessMetadataDAL,
		reportStatsService,
	)
	if err != nil {
		return nil, err
	}

	validatorService, err := validator.NewSaaSConsoleValidatorService(log, conn)
	if err != nil {
		return nil, err
	}

	customersDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)

	cloudAnalyticsService, err := cloudanalytics.NewCloudAnalyticsService(log, conn, reportDAL, customersDAL)
	if err != nil {
		panic(err)
	}

	contractService := contract.NewContractService(log, conn, cloudAnalyticsService)

	return &SaaSConsoleService{
		log,
		conn,
		service_accounts.NewGCPSaaSConsoleServiceAccountsService(log, conn),
		sharedDal.NewBillingImportStatusWithClient(conn.Firestore),
		fsdal.NewSaaSConsoleOnboardDALWithClient(conn.Firestore(ctx)),
		fsdal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		customersDAL,
		metadataDAL.NewMetadataFirestoreWithClient(conn.Firestore),
		*userDal.NewUserFirestoreDALWithClient(conn.Firestore),
		fsdal.NewRolesDALWithClient(conn.Firestore(context.Background())),
		widgetUpdateService,
		tiers.NewTiersService(conn.Firestore),
		notificationClient,
		contractService,
		billingpipeline.NewService(log, conn),
		validatorService,
	}, nil
}

func (s *SaaSConsoleService) ValidateBillingData(ctx context.Context) error {
	return s.validatorService.ValidateBillingData(ctx)
}

func (s *SaaSConsoleService) ValidatePermissions(ctx context.Context) error {
	return s.validatorService.ValidatePermissions(ctx)
}

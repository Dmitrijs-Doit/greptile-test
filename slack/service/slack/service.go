package slack

import (
	"context"

	fsDal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	highchartsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slack/dal"
	dalIface "github.com/doitintl/hello/scheduled-tasks/slack/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/budgets"
	budgetsIface "github.com/doitintl/hello/scheduled-tasks/slack/service/budgets/iface"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/reports"
	reportsIface "github.com/doitintl/hello/scheduled-tasks/slack/service/reports/iface"
	"github.com/doitintl/slackapi"
)

// SlackService - service for DoiT International Slack app (AF79TTA7N)
type SlackService struct {
	loggerProvider logger.Provider
	*connection.Connection
	firestoreDAL    dalIface.IFirestoreDAL
	slackDAL        dalIface.ISlackDAL
	budgetsService  budgetsIface.IBudgetsService
	reportsService  reportsIface.IReportsService
	tenantService   *tenant.TenantService
	customerTypeDal fsDal.CustomerTypeIface
	doitsyBotToken  string
	API             *slackapi.SlackAPI
}

func NewSlackService(loggerProvider logger.Provider, conn *connection.Connection) (*SlackService, error) {
	ctx := context.Background()
	project := common.ProjectID

	doitsyBotToken, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSlackBot)
	if err != nil {
		return nil, err
	}

	tenantService, err := tenant.NewTenantsService(conn)
	if err != nil {
		return nil, err
	}

	budgetsClient, err := service.NewBudgetsService(loggerProvider, conn)
	if err != nil {
		return nil, err
	}

	highcharts, err := highchartsService.NewHighcharts(loggerProvider, conn, budgetsClient)
	if err != nil {
		return nil, err
	}

	budgetsService, err := budgets.NewBudgetsService(budgetsClient, highcharts)
	if err != nil {
		return nil, err
	}

	reportsService, err := reports.NewReportsService(loggerProvider, conn, highcharts)
	if err != nil {
		return nil, err
	}

	customerTypeDal := fsDal.NewCustomerTypeDALWithClient(conn.Firestore(ctx))

	slackAPI, err := slackapi.NewSlackAPI(ctx, project)
	if err != nil {
		return nil, err
	}

	firestoreDAL := dal.NewFirestoreDAL(ctx, conn)
	slackDAL := dal.NewSlackDALWithClient(slackAPI, firestoreDAL, loggerProvider)

	return &SlackService{
		loggerProvider,
		conn,
		firestoreDAL,
		slackDAL,
		budgetsService,
		reportsService,
		tenantService,
		customerTypeDal,
		string(doitsyBotToken),
		slackAPI,
	}, nil
}

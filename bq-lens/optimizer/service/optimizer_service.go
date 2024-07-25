package service

import (
	"context"
	"time"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal"
	bqDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery"
	optimizerDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	optimizerIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore/iface"
	dalIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/executor"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/iface"
	pricebook "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	cloudConnectServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudconnect/iface"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tiersService "github.com/doitintl/tiers/service"
)

type OptimizerService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	serviceBQ      iface.Bigquery
	dalFS          optimizerIface.Optimizer
	cloudConnect   cloudConnectServiceIface.CloudConnectService
	executor       iface.Executor
	reservations   iface.Reservations
	pricebook      pricebook.Pricebook
	insights       dalIface.Insights
	tiers          tiersService.TierServiceIface
	customerDAL    customerDAL.Customers
	timeNowFunc    func() time.Time
}

func NewOptimizer(
	ctx context.Context,
	log logger.Provider,
	conn *connection.Connection,
) *OptimizerService {
	cloudConnect := cloudconnect.NewCloudConnectService(log, conn)
	dalBQ := bqDal.NewBigquery(log, &doitBQ.QueryHandler{})
	dalFS := optimizerDal.NewDAL(conn.Firestore(context.Background()))
	reservations := NewReservations(log, dal.NewReservations(), dal.NewCloudResourceManager())
	exec := executor.NewExecutor(dalBQ)
	serviceBQ := NewBigQueryService(log, conn, dalBQ)
	pricebook := pricebook.NewPricebook(log, conn)
	tiers := tiersService.NewTiersService(conn.Firestore)
	customerDAL := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	insightsDAL, err := dal.NewInsights(ctx)
	if err != nil {
		panic(err)
	}

	return &OptimizerService{
		loggerProvider: log,
		conn:           conn,
		serviceBQ:      serviceBQ,
		cloudConnect:   cloudConnect,
		reservations:   reservations,
		pricebook:      pricebook,
		insights:       insightsDAL,
		tiers:          tiers,
		customerDAL:    customerDAL,
		executor:       exec,
		dalFS:          dalFS,
		timeNowFunc:    func() time.Time { return time.Now().UTC() },
	}
}

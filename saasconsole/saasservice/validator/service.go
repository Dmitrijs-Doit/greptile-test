package validator

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/billingpipeline"
	tiers "github.com/doitintl/tiers/service"
)

type SaaSConsoleValidatorService struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	saasConsoleDAL         fsdal.SaaSConsoleOnboard
	cloudConnectDAL        fsdal.CloudConnect
	customersDAL           customerDal.Customers
	billingPipelineService billingpipeline.ServiceInterface
	tiersService           *tiers.TiersService
}

func NewSaaSConsoleValidatorService(log logger.Provider, conn *connection.Connection) (*SaaSConsoleValidatorService, error) {
	ctx := context.Background()

	return &SaaSConsoleValidatorService{
		log,
		conn,
		fsdal.NewSaaSConsoleOnboardDALWithClient(conn.Firestore(ctx)),
		fsdal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		billingpipeline.NewService(log, conn),
		tiers.NewTiersService(conn.Firestore),
	}, nil
}

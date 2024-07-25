package onboarding

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/billingpipeline"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts"
	sharedDal "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared/dal"
)

type GCPSaaSConsoleOnboardService struct {
	loggerProvider         logger.Provider
	serviceAccounts        *service_accounts.GCPSaaSConsoleServiceAccountsService
	billingPipelineService billingpipeline.ServiceInterface
	billingImportStatus    sharedDal.BillingImportStatus
	saasConsoleDAL         fsdal.SaaSConsoleOnboard
	contractsDAL           fsdal.Contracts
	entitiesDAL            fsdal.Entities
	cloudConnectDAL        fsdal.CloudConnect
	accountManagersDAL     fsdal.AccountManagers
	assetsDAL              assetsDal.Assets
	customersDAL           customerDal.Customers
}

func NewGCPSaaSConsoleOnboardService(log logger.Provider, conn *connection.Connection) (*GCPSaaSConsoleOnboardService, error) {
	ctx := context.Background()

	return &GCPSaaSConsoleOnboardService{
		log,
		service_accounts.NewGCPSaaSConsoleServiceAccountsService(log, conn),
		billingpipeline.NewService(log, conn),
		sharedDal.NewBillingImportStatusWithClient(conn.Firestore),
		fsdal.NewSaaSConsoleOnboardDALWithClient(conn.Firestore(ctx)),
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
		fsdal.NewEntitiesDALWithClient(conn.Firestore(ctx)),
		fsdal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		fsdal.NewAccountManagersDALWithClient(conn.Firestore(ctx)),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}, nil
}

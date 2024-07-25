package onboarding

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	billing "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/application"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/test_connection"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/savingsreportfile"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts"
	sharedDal "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type GcpStandaloneOnboardingService struct {
	loggerProvider        logger.Provider
	serviceAccounts       *service_accounts.GcpStandaloneServiceAccountsService
	testBillingConnection *test_connection.TestConnection
	//billingPipelineService billingpipeline.ServiceInterface  // detailed billing pipeline service wrapper
	billing               *billing.Onboarding
	billingImportStatus   sharedDal.BillingImportStatus
	gcpService            *flexsaveresold.GCPService
	flexsaveStandaloneDAL fsdal.FlexsaveStandalone
	contractsDAL          fsdal.Contracts
	entitiesDAL           fsdal.Entities
	cloudConnectDAL       fsdal.CloudConnect
	accountManagersDAL    fsdal.AccountManagers
	assetsDAL             assetsDal.Assets
	customersDAL          customerDal.Customers
	savingsReportService  *savingsreportfile.Service
}

func NewGcpStandaloneOnboardingService(log logger.Provider, conn *connection.Connection) (*GcpStandaloneOnboardingService, error) {
	ctx := context.Background()

	return &GcpStandaloneOnboardingService{
		log,
		service_accounts.NewGcpStandaloneServiceAccountsService(log, conn),
		test_connection.NewTestConnection(log, conn),
		//billingpipeline.NewService(log, conn), // detailed billing pipeline service wrapper
		billing.NewOnboarding(log, conn),
		sharedDal.NewBillingImportStatusWithClient(conn.Firestore(ctx)),
		flexsaveresold.NewGCPService(log, conn),
		fsdal.NewFlexsaveStandaloneDALWithClient(conn.Firestore(ctx)),
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
		fsdal.NewEntitiesDALWithClient(conn.Firestore(ctx)),
		fsdal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		fsdal.NewAccountManagersDALWithClient(conn.Firestore(ctx)),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		savingsreportfile.NewService(conn),
	}, nil
}

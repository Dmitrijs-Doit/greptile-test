package aws

import (
	"context"
	"time"

	cloudTaskClientIface "github.com/doitintl/cloudtasks/iface"
	dal "github.com/doitintl/firestore"
	accessService "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access"
	accessIface "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access/iface"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AWSSaaSConsoleOnboardService struct {
	now                func() time.Time
	loggerProvider     logger.Provider
	conn               *connection.Connection
	saasConsoleDAL     dal.SaaSConsoleOnboard
	contractsDAL       dal.Contracts
	cloudConnectDAL    dal.CloudConnect
	entitiesDAL        dal.Entities
	integrationsDAL    dal.Integrations
	accountManagersDAL dal.AccountManagers
	assetsDAL          assetsDal.Assets
	customersDAL       customerDal.Customers
	flexsaveAPI        flexapi.FlexAPI
	awsAssetsService   assets.IAWSAssetsService
	awsAccessService   accessIface.Access
	cloudTaskClient    cloudTaskClientIface.CloudTaskClient
}

func NewAWSSaaSConsoleOnboardService(
	log logger.Provider,
	conn *connection.Connection,
	flexAPIService flexapi.FlexAPI,
	awsAssetsService assets.IAWSAssetsService,
) (*AWSSaaSConsoleOnboardService, error) {
	ctx := context.Background()

	now := func() time.Time {
		return time.Now().UTC()
	}

	return &AWSSaaSConsoleOnboardService{
		now,
		log,
		conn,
		dal.NewSaaSConsoleOnboardDALWithClient(conn.Firestore(ctx)),
		dal.NewContractsDALWithClient(conn.Firestore(ctx)),
		dal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		dal.NewEntitiesDALWithClient(conn.Firestore(ctx)),
		dal.NewIntegrationsDALWithClient(conn.Firestore(ctx)),
		dal.NewAccountManagersDALWithClient(conn.Firestore(ctx)),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		flexAPIService,
		awsAssetsService,
		accessService.New(),
		conn.CloudTaskClient,
	}, nil
}

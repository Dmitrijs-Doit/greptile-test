package aws

import (
	"context"
	"time"

	bq "cloud.google.com/go/bigquery"

	"github.com/doitintl/bigquery"
	bigqueryIface "github.com/doitintl/bigquery/iface"
	cloudTaskClientIface "github.com/doitintl/cloudtasks/iface"
	fsdal "github.com/doitintl/firestore"
	accessService "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access"
	accessIface "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/access/iface"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	flexsave "github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws/savingsreportfile"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AwsStandaloneService struct {
	now                   func() time.Time
	loggerProvider        logger.Provider
	conn                  *connection.Connection
	flexsaveStandaloneDAL fsdal.FlexsaveStandalone
	contractsDAL          fsdal.Contracts
	cloudConnectDAL       fsdal.CloudConnect
	entitiesDAL           fsdal.Entities
	integrationsDAL       fsdal.Integrations
	accountManagersDAL    fsdal.AccountManagers
	assetsDAL             assetsDal.Assets
	customersDAL          customerDal.Customers
	queryHandler          bigqueryIface.QueryHandler
	bigQueryClient        *bq.Client
	bqmh                  bigqueryIface.BigqueryManagerHandler
	AWSAccess             iface.AWSAccess
	flexRIService         *flexsave.Service
	savingsReportService  *savingsreportfile.Service
	flexAPI               flexapi.FlexAPI
	payers                payers.Service
	awsAssetsService      assets.IAWSAssetsService
	awsAccessService      accessIface.Access
	cloudTaskClient       cloudTaskClientIface.CloudTaskClient
}

func NewAwsStandaloneService(log logger.Provider, conn *connection.Connection) (*AwsStandaloneService, error) {
	ctx := context.Background()

	bqClient, err := bq.NewClient(context.Background(), getEnvironment())
	if err != nil {
		return nil, err
	}

	now := func() time.Time {
		return time.Now().UTC()
	}

	flexAPI, err := flexapi.NewFlexAPIService()
	if err != nil {
		panic(err)
	}

	payers, err := payers.NewService()
	if err != nil {
		panic(err)
	}

	awsAssetsService, err := assets.NewAWSAssetsService(log, conn, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	return &AwsStandaloneService{
		now,
		log,
		conn,
		fsdal.NewFlexsaveStandaloneDALWithClient(conn.Firestore(ctx)),
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
		fsdal.NewCloudConnectDALWithClient(conn.Firestore(ctx)),
		fsdal.NewEntitiesDALWithClient(conn.Firestore(ctx)),
		fsdal.NewIntegrationsDALWithClient(conn.Firestore(ctx)),
		fsdal.NewAccountManagersDALWithClient(conn.Firestore(ctx)),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		bigquery.QueryHandler{},
		bqClient,
		bigquery.BigqueryManagerHandler{},
		&AWSAccess{},
		flexsave.NewService(log, conn),
		savingsreportfile.NewService(conn),
		flexAPI,
		payers,
		awsAssetsService,
		accessService.New(),
		conn.CloudTaskClient,
	}, nil
}

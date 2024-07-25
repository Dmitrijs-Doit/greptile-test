package azure

import (
	"context"

	"cloud.google.com/go/bigquery"

	cloudTaskClientIface "github.com/doitintl/cloudtasks/iface"
	dal "github.com/doitintl/firestore"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	common "github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AzureSaaSConsoleService struct {
	loggerProvider   logger.Provider
	conn             *connection.Connection
	cloudTaskClient  cloudTaskClientIface.CloudTaskClient
	logger           *logger.Logging
	azureSaasDal     dal.IAzureFirestore
	bq               *bigquery.Client
	assetsDAL        assetsDal.Assets
	customersDAL     customerDal.Customers
	assetSettingsDAL assetsDal.AssetSettings
}

func NewAzureSaaSConsoleService(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) (*AzureSaaSConsoleService, error) {
	ctx := context.Background()

	bigquery, err := bigquery.NewClient(context.Background(), common.ProjectID)
	if err != nil {
		panic(err)
	}

	return &AzureSaaSConsoleService{
		log,
		conn,
		conn.CloudTaskClient,
		oldLog,
		dal.NewAzureFirestoreWithClient(conn.Firestore(ctx)),
		bigquery,
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		assetsDal.NewAssetSettingsFirestoreWithClient(conn.Firestore),
	}, nil
}

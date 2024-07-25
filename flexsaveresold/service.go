package flexsaveresold

import (
	"context"

	"cloud.google.com/go/bigquery"

	sharedbq "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"

	fsdal "github.com/doitintl/firestore"
	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	asset "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	flexsaveHistoryMonthAmount int = 13
	flexRIDetailPrefix             = "Flexible"
	flexSaveDetailPrefix           = "Flexsave"
)

type Service struct {
	Logger logger.Provider
	*connection.Connection

	flexsaveGCPUsageTable string
	flexsaveGlobalTable   string
	flexAPI               flexapi.FlexAPI
	assets                asset.Assets
	integrationsDAL       fsdal.Integrations
	customersDAL          customersDal.Customers
	mpaDAL                mpaDAL.MasterPayerAccounts
	bigqueryClient        *bigquery.Client
	queryHandler          iface.QueryHandler
	jobHandler            iface.JobHandler
}

func NewService(loggerProvider logger.Provider, conn *connection.Connection) *Service {
	flexsaveGCPUsageTable := devFlexsaveGCPUsageTable
	flexsaveGlobalTable := devFlexsaveGlobalTable

	globalProject := devProjectID

	if common.Production {
		flexsaveGCPUsageTable = prodFlexsaveGCPUsageTable
		flexsaveGlobalTable = prodFlexsaveGlobalTable
		globalProject = prodProjectID
	}

	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		panic(err)
	}

	assetsDAL := asset.NewAssetsFirestoreWithClient(conn.Firestore)

	integrationsDAL := fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background()))

	customersDAL := customersDal.NewCustomersFirestoreWithClient(conn.Firestore)

	mpaDAL := mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background()))

	bigquery, err := bigquery.NewClient(context.Background(), globalProject)
	if err != nil {
		panic(err)
	}

	return &Service{
		loggerProvider,
		conn,
		flexsaveGCPUsageTable,
		flexsaveGlobalTable,
		flexAPIService,
		assetsDAL,
		integrationsDAL,
		customersDAL,
		mpaDAL,
		bigquery,
		sharedbq.QueryHandler{},
		sharedbq.JobHandler{},
	}
}

type ServiceError struct {
	UserMsg string
	Err     error
}

func NewServiceError(msg string, err error) error {
	return &ServiceError{
		msg,
		err,
	}
}
func (e *ServiceError) Error() string { return e.UserMsg }
func (e *ServiceError) Unwrap() error { return e.Err }

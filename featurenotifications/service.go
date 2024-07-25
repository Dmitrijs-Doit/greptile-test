package featurenotifications

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/fixer/converter"
	"github.com/doitintl/hello/scheduled-tasks/fixer/converter/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FeatureNotificationService struct {
	logger logger.Provider
	*connection.Connection
	FlexSave        *flexsaveresold.Service
	integrationsDAL fsdal.Integrations
	usersDAL        fsdal.Users
	rolesDal        fsdal.Roles
	permissionsDal  fsdal.Permissions
	customersDAL    fsdal.Customer
	currencyService iface.Converter
}

func NewFeatureNotificationService(log logger.Provider, conn *connection.Connection) *FeatureNotificationService {
	flexSaveService := flexsaveresold.NewService(log, conn)

	return &FeatureNotificationService{
		log,
		conn,
		flexSaveService,
		fsdal.NewIntegrationsDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewUsersDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewRolesDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewPermissionsDALWithClient(conn.Firestore(context.Background())),
		fsdal.NewCustomersDALWithClient(conn.Firestore(context.Background())),
		converter.NewCurrencyConverterService(),
	}
}

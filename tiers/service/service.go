package service

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	alertsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	budgetsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/tiers/dal"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	userDalIface "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
	tiersService "github.com/doitintl/tiers/service"
)

var userRoleFilter = []common.PresetRole{common.PresetRoleAdmin}

type TiersService struct {
	loggerProvider logger.Provider
	*connection.Connection
	customerDal           customerDal.Customers
	usersDal              userDalIface.IUserFirestoreDAL
	trialNotificationsDal dal.TrialNotificationsDAL
	tiersSvc              tiersService.TierServiceIface
	notificationClient    notificationcenter.NotificationSender
	attributionsDal       attributionsDal.AttributionsFirestore
	contractsDal          fsdal.Contracts
	assetsDal             assetsDal.AssetsFirestore
	alertsDal             alertsDal.AlertsFirestore
	budgetsDal            budgetsDal.BudgetsFirestore
}

func NewTiersService(log logger.Provider, conn *connection.Connection) (*TiersService, error) {
	ctx := context.Background()

	notificationClient, err := notificationcenter.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	return &TiersService{
		log,
		conn,
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		userDal.NewUserFirestoreDALWithClient(conn.Firestore),
		*dal.NewTrialNotificationsDALClient(conn.Firestore),
		tiersService.NewTiersService(conn.Firestore),
		notificationClient,
		*attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore),
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
		*assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		*alertsDal.NewAlertsFirestoreWithClient(conn.Firestore),
		*budgetsDal.NewBudgetsFirestoreWithClient(conn.Firestore),
	}, nil
}

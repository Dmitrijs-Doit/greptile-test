package service

import (
	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	costAllocationDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/dal"
	costAllocationDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/dal/iface"
	customersDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CostAllocationService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	dal            costAllocationDalIface.CostAllocations
	assetsDal      assetsDal.Assets
	customersDal   customersDal.Customers
}

func NewCostAllocationService(log logger.Provider, conn *connection.Connection) (*CostAllocationService, error) {
	return &CostAllocationService{
		log,
		conn,
		costAllocationDal.NewCostAllocationsFirestoreWithClient(conn.Firestore),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		customersDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}, nil
}

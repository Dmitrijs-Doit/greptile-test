package service

import (
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/dal/iface"
	cloudConnectServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudconnect/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type DiscoveryService struct {
	loggerProvider    logger.Provider
	conn              *connection.Connection
	discoveryBigquery iface.Bigquery
	cloudConnect      cloudConnectServiceIface.CloudConnectService
}

func NewDiscovery(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	discoveryBigquery iface.Bigquery,
	cloudConnect cloudConnectServiceIface.CloudConnectService,
) *DiscoveryService {
	return &DiscoveryService{
		loggerProvider,
		conn,
		discoveryBigquery,
		cloudConnect,
	}
}

package service

import (
	backfillDALIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BackfillService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	backfillDAL    backfillDALIface.IBackfillFirestore
}

func NewBackfillService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	backfillDAL backfillDALIface.IBackfillFirestore) *BackfillService {
	return &BackfillService{
		loggerProvider: loggerProvider,
		conn:           conn,
		backfillDAL:    backfillDAL,
	}
}

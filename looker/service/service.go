package service

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AssetsService struct {
	*logger.Logging
	conn                   *connection.Connection
	contractsDAL           fsdal.Contracts
	bigQueryFromContextFun connection.BigQueryFromContextFun
}

func NewAssetsService(log *logger.Logging, conn *connection.Connection) *AssetsService {
	ctx := context.Background()

	return &AssetsService{
		log,
		conn,
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
		conn.Bigquery,
	}
}

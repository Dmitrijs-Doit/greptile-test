package service

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type InvoicingService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	contractsDAL   fsdal.Contracts
}

func NewInvoicingService(loggerProvider logger.Provider, conn *connection.Connection) (*InvoicingService, error) {
	ctx := context.Background()

	return &InvoicingService{
		loggerProvider,
		conn,
		fsdal.NewContractsDALWithClient(conn.Firestore(ctx)),
	}, nil
}

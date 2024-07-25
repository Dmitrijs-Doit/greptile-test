package knownissues

import (
	"context"

	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Service struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	mpaDAL         mpaDAL.MasterPayerAccounts
}

func NewService(loggerProvider logger.Provider, conn *connection.Connection) *Service {
	return &Service{
		loggerProvider,
		conn,
		mpaDAL.NewMasterPayerAccountDALWithClient(conn.Firestore(context.Background())),
	}
}

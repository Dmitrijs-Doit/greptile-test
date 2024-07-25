package generatedaccounts

import (
	"context"

	generatedAccountsDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/generatedaccounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Service struct {
	loggerProvider logger.Provider
	*connection.Connection
	awsAccountsDAL generatedAccountsDAL.AwsAccounts
}

func NewGeneratedAccountsService(log logger.Provider, conn *connection.Connection) (IGeneratedAccountsService, error) {
	ctx := context.Background()

	return &Service{
		log,
		conn,
		generatedAccountsDAL.NewAwsAccountsDAL(conn.Firestore(ctx)),
	}, nil
}

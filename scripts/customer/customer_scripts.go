package customer

import (
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Scripts struct {
	conn           *connection.Connection
	loggerProvider logger.Provider
}

func NewCustomerScripts(loggerProvider logger.Provider, conn *connection.Connection) *Scripts {
	return &Scripts{
		conn,
		loggerProvider,
	}
}

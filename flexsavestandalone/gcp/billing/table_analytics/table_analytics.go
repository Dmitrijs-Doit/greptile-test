package tableanalytics

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TableAnalytics struct {
	loggerProvider logger.Provider
	*connection.Connection
	tableQuery service.TableQuery
	table      service.Table
}

func NewTableAnalytics(log logger.Provider, conn *connection.Connection) *TableAnalytics {
	return &TableAnalytics{
		loggerProvider: log,
		Connection:     conn,
		tableQuery:     service.NewTableQuery(log, conn),
		table:          service.NewTable(log, conn),
	}
}

func (ta *TableAnalytics) RunDetailedTableRewritesMapping(ctx context.Context) error {
	return ta.tableQuery.RunDetailedTableRewritesMapping(ctx)
}

func (ta *TableAnalytics) RunDataFreshnessReport(ctx context.Context) error {
	return ta.tableQuery.RunDataFreshnessReport(ctx)
}

package rows_validator

import (
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type RowsValidator struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	customerBQClient service.ExternalBigQueryClient
	tableQuery       service.TableQuery
	bqUtils          *bq_utils.BQ_Utils
	notification     *service.Notification
}

func NewRowsValidator(log logger.Provider, conn *connection.Connection) *RowsValidator {
	return &RowsValidator{
		log,
		conn,
		service.NewMetadata(log, conn),
		service.NewExternalBigQueryClient(log, conn),
		service.NewTableQuery(log, conn),
		bq_utils.NewBQ_UTils(log, conn),
		service.NewNotification(log),
	}
}

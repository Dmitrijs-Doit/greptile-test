package tablemanagement

import (
	discountsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/service"
	discountsServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/service/iface"
	reportStatus "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/statuses"
	customer "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BillingTableManagementService struct {
	loggerProvider      logger.Provider
	conn                *connection.Connection
	reportStatusService *reportStatus.ReportStatusesService
	discountsService    discountsServiceIface.IDiscountsService
	customerDAL         customer.Customers
}

func NewBillingTableManagementService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	customerDAL customer.Customers,
) (*BillingTableManagementService, error) {
	reportStatusService, err := reportStatus.NewReportStatusesService(loggerProvider, conn)
	if err != nil {
		return nil, err
	}

	return &BillingTableManagementService{
		loggerProvider,
		conn,
		reportStatusService,
		discountsService.NewDiscountsService(conn),
		customerDAL,
	}, nil
}

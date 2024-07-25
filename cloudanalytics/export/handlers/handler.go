package handlers

import (
	"net/http"

	domainExport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/domain"
	exportService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/service"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
)

type BillingExport struct {
	loggerProvider logger.Provider
	service        serviceIface.IExportService
}

func NewBillingExportHandler(log logger.Provider, conn *connection.Connection) *BillingExport {
	s := exportService.NewBillingExportService(log, conn)

	return &BillingExport{
		log,
		s,
	}
}

func (h *BillingExport) HandleCustomerBillingExport(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var taskBody domainExport.BillingExportInputStruct

	if err := ctx.ShouldBindJSON(&taskBody); err != nil {
		l.Errorf("Billing data export failed while parsing request body.\n Error: %v", err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	validate := validator.New()

	if err := validate.Struct(taskBody); err != nil {
		return web.Respond(ctx, "Missing required fields", http.StatusBadRequest)
	}

	customerID := ctx.Param("customerID")

	err := h.service.ExportBillingData(ctx, customerID, &taskBody)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

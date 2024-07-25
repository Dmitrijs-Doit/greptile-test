package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	backfillDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/dal"
	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	backfillService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/service"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Backfill struct {
	loggerProvider logger.Provider
	service        serviceIface.IBackfillService
}

func NewBackfill(log logger.Provider, conn *connection.Connection) *Backfill {
	backfillDAL := backfillDAL.NewBackfillFirestoreWithClient(conn.Firestore)
	backfillService := backfillService.NewBackfillService(log, conn, backfillDAL)

	return &Backfill{
		loggerProvider: log,
		service:        backfillService,
	}
}

func (h *Backfill) HandleCustomer(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var taskBody domainBackfill.TaskBodyHandlerCustomer

	if err := ctx.ShouldBindJSON(&taskBody); err != nil {
		l.Errorf("GCP billing data backfill failed while parsing request body.\n Error: %v", err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	customerID := ctx.Param("customerID")

	err := h.service.BackfillCustomer(ctx, customerID, &taskBody)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

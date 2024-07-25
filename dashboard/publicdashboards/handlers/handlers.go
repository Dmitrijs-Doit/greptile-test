package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/dashboard/publicdashboards"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Dashboard struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	publicDashboardService *publicdashboards.PublicDashboardService
}

func NewDashboard(loggerProvider logger.Provider, conn *connection.Connection) *Dashboard {
	service := publicdashboards.NewPublicDashboardService(loggerProvider, conn)

	return &Dashboard{
		loggerProvider,
		conn,
		service,
	}
}

func (h *Dashboard) AttachAllDashboardsHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.publicDashboardService.AttachAllDashboards(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, 200)
}

package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/domain"
	service "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/service"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/service/iface"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type DashboardSubscription struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        serviceIface.IService
}

func NewDashboardSubscription(loggerProvider logger.Provider, conn *connection.Connection) *DashboardSubscription {
	service := service.NewService(conn, loggerProvider)

	return &DashboardSubscription{
		loggerProvider,
		conn,
		service,
	}
}

func (h *DashboardSubscription) SendSubscription(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginWidgets)

	var request domain.HandleReportSubscriptionRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if request.CustomerID == "" || request.ConfigID == "" {
		return web.NewRequestError(fmt.Errorf("customerID and configID are required"), http.StatusBadRequest)
	}

	if err := h.service.HandleDashboardSubscription(ctx, request); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

package handlers

import (
	"net/http"
	"strconv"

	"github.com/doitintl/hello/scheduled-tasks/entity/service"
	"github.com/doitintl/hello/scheduled-tasks/entity/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type Entities struct {
	loggerProvider logger.Provider
	service        iface.EntitiesIface
}

func NewEntities(log logger.Provider, conn *connection.Connection) *Entities {
	s := service.NewEntitiesService(log, conn)

	return &Entities{
		log,
		s,
	}
}

func (h *Entities) SyncEntitiesInvoiceAttributions(ctx *gin.Context) error {
	forceUpdate, _ := strconv.ParseBool(ctx.Query("forceUpdate"))
	if err := h.service.SyncEntitiesInvoiceAttributions(ctx, forceUpdate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

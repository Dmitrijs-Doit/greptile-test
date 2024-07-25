package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/looker/domain"
	lookerAssetsService "github.com/doitintl/hello/scheduled-tasks/looker/service"
	lookerAssetsServiceIface "github.com/doitintl/hello/scheduled-tasks/looker/service/iface"
)

type LookerAssetsHandler struct {
	*logger.Logging
	service lookerAssetsServiceIface.AssetsServiceIface
}

func (h LookerAssetsHandler) UpdateCustomersLookerTable(ctx *gin.Context) error {
	var request domain.UpdateTableInterval
	if err := ctx.ShouldBindJSON(&request); err != nil && err.Error() != "EOF" {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.LoadLookerContractsToBQ(ctx, request)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func NewLookerAssetsHandler(log *logger.Logging, conn *connection.Connection) *LookerAssetsHandler {
	return &LookerAssetsHandler{
		log,
		lookerAssetsService.NewAssetsService(log, conn),
	}
}

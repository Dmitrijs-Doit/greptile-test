package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/spot0/fbod"
)

type SpotZeroFbod struct {
	loggerProvider logger.Provider
	service        *fbod.SpotZeroFbodService
}

func NewSpotZeroFbod(loggerProvider logger.Provider, conn *connection.Connection) *SpotZeroFbod {
	service := fbod.NewSpotScalingFbodService(loggerProvider, conn)

	return &SpotZeroFbod{
		loggerProvider,
		service,
	}
}

func (h *SpotZeroFbod) FbodHealthCheck(ctx *gin.Context) error {
	if err := h.service.FbodHealthCheck(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

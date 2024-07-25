package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	monitoring "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FlexsaveMonitoring interface {
	DetectSharedPayerSavingsDiscrepancies(ctx *gin.Context) error
}

type flexsaveMonitoring struct {
	loggerProvider logger.Provider
	service        monitoring.Service
}

func NewFlexsaveMonitoring(log logger.Provider, conn *connection.Connection) FlexsaveMonitoring {
	return &flexsaveMonitoring{
		log,
		monitoring.NewService(log, conn),
	}
}

func (h *flexsaveMonitoring) DetectSharedPayerSavingsDiscrepancies(ctx *gin.Context) error {
	err := h.service.DetectSharedPayerSavingsDiscrepancies(ctx, time.Now().UTC())
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

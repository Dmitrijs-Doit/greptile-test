package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	pkg "github.com/doitintl/hello/scheduled-tasks/azure/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	azuresaasconsole "github.com/doitintl/hello/scheduled-tasks/saasconsole/azure"
)

const (
	invalidPayload = "invalid payload"
)

type AzureSaaSConsoleHandler struct {
	loggerProvider         logger.Provider
	azureStandaloneService *azuresaasconsole.AzureSaaSConsoleService
	logger                 *logger.Logging
}

func NewAzureSaaSConsoleHandler(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *AzureSaaSConsoleHandler {
	azureStandaloneService, err := azuresaasconsole.NewAzureSaaSConsoleService(log, conn, oldLog)
	if err != nil {
		panic(err)
	}

	return &AzureSaaSConsoleHandler{
		log,
		azureStandaloneService,
		oldLog,
	}
}

// create asset discovery tasks handler
func (h *AzureSaaSConsoleHandler) CreateAssetDiscoveryTasks(ctx *gin.Context) error {
	if err := h.azureStandaloneService.CreateAssetDiscoveryTasks(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// RunAssetDiscoveryTaskHandler runs asset discovery task handler
func (h *AzureSaaSConsoleHandler) RunAssetDiscoveryTaskHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var body pkg.BillingDataConfig

	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(errors.New(invalidPayload), http.StatusBadRequest)
	}

	if err := h.azureStandaloneService.RunAssetDiscoveryTask(ctx, customerID, body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

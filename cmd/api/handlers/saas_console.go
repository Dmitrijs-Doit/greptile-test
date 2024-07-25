package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/saasservice"
)

type SaaSConsoleHandler struct {
	loggerProvider logger.Provider
	service        *saasservice.SaaSConsoleService
}

func NewSaaSConsoleHandler(log logger.Provider, conn *connection.Connection) *SaaSConsoleHandler {
	service, err := saasservice.NewSaaSConsoleService(log, conn)
	if err != nil {
		panic(err)
	}

	return &SaaSConsoleHandler{
		log,
		service,
	}
}

func (h *SaaSConsoleHandler) NotifyNewAccounts(ctx *gin.Context) error {
	dryRun := ctx.Query("dryRun") == "true"

	if err := h.service.NotifyCloudAnalyticsBillingDataReady(ctx, dryRun); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *SaaSConsoleHandler) ValidateBillingData(ctx *gin.Context) error {
	if err := h.service.ValidateBillingData(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *SaaSConsoleHandler) ValidatePermissions(ctx *gin.Context) error {
	if err := h.service.ValidatePermissions(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *SaaSConsoleHandler) DeactivateNoActiveTierBilling(ctx *gin.Context) error {
	if err := h.service.DeactivateNoActiveTierBillingImport(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

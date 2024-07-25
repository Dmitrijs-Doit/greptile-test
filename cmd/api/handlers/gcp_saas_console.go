package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	gcpsaasconsole "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/onboarding"
)

type GCPSaaSConsoleHandler struct {
	loggerProvider    logger.Provider
	onboardingService *gcpsaasconsole.GCPSaaSConsoleOnboardService
}

func NewGCPSaaSConsoleHandler(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *GCPSaaSConsoleHandler {
	onboardingService, err := gcpsaasconsole.NewGCPSaaSConsoleOnboardService(log, conn)
	if err != nil {
		panic(err)
	}

	return &GCPSaaSConsoleHandler{
		log,
		onboardingService,
	}
}

func (h *GCPSaaSConsoleHandler) InitOnboarding(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	res := h.onboardingService.InitOnboarding(ctx, customerID)

	return web.Respond(ctx, res, http.StatusOK)
}

// AddContract creates contract for a given customer
func (h *GCPSaaSConsoleHandler) AddContract(ctx *gin.Context) error {
	req, res := h.onboardingService.ParseContractRequest(ctx)
	if res.Success {
		res = h.onboardingService.AddContract(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *GCPSaaSConsoleHandler) Activate(ctx *gin.Context) error {
	req, res := h.onboardingService.ParseRequest(ctx)
	if res.Success {
		res = h.onboardingService.Activate(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *GCPSaaSConsoleHandler) RunCreateServiceAccountsTask(ctx *gin.Context) error {
	if err := h.onboardingService.CreateServiceAccounts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *GCPSaaSConsoleHandler) InitEnvironmentTask(ctx *gin.Context) error {
	if err := h.onboardingService.InitEnvironment(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

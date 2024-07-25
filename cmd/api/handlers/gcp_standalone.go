package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/onboarding"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/savingsreportfile"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type GcpStandaloneHandler struct {
	loggerProvider    logger.Provider
	onboardingService *onboarding.GcpStandaloneOnboardingService
}

func NewGcpStandaloneHandler(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *GcpStandaloneHandler {
	onboardingService, err := onboarding.NewGcpStandaloneOnboardingService(log, conn)
	if err != nil {
		panic(err)
	}

	return &GcpStandaloneHandler{
		log,
		onboardingService,
	}
}

func (h *GcpStandaloneHandler) InitOnboarding(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	res := h.onboardingService.InitOnboarding(ctx, customerID)

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *GcpStandaloneHandler) TestEstimationsConnection(ctx *gin.Context) error {
	req, res := h.onboardingService.ParseRequest(ctx, false)
	if res.Success {
		res = h.onboardingService.TestEstimationsConnection(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *GcpStandaloneHandler) RefreshEstimations(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	res := h.onboardingService.RefreshEstimationsWrapper(ctx, customerID)

	return web.Respond(ctx, res, http.StatusOK)
}

// AddContract creates contract for a given customer
func (h *GcpStandaloneHandler) AddContract(ctx *gin.Context) error {
	req, res := h.onboardingService.ParseContractRequest(ctx)
	if res.Success {
		res = h.onboardingService.AddContract(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *GcpStandaloneHandler) Activate(ctx *gin.Context) error {
	req, res := h.onboardingService.ParseRequest(ctx, true)
	if res.Success {
		res = h.onboardingService.Activate(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *GcpStandaloneHandler) RunCreateServiceAccountsTask(ctx *gin.Context) error {
	if err := h.onboardingService.CreateServiceAccounts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *GcpStandaloneHandler) InitEnvironmentTask(ctx *gin.Context) error {
	if err := h.onboardingService.InitEnvironment(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *GcpStandaloneHandler) SavingsReport(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var savingsReport savingsreportfile.StandaloneSavingsReport
	if err := ctx.ShouldBindQuery(&savingsReport); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	pdfByte, err := h.onboardingService.GetSavingsFileReport(ctx, customerID, savingsReport)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.RespondDownloadFile(ctx, pdfByte, "gcp-standalone-savings-report.pdf")
}

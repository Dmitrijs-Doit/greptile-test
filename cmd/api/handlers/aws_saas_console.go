package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	awssaasconsole "github.com/doitintl/hello/scheduled-tasks/saasconsole/aws"
)

type AWSSaaSConsoleHandler struct {
	loggerProvider       logger.Provider
	awsStandaloneService *awssaasconsole.AWSSaaSConsoleOnboardService
	logger               *logger.Logging
}

func NewAWSSaaSConsoleHandler(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *AWSSaaSConsoleHandler {
	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		panic(err)
	}

	awsAssetsService, err := assets.NewAWSAssetsService(log, conn, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	awsStandaloneService, err := awssaasconsole.NewAWSSaaSConsoleOnboardService(log, conn, flexAPIService, awsAssetsService)
	if err != nil {
		panic(err)
	}

	return &AWSSaaSConsoleHandler{
		log,
		awsStandaloneService,
		oldLog,
	}
}

// InitOnboarding creates onboarding document for the 1st time
func (h *AWSSaaSConsoleHandler) InitOnboarding(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	req, res := h.awsStandaloneService.ParseInitRequest(ctx, customerID)
	if res.Success {
		res = h.awsStandaloneService.InitOnboarding(ctx, customerID, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

// AddContract creates contract for a given customer
func (h *AWSSaaSConsoleHandler) AddContract(ctx *gin.Context) error {
	req, res := h.awsStandaloneService.ParseContractRequest(ctx)

	if res.Success {
		res = h.awsStandaloneService.AddContract(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

// UpdateBilling updates billing for the customer, final onboarding step
func (h *AWSSaaSConsoleHandler) CURDiscovery(ctx *gin.Context) error {
	req, res := h.awsStandaloneService.ParseCURDiscoveryRequest(ctx, false)

	if res.Success {
		res = h.awsStandaloneService.CURDiscovery(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

// UpdateBilling updates billing for the customer, final onboarding step
func (h *AWSSaaSConsoleHandler) CURRefresh(ctx *gin.Context) error {
	req, res := h.awsStandaloneService.ParseCURRefreshRequest(ctx)

	if res.Success {
		res = h.awsStandaloneService.CURRefresh(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

// UpdateBilling updates billing for the customer, final onboarding step
func (h *AWSSaaSConsoleHandler) Activate(ctx *gin.Context) error {
	req, res := h.awsStandaloneService.ParseActivateRequest(ctx)

	if res.Success {
		res = h.awsStandaloneService.UpdateBilling(ctx, req)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

// StackDeletion add log pointing for stack deletion initiated by AWS console
func (h *AWSSaaSConsoleHandler) StackDeletion(ctx *gin.Context) error {
	req, res := h.awsStandaloneService.ParseCURDiscoveryRequest(ctx, true)
	if res.Success {
		res = h.awsStandaloneService.StackDeletion(ctx, req.CustomerID, req.AccountID)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *AWSSaaSConsoleHandler) UpdateAllSaaSAssets(ctx *gin.Context) error {
	if err := h.awsStandaloneService.UpdateAllSaaSAssets(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWSSaaSConsoleHandler) UpdateSaaSAssets(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var req awssaasconsole.UpdateSaaSAssetsRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.awsStandaloneService.UpdateSaaSAssets(ctx, customerID, req.Accounts); err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

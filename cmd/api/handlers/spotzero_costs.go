package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/spot0/costs"
)

type SpotZeroCosts struct {
	loggerProvider logger.Provider
	service        *costs.SpotZeroCostsService
	ceService      *costs.SpotZeroCostsExplorerService
}

func NewSpotZeroCosts(loggerProvider logger.Provider, conn *connection.Connection) *SpotZeroCosts {
	service := costs.NewSpotScalingCostsService(loggerProvider, conn)
	ceService := costs.NewSpotZeroCostsExplorerService(loggerProvider, conn)

	return &SpotZeroCosts{
		loggerProvider,
		service,
		ceService,
	}
}

type DailyCostsRequest struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Year      string `json:"billing_year"`
	Month     string `json:"billing_month"`
	AccountID string `json:"account_id"`
}

type MonthCostsRequest struct {
	Year      string `json:"billing_year"`
	Month     string `json:"billing_month"`
	AccountID string `json:"account_id"`
}

func (h *SpotZeroCosts) SpotScalingDailyCosts(ctx *gin.Context) error {
	var req DailyCostsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return err
	}

	return h.service.SpotScalingDailyCosts(ctx, req.StartDate, req.EndDate, req.Year, req.Month, req.AccountID)
}

func (h *SpotZeroCosts) SpotScalingMonthlyCosts(ctx *gin.Context) error {
	var req MonthCostsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return err
	}

	return h.service.SpotScalingMonthlyCosts(ctx, req.Year, req.Month, req.AccountID)
}

func (h *SpotZeroCosts) UpdateCostAllocationTags(ctx *gin.Context) error {
	err := h.ceService.UpdateCostAllocationTags(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

type ASGCustomerEmailHandler struct {
	loggerProvider logger.Provider
	service        *costs.ASGCustomerEmail
}

type SendMarketingEmailRequest struct {
	MaxCustomersPerTask int `json:"max_customers_per_task"`
	MinDaysOnboarded    int `json:"min_days_onboarded"`
}

func NewASGCustomerEmailHandler(loggerProvider logger.Provider, conn *connection.Connection) *ASGCustomerEmailHandler {
	service := costs.NewASGCustomerEmail(loggerProvider, conn)

	return &ASGCustomerEmailHandler{
		loggerProvider,
		service,
	}
}

func (h *ASGCustomerEmailHandler) SendMarketingEmail(ctx *gin.Context) error {
	var req SendMarketingEmailRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		return err
	}

	return h.service.SendMarketingEmail(ctx, req.MaxCustomersPerTask, req.MinDaysOnboarded)
}

package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	costAllocation "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/service/cost_allocation"
	costAllocationIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/service/cost_allocation/iface"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CostAllocation struct {
	logger.Provider
	service costAllocationIface.ICostAllocationService
}

var (
	errEmptyCustomerID = errors.New("empty customer id")
)

func NewCostAllocation(loggerProvider logger.Provider, conn *connection.Connection) *CostAllocation {
	service, err := costAllocation.NewCostAllocationService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	return &CostAllocation{
		loggerProvider,
		service,
	}
}

func (h *CostAllocation) UpdateActiveCustomersHandler(ctx *gin.Context) error {
	err := h.service.UpdateActiveCustomers(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CostAllocation) UpdateMissingClustersHandler(ctx *gin.Context) error {
	err := h.service.UpdateMissingClusters(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CostAllocation) ScheduleInitStandaloneAccountsHandler(ctx *gin.Context) error {
	var req struct {
		BillingAccountIDs []string `json:"billingAccountIDs"`
	}

	if err := ctx.BindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.ScheduleInitStandaloneAccounts(ctx, req.BillingAccountIDs)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *CostAllocation) InitStandaloneAccountHandler(ctx *gin.Context) error {
	billingAccountID := ctx.Param("billingAccountID")

	err := h.service.InitStandaloneAccount(ctx, billingAccountID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

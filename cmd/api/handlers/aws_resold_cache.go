package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/fanout"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/savingsplans"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ResoldCacheService interface {
	RunCacheForSingleCustomer(ctx context.Context, customerID string) (*pkg.FlexsaveSavings, error)
}

type ResoldAWSCache struct {
	loggerProvider      logger.Provider
	fanoutService       fanout.Service
	service             ResoldCacheService
	manageFlexsave      manage.Service
	savingsPlansService savingsplans.Service
	doitemployees       doitemployees.ServiceInterface
}

func NewAWSResoldCache(log logger.Provider, conn *connection.Connection) *ResoldAWSCache {
	service := cache.NewService(log, conn)
	fanoutService := fanout.NewFanoutService(log, conn)
	manageFlexsaveService := manage.NewService(log, conn, service)

	savingsPlansService := savingsplans.NewService(log, conn)

	return &ResoldAWSCache{
		log,
		fanoutService,
		service,
		manageFlexsaveService,
		*savingsPlansService,
		doitemployees.NewService(conn),
	}
}

func (h *ResoldAWSCache) CreateForSingleCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	cacheData, err := h.service.RunCacheForSingleCustomer(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, cacheData, http.StatusOK)
}

func (h *ResoldAWSCache) CreateCacheForAllCustomers(ctx *gin.Context) error {
	err := h.fanoutService.CreateCacheForAllCustomers(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *ResoldAWSCache) CreateSavingsPlansCacheForAllCustomers(ctx *gin.Context) error {
	err := h.fanoutService.CreateSavingsPlansCacheForAllCustomers(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *ResoldAWSCache) UpdateCustomerStatuses(ctx *gin.Context) error {
	err := h.manageFlexsave.PayerStatusUpdateForEnabledCustomers(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	err = h.manageFlexsave.EnableEligiblePayers(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *ResoldAWSCache) Disable(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return handleServiceError(errors.New("customer id not provided"))
	}

	err := h.manageFlexsave.Disable(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *ResoldAWSCache) CanDisable(ctx *gin.Context) error {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if !doitEmployee {
		return web.Respond(ctx, false, http.StatusOK)
	}

	isFlexsaveAdmin, err := h.doitemployees.CheckDoiTEmployeeRole(ctx, flexsaveresold.FlexsaveAdmin, email)
	if err != nil {
		return handleServiceError(err)
	}

	if common.Production && !isFlexsaveAdmin {
		return web.Respond(ctx, false, http.StatusOK)
	}

	return web.Respond(ctx, true, http.StatusOK)
}

func (h *ResoldAWSCache) MPAActivatedHandler(ctx *gin.Context) error {
	accountNumber := ctx.Param("accountNumber")

	err := h.manageFlexsave.HandleMPAActivation(ctx, accountNumber)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *ResoldAWSCache) CustomerSavingsPlansCache(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	cacheData, err := h.savingsPlansService.CustomerSavingsPlansCache(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, cacheData, http.StatusOK)
}

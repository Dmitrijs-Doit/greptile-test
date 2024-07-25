package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/tiers/domain"
	tiers "github.com/doitintl/hello/scheduled-tasks/tiers/service"
	tiersService "github.com/doitintl/tiers/service"
)

type TiersHandler struct {
	loggerProvider logger.Provider
	tiersService   *tiers.TiersService
}

func NewTiersHandler(log logger.Provider, conn *connection.Connection) *TiersHandler {
	service, err := tiers.NewTiersService(log, conn)
	if err != nil {
		panic(err)
	}

	return &TiersHandler{
		log,
		service,
	}
}

// InitOnboarding creates onboarding document for the 1st time
func (h *TiersHandler) SendTrialNotifications(ctx *gin.Context) error {
	dryRun := ctx.Query("dryRun") == "true"

	if err := h.tiersService.SendTrialNotifications(ctx, dryRun); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *TiersHandler) UpdateTier(ctx *gin.Context) error {
	id := ctx.Param("id")
	if id == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	var tierUpdate tiers.TierUpdateRequest
	if err := ctx.ShouldBindJSON(&tierUpdate); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.tiersService.UpdateTier(ctx, id, &tierUpdate); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *TiersHandler) SetCustomerTiers(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	var customerTiersSetRequest domain.SetCustomerTiersRequest

	if err := ctx.ShouldBindJSON(&customerTiersSetRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.tiersService.SetCustomerTiers(ctx, customerID, customerTiersSetRequest); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}
	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *TiersHandler) SetCustomersTier(ctx *gin.Context) error {
	if common.Production {
		return web.NewRequestError(errors.New("cannot set customers tier in production"), http.StatusUnauthorized)
	}

	var customerTierSetRequest domain.SetCustomersTierRequest
	if err := ctx.ShouldBindJSON(&customerTierSetRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(customerTierSetRequest.CustomersTiers) == 0 {
		return web.NewRequestError(errors.New("no customers or tiers provided"), http.StatusBadRequest)
	}

	if err := h.tiersService.SetCustomersTier(ctx, customerTierSetRequest); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *TiersHandler) CustomerCanAccessFeature(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	var req tiers.CustomerFeatureAccessRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	access, err := h.tiersService.CustomerCanAccessFeature(ctx, customerID, pkg.TiersFeatureKey(req.Key))
	if err != nil {
		if errors.Is(err, tiersService.ErrInvalidFeatureKey) {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, access, http.StatusOK)
}

func (h *TiersHandler) TurnOffPresentationMode(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	var customerTiersSetRequest domain.SetCustomerTiersRequest

	if err := ctx.ShouldBindJSON(&customerTiersSetRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.tiersService.TurnOffPresentationMode(ctx, customerID, customerTiersSetRequest); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}
	return web.Respond(ctx, nil, http.StatusOK)
}

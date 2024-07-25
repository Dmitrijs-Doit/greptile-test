package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/support/service"
)

type Support struct {
	loggerProvider logger.Provider
	service        service.SupportServiceInterface
}

func NewSupport(log logger.Provider, conn *connection.Connection) *Support {
	supportService := service.NewSupportService(log, conn)

	return &Support{
		log,
		supportService,
	}
}

func (h *Support) ListPlatforms(ctx *gin.Context) error {
	logger := h.loggerProvider(ctx)
	platforms, err := h.service.ListPlatforms(ctx)

	if err != nil {
		logger.Error(err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, platforms, http.StatusOK)
}

func (h *Support) ListProducts(ctx *gin.Context) error {
	logger := h.loggerProvider(ctx)
	platform := ctx.Query("platform")
	products, err := h.service.ListProducts(ctx, platform)

	if err != nil {
		logger.Error(err)

		if err == service.ErrInvalidPlatform {
			return web.NewRequestError(service.ErrInvalidPlatform, http.StatusBadRequest)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, products, http.StatusOK)
}

type RequestBody struct {
	ServiceType ServiceType `json:"serviceType"`
	PackageType string      `json:"packageType"`
	Email       string      `json:"email"`
}

type ServiceType string

const (
	OneTime      ServiceType = "one-time"
	Subscription ServiceType = "subscription"
)

func (h *Support) ChangeCustomerTier(ctx *gin.Context) error {
	logger := h.loggerProvider(ctx)
	customerID := ctx.Param("customerID")

	var req RequestBody

	var err error

	if err = ctx.BindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.ServiceType == "one-time" {
		err = h.service.ApplyOneTimeSupport(ctx, customerID, pkg.OneTimeProductType(req.PackageType), req.Email)
	} else {
		err = h.service.ApplyNewSupportTier(ctx, customerID, pkg.TierNameType(req.PackageType))
	}

	if err != nil {
		logger.Error(err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

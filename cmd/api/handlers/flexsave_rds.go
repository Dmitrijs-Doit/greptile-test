package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/cache"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/manage"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FlexsaveRDS struct {
	loggerProvider logger.Provider
	cacheService   cache.Service
	manageService  manage.Service
}

func NewFlexsaveRDS(log logger.Provider, conn *connection.Connection) *FlexsaveRDS {
	cacheService := cache.NewService(log, conn)
	manageService := manage.NewService(log, conn)

	return &FlexsaveRDS{
		log,
		cacheService,
		manageService,
	}
}

func (h *FlexsaveRDS) Cache(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	exists, err := h.cacheService.CheckCacheExists(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	if !exists {
		err = h.cacheService.CreateEmptyCache(ctx, customerID)
		if err != nil {
			return handleServiceError(err)
		}
	}

	err = h.cacheService.RunCache(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveRDS) Enable(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	err := h.manageService.Enable(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

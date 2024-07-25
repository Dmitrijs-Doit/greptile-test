package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/cache"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/sagemaker/manage"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FlexsaveSageMaker struct {
	loggerProvider logger.Provider
	cacheService   cache.Service
	manageService  manage.Service
}

func NewFlexsaveSageMaker(log logger.Provider, conn *connection.Connection) *FlexsaveSageMaker {
	cacheService := cache.NewService(log, conn)
	manageService := manage.NewService(log, conn)

	return &FlexsaveSageMaker{
		log,
		cacheService,
		manageService,
	}
}

func (h *FlexsaveSageMaker) Cache(ctx *gin.Context) error {
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

func (h *FlexsaveSageMaker) Enable(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	err := h.manageService.Enable(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

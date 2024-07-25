package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack"
)

func (h *CloudConnect) Health(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if err := h.service.Health(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func AWSPermissionsHandler(ctx *gin.Context) error {
	cloudconnect.AWSPermissionsHandler(ctx)

	return nil
}

func GetSlackSharedChannelsInfo(ctx *gin.Context) error {
	slack.GetSlackSharedChannelsInfo(ctx)

	return nil
}

func (h *GoogleCloud) GetCustomerServicesLimitsGCP(ctx *gin.Context) error {
	if err := h.service.GetCustomerServicesLimits(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *GoogleCloud) GetCustomersRecommendations(ctx *gin.Context) error {
	if err := h.service.GetCustomersRecommendations(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

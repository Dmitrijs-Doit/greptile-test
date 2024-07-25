package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/awsAssetsSupport"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AWSAssetsSupportHandler struct {
	*logger.Logging
	service *awsAssetsSupport.AwsAssetsSupportService
}

func NewAWSAssetsSupportHandler(log *logger.Logging, conn *connection.Connection) *AWSAssetsSupportHandler {
	service, err := awsAssetsSupport.NewAWSAssetsSupportService(log, conn)

	if err != nil {
		panic(err)
	}

	return &AWSAssetsSupportHandler{
		log,
		service,
	}
}

// handlers per function

func (h *AWSAssetsSupportHandler) UpdateAWSSupportAssetsTypeInFS(ctx *gin.Context) error {
	if err := h.service.UpdateAWSSupportAssetsTypeInFS(ctx); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

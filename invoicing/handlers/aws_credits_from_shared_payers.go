package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/credits"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AWSSharedPayersCreditsHandler struct {
	*logger.Logging
	service *credits.AwsSharedPayersCreditService
}

func NewSharedPayersCreditsHandler(log *logger.Logging, conn *connection.Connection) *AWSSharedPayersCreditsHandler {
	service, err := credits.NewAWSSharedPayersCredits(log, conn)
	if err != nil {
		panic(err)
	}

	return &AWSSharedPayersCreditsHandler{
		log,
		service,
	}
}

func (h *AWSSharedPayersCreditsHandler) UpdateSharedPayersCredits(ctx *gin.Context) error {
	if err := h.service.UpdateAWSSharedPayersCredits(ctx); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

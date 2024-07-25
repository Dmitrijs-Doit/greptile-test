package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Webhook struct {
	*logger.Logging
	*connection.Connection
}

func NewWebhook(log *logger.Logging, conn *connection.Connection) *Webhook {
	return &Webhook{
		log,
		conn,
	}
}

func AWSUpdateRoleHandler(ctx *gin.Context) error {
	cloudconnect.AWSUpdateRoleHandler(ctx)

	return nil
}

func AWSDeleteRoleHandler(ctx *gin.Context) error {
	cloudconnect.AWSDeleteRoleHandler(ctx)

	return nil
}

func AWSUpdateFeature(ctx *gin.Context) error {
	cloudconnect.AWSUpdateFeature(ctx)

	return nil
}

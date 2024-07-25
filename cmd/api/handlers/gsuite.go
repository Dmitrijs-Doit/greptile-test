package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/gsuite"
)

func SubscriptionsListHandler(ctx *gin.Context) error {
	gsuite.SubscriptionsListHandler(ctx)

	return nil
}

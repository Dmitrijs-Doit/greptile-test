package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/receipt"
)

func Aggregate(ctx *gin.Context) error {
	receipt.Aggregate(ctx)

	return nil
}

func AccountReceiveables(ctx *gin.Context) error {
	receipt.AccountReceiveables(ctx)

	return nil
}

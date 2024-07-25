package handlers

import (
	"github.com/gin-gonic/gin"

	bqlens "github.com/doitintl/hello/scheduled-tasks/bq-lens"
)

func InvokeBigQueryProcess(ctx *gin.Context) error {
	bqlens.InvokeBigQueryProcess(ctx)

	return nil
}

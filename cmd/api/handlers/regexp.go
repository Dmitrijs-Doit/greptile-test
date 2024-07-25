package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func ValidateRegexp(ctx *gin.Context) error {
	common.ValidateRegexp(ctx)

	return nil
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func Ping(ctx *gin.Context) error {
	_ = web.Respond(ctx, nil, http.StatusOK)
	return nil
}

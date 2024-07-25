package zerobounce

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func ValidateEmailHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	var params ValidateParams
	if err := ctx.ShouldBindQuery(&params); err != nil {
		l.Error(err)
		ctx.AbortWithError(http.StatusBadRequest, err)

		return
	}

	service := New()

	resp, err := service.Validate(&params)
	if err != nil {
		l.Error(err)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	ctx.JSON(http.StatusOK, resp)
}

package mid

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func ValidatePathParamNotEmpty(paramName string) web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			if paramValue := ctx.Param(paramName); paramValue == "" {
				return web.NewRequestError(errors.New("error: "+paramName+" cannot be empty"), http.StatusBadRequest)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

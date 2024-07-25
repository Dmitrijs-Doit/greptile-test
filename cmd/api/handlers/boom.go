package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

func Boom(ctx *gin.Context) error {
	boomType := ctx.Query("type")
	message := ctx.Query("message")

	errorMessage := strings.TrimSpace(fmt.Sprintf("boom (%s) %s", boomType, message))

	var returnVal error = nil

	switch boomType {
	case "message":
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.WithScope(func(scope *sentry.Scope) {
				hub.CaptureMessage(errorMessage)
			})
		}
	case "abort":
		err := errors.New(errorMessage)
		ctx.AbortWithError(http.StatusServiceUnavailable, err)
	case "return-value":
		err := errors.New(errorMessage)
		returnVal = web.NewRequestError(err, http.StatusInternalServerError)
	default:
		err := errors.New(errorMessage)
		panic(err)
	}

	if returnVal == nil {
		if _, ok := ctx.GetQuery("returnError"); ok {
			returnVal = errors.New(errorMessage + " (from return value)")
		}
	}

	return returnVal
}

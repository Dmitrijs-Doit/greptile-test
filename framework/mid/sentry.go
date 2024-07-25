package mid

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func logSentryErrorMessage(ctx *gin.Context, errMessage error) {
	if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			scope.SetLevel(sentry.LevelError)
			hub.CaptureMessage(errMessage.Error())
		})
	}
}

// Sentry middleware, we log if the handler returned error 500 or if it returns nil but was aborted with Error
func Sentry() web.Middleware {
	f := func(before web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			if err := before(ctx); err != nil {
				if webErr, ok := err.(*web.Error); ok {
					if webErr.Status >= http.StatusInternalServerError {
						logSentryErrorMessage(ctx, err)
					}
				}

				return err
			}

			// access the status we are sending
			status := ctx.Writer.Status()
			if status >= http.StatusBadRequest {
				lastErr := ctx.Errors.Last()
				if lastErr != nil {
					logSentryErrorMessage(ctx, lastErr.Err)
				}
			}

			return nil
		}

		return h
	}

	return f
}

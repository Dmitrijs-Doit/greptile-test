package mid

import (
	"fmt"
	"runtime/debug"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/internal"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Panics recovers from panics and converts the panic to an error.
func Panics() web.Middleware {
	f := func(after web.Handler) web.Handler {
		h := func(ctx *gin.Context) (err error) {
			v, ok := internal.DataFromContext(ctx)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			log := logger.FromContext(ctx)

			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic: %v", r)
					log.Errorf("%s: %s\n%s", v.TraceID, err, debug.Stack())

					if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
						hub.WithScope(func(scope *sentry.Scope) {
							hub.Recover(err)
							sentry.Flush(time.Second * 5)
						})
					}
				}
			}()

			return after(ctx)
		}

		return h
	}

	return f
}

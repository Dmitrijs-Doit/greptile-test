package mid

import (
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/internal"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Errors handles errors coming out of the call chain. It detects normal
// application errors which are used to respond to the client in a uniform way.
func Errors() web.Middleware {
	f := func(before web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			v, ok := internal.DataFromContext(ctx)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			log := logger.FromContext(ctx)

			if err := before(ctx); err != nil {
				log.Errorf("%s: ERROR: %v", v.TraceID, err)

				if err := web.RespondError(ctx, err); err != nil {
					return err
				}

				// If we receive the shutdown err we need to return it
				// back to the base handler to shutdown the service.
				if ok := web.IsShutdown(err); ok {
					return err
				}
			}

			return nil
		}

		return h
	}

	return f
}

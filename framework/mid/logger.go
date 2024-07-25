package mid

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/internal"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	healthCheckExcludePath = "/health"
)

// Logger writes some information about the request to the logs in the
// format: TraceID : (200) GET /foo -> IP ADDR (latency)
func Logger() web.Middleware {
	f := func(before web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			if ctx.Request.URL.String() == healthCheckExcludePath {
				return before(ctx)
			}

			v, ok := internal.DataFromContext(ctx)
			if !ok {
				return web.NewShutdownError("web value missing from context")
			}

			log := logger.FromContext(ctx)

			log.Printf("%s: started : %s %s -> %s",
				v.TraceID,
				ctx.Request.Method, ctx.Request.URL.Path, ctx.Request.RemoteAddr,
			)

			err := before(ctx)

			if err != nil {
				log.Printf("ERROR: %s", err)
			} else if v.StatusCode >= http.StatusBadRequest || v.StatusCode == 0 {
				lastErr := ctx.Errors.Last()
				if lastErr != nil {
					log.Errorf("Request fails %s", lastErr)
				}
			}

			log.Printf("%s: completed : %s %s -> %s (%d) (%s)",
				v.TraceID,
				ctx.Request.Method, ctx.Request.URL.Path, ctx.Request.RemoteAddr,
				v.StatusCode, time.Since(v.Now),
			)

			return err
		}

		return h
	}

	return f
}

package mid

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/cmd/api/handlers"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/mixpanel"
)

func ExternalAPIMixpanel(mixpanelService *handlers.Mixpanel, method mixpanel.Method, feature mixpanel.Feature) web.Middleware {
	f := func(before web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			log := logger.FromContext(ctx)
			t := time.Now()

			ctx.Next()
			afterErr := before(ctx)
			latency := time.Since(t)

			go func() {
				email := ctx.GetString("email")
				statusCode := ctx.Writer.Status()

				if webErr, ok := afterErr.(*web.Error); ok {
					statusCode = webErr.Status
				}

				err := mixpanelService.TrackRequest(ctx, email, &mixpanel.TrackAPIRequest{
					CustomerContext: ctx.GetString(auth.CtxKeyVerifiedCustomerID),
					Method:          method,
					Feature:         feature,
					RestMethod:      ctx.Request.Method,
					RequestURL:      ctx.Request.URL.Path,
					QueryParams:     ctx.Request.URL.Query(),
					RequestFullPath: ctx.Request.RequestURI,
					Status:          statusCode,
					LatencyTime:     latency.String(),
					UserAgent:       ctx.Request.UserAgent(),
				})

				if err != nil {
					log.Error(err)
				}
			}()

			return afterErr
		}

		return h
	}

	return f
}

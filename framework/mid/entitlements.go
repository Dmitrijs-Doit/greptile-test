package mid

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/tiers/service"
)

// HasEntitlementFunc returns a web.Middleware generator function.
//
// The returned function takes a TiersFeatureKey and returns a web.Middleware.
// The returned function is a middleware that can be used to evaluate whether a customer has the necessary entitlement
// for the provided feature. The middleware depends on either 'customerID' or 'verifiedCustomerId' being set on the
// context
//
// Example usage:
//
//	hasEntitlement := HasEntitlementFunc(tierService)
//	attributionGroup := attributionGroup.NewSubgroup("/attribution", hasEntitlement(tierDAL.TiersFeatureKeyAnalyticsAttributionGroups))
func HasEntitlementFunc(tierService service.TierServiceIface) func(key pkg.TiersFeatureKey) web.Middleware {
	return func(key pkg.TiersFeatureKey) web.Middleware {
		return func(handler web.Handler) web.Handler {
			return func(ctx *gin.Context) error {
				isDoitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)
				if isDoitEmployee {
					return handler(ctx)
				}

				customerID := ctx.GetString(common.CtxKeys.CustomerID)

				if auth.IsExternalAPIFlow(ctx) {
					customerID = ctx.GetString(auth.CtxKeyVerifiedCustomerID)
				}

				if customerID == "" {
					return web.NewRequestError(errors.New("no customer id provided for entitlement check"), http.StatusBadRequest)
				}

				if ok, err := tierService.CustomerCanAccessFeature(ctx, customerID, key); err != nil || !ok {
					return web.NewRequestError(service.ErrFeatureNotAccessible, http.StatusForbidden)
				}

				return handler(ctx)
			}
		}
	}
}

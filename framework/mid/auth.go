package mid

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const (
	dayDuration                  = 24 * time.Hour
	MaxValidRefreshTokenDuration = 2 * dayDuration

	// https://cloud.google.com/tasks/docs/creating-appengine-tasks#firewall_rules
	appEngineUserIPHeader = "X-Appengine-User-IP"
	appEngineCloudTasksIP = "0.1.0.2"
)

func GetAllowedCloudJobsEmails() []string {
	return []string{
		fmt.Sprintf("gcp-jobs@%s.iam.gserviceaccount.com", common.ProjectID),
		"gcp-jobs@doitintl-cmp-reports-scheduler.iam.gserviceaccount.com",
		fmt.Sprintf("%s-compute@developer.gserviceaccount.com", common.ProjectNumber),
	}
}

func getAllowedInternalServiceEmailsByEnv() []string {
	if common.Production {
		return []string{
			"concedefy-accessor@doit-support.iam.gserviceaccount.com",
			"doit-aws-cli@doit-support.iam.gserviceaccount.com",
			"integration-svc-sa@aws-mp-integration-svc-prod.iam.gserviceaccount.com",
			"cloud-composer@cmp-aws-etl-prod.iam.gserviceaccount.com",
		}
	}

	return []string{
		"concedefy-accessor@concedefy-oidc-staging.iam.gserviceaccount.com",
		"doit-aws-cli@concedefy-oidc-staging.iam.gserviceaccount.com",
		"integration-svc-sa@aws-mp-integration-service.iam.gserviceaccount.com",
		"cloud-composer@cmp-aws-etl-dev.iam.gserviceaccount.com",
	}
}

func getAllowedInternalServiceEmailsByProject() []string {
	return []string{
		fmt.Sprintf("%s@appspot.gserviceaccount.com", common.ProjectID),
		fmt.Sprintf("%s-compute@developer.gserviceaccount.com", common.ProjectNumber),
		fmt.Sprintf("anomalies-notification@%s.iam.gserviceaccount.com", common.ProjectID),
		fmt.Sprintf("ingestion-api-frontend-sa@%s.iam.gserviceaccount.com", common.ProjectID),
	}
}

func GetAllowedMarketplaceGcpEventsEmails() []string {
	return []string{
		fmt.Sprintf("gcp-marketplace-push-sub@%s.iam.gserviceaccount.com", common.ProjectID),
	}
}

// Auth errors
var (
	ErrForbidden    = errors.New("forbidden operation")
	ErrUnauthorized = errors.New("unauthorized operation")
)

var externalApiAuthService = auth.NewService()

// AuthRequired middleware that auth requests coming from client app
func AuthRequired(conn *connection.Connection) web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			l := logger.FromContext(ctx)

			token, authTime, err := fb.VerifyIDToken(ctx)
			if err != nil {
				return web.NewRequestError(err, http.StatusUnauthorized)
			}

			claims := token.Claims

			ctx.Set("claims", claims)
			ctx.Set("uid", token.UID)
			ctx.Set("tenantId", token.Firebase.Tenant)

			// If it's been too long since the user last logged in, check if token is revoked
			if time.Since(*authTime) > MaxValidRefreshTokenDuration {
				if err := fb.VerifyIDTokenAndCheckRevoked(ctx, token.Firebase.Tenant); err != nil {
					return web.NewRequestError(err, http.StatusUnauthorized)
				}
			}

			// Set email in context
			email, ok := claims["email"]
			if !ok {
				return web.NewRequestError(ErrUnauthorized, http.StatusUnauthorized)
			}

			emailStr := email.(string)
			ctx.Set("email", strings.ToLower(emailStr))

			// Set name in context
			if name, ok := claims["name"]; ok {
				ctx.Set("name", name.(string))
			}

			l.SetLabels(map[string]string{
				"email": emailStr,
				"uid":   token.UID,
			})

			l.Printf("request executed by email [%s] uid [%s] tenant [%s]", emailStr, token.UID, token.Firebase.Tenant)

			conn.FirestoreWithContext(ctx)

			isDoitPartner, _ := claims["doitPartner"].(bool)
			isDoitEmployee, _ := claims["doitEmployee"].(bool)
			isDoitOwner, _ := claims["doitOwner"].(bool)

			ctx.Set("doitPartner", isDoitPartner)
			ctx.Set("doitEmployee", isDoitEmployee)
			ctx.Set("doitOwner", isDoitOwner)

			userID, ok := claims["userId"]
			if !ok {
				return web.NewRequestError(ErrForbidden, http.StatusForbidden)
			}

			ctx.Set("userId", userID)

			return handler(ctx)
		}

		return h
	}

	return f
}

// AuthDoitEmployee middleware validates that the user is a doit employee
func AuthDoitEmployee() web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			if !ctx.GetBool("doitEmployee") {
				return web.NewRequestError(ErrForbidden, http.StatusForbidden)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

// AuthDoitOwner middleware validates that the user is a doit owner
func AuthDoitOwner() web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			if !ctx.GetBool("doitOwner") {
				return web.NewRequestError(ErrForbidden, http.StatusForbidden)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

func AssertCacheDisableAccess(permissions permissions.Service) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx *gin.Context) error {
			updatedContext, err := permissions.AssertCacheDisableAccess(ctx)
			if err != nil {
				return err
			}

			return handler(updatedContext)
		}
	}
}

func AssertCacheEnableAccess(permissions permissions.Service) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx *gin.Context) error {
			updatedContext, err := permissions.AssertCacheEnableAccess(ctx)
			if err != nil {
				return err
			}

			return handler(updatedContext)
		}
	}
}

func AssertUserHasPermissions(permissions []string, conn *connection.Connection) web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx *gin.Context) error {
			if ctx.GetBool("doitEmployee") {
				return handler(ctx)
			}

			if err := externalApiAuthService.AssertUserHasPermissions(ctx, permissions, conn.Firestore(ctx)); err != nil {
				if strings.HasPrefix(err.Error(), common.MissingPermissionsPrefix) {
					return web.NewRequestError(err, http.StatusForbidden)
				}

				return web.NewRequestError(err, http.StatusInternalServerError)
			}

			return handler(ctx)
		}
	}
}

func AuthDoitEmployeeRole(conn *connection.Connection, role domain.DoitRole) web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			email := ctx.GetString("email")

			s := doitemployees.NewService(conn)

			hasPermissions, err := s.CheckDoiTEmployeeRole(ctx, string(role), email)
			if err != nil {
				return err
			}

			if !hasPermissions {
				return web.NewRequestError(ErrForbidden, http.StatusForbidden)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

func AuthMPAOrFlexsaveAdmin(conn *connection.Connection) web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			email := ctx.GetString("email")

			s := doitemployees.NewService(conn)

			isFlexsaveAdmin, err := s.CheckDoiTEmployeeRole(ctx, string(domain.DoitRoleFlexsaveAdmin), email)
			if err != nil {
				return err
			}

			isMPAAdmin, err := s.CheckDoiTEmployeeRole(ctx, string(domain.DoitRoleMasterPayerAccountOpsAdmin), email)
			if err != nil {
				return err
			}

			if isFlexsaveAdmin || isMPAAdmin {
				return handler(ctx)
			}

			return web.NewRequestError(ErrForbidden, http.StatusForbidden)
		}

		return h
	}

	return f
}

// AuthCustomerRequired middleware validates that the user belongs to the customer, sets the customerID on the context
// if successful
func AuthCustomerRequired() web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx *gin.Context) error {
			customerID := ctx.Param("customerID")
			if customerID == "" {
				return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
			}

			l := logger.FromContext(ctx)
			l.SetLabel("customerId", customerID)

			if !ctx.GetBool("doitEmployee") && !ctx.GetBool("doitPartner") {
				claims := ctx.GetStringMap("claims")
				if userCustomerID, ok := claims["customerId"]; !ok || userCustomerID.(string) != customerID {
					return web.NewRequestError(ErrForbidden, http.StatusForbidden)
				}
			}

			ctx.Set(common.CtxKeys.CustomerID, customerID)

			return handler(ctx)
		}
	}
}

// AuthEntityRequired middleware validates that the user belongs to the entity
func AuthEntityRequired() web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			entityID := ctx.Param("entityID")
			if entityID == "" {
				return web.NewRequestError(errors.New("missing entity id"), http.StatusBadRequest)
			}

			l := logger.FromContext(ctx)
			l.SetLabel("entityId", entityID)

			return handler(ctx)
		}

		return h
	}

	return f
}

func AddDefaultLoggerLabels() web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			l := logger.FromContext(ctx)

			// Add customerID label if present
			customerID := ctx.Param("customerID")
			if customerID != "" {
				l.SetLabel("customerId", customerID)
			}

			// Add entityID label if present
			entityID := ctx.Param("entityID")
			if entityID != "" {
				l.SetLabel("entityId", entityID)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

// validateIDTokenPayload validates the authorization header bearer token
// using Google's idtoken package
func validateIDTokenPayload(ctx *gin.Context) (*idtoken.Payload, error) {
	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		err := errors.New("no authorization header")
		return nil, web.NewRequestError(err, http.StatusUnauthorized)
	}

	parts := strings.Split(authHeader, " ")

	// Validate auth header structure
	if len(parts) != 2 || parts[0] != "Bearer" {
		err := errors.New("invalid authorization header format, expected Bearer <token>")
		return nil, web.NewRequestError(err, http.StatusUnauthorized)
	}

	// Validate auth header bearer token
	payload, err := validateIDTokenWithAudienceList(ctx, parts[1], []string{common.GAEService, common.APIGateway})
	if err != nil {
		return nil, web.NewRequestError(err, http.StatusUnauthorized)
	}

	return payload, nil
}

func validateIDTokenWithAudienceList(ctx *gin.Context, authHeader string, audiences []string) (*idtoken.Payload, error) {
	l := logger.FromContext(ctx)

	for _, audience := range audiences {
		payload, err := idtoken.Validate(ctx, authHeader, audience)
		if err == nil {
			return payload, nil
		}
	}

	l.Println("invalid token: does not match any valid audience")

	return nil, errors.New("invalid token: does not match any valid audience")
}

func AuthInternalServiceAccounts() web.Middleware {
	var internalEmails []string

	internalEmails = append(internalEmails, getAllowedInternalServiceEmailsByEnv()...)
	internalEmails = append(internalEmails, getAllowedInternalServiceEmailsByProject()...)

	return AuthServiceAccount(internalEmails)
}

// AuthServiceAccount validates requests from service accounts in production
func AuthServiceAccount(validClaimEmails []string) web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			l := logger.FromContext(ctx)

			// Skip validation when running in localhost
			if common.IsLocalhost {
				return handler(ctx)
			}

			// Skip OIDC auth validation when running app engine jobs
			if ctx.Request.Header.Get(appEngineUserIPHeader) == appEngineCloudTasksIP {
				return handler(ctx)
			}

			payload, err := validateIDTokenPayload(ctx)

			if err != nil {
				return err
			}

			// Verify email claim matches the required service account
			if claimsEmail, prs := payload.Claims["email"]; !prs || !isClaimEmailValid(validClaimEmails, claimsEmail.(string)) {
				l.Println("invalid token: does not match any valid claims email", payload.Claims["email"], validClaimEmails)
				return web.NewRequestError(ErrForbidden, http.StatusForbidden)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

func isClaimEmailValid(emails []string, claimsEmail string) bool {
	return slice.Contains(emails, claimsEmail)
}

// ExternalAPIAuthMiddleware middleware that auth requests coming from external API
// and sets logging labels for the user's email, uid, userID, and doitEmployee.
func ExternalAPIAuthMiddleware() web.Middleware {
	return func(handler web.Handler) web.Handler {
		return func(ctx *gin.Context) error {
			if err := externalApiAuthService.ProcessToken(ctx); err != nil {
				return web.NewRequestError(errors.New(err.Message), err.Code)
			}

			l := logger.FromContext(ctx)
			l.SetLabels(map[string]string{
				common.CtxKeys.Email:        ctx.GetString(common.CtxKeys.Email),
				common.CtxKeys.UID:          ctx.GetString(common.CtxKeys.UID),
				common.CtxKeys.UserID:       ctx.GetString(common.CtxKeys.UserID),
				common.CtxKeys.DoitEmployee: strconv.FormatBool(ctx.GetBool(common.CtxKeys.DoitEmployee)),
			})

			return handler(ctx)
		}
	}
}

func ExternalAPIAssertCustomerTypeProductOnly() web.Middleware {
	f := func(handler web.Handler) web.Handler {
		h := func(ctx *gin.Context) error {
			err := externalApiAuthService.AssertIsCustomerTypeProductOnly(ctx)
			if err != nil {
				return web.NewRequestError(err, http.StatusForbidden)
			}

			return handler(ctx)
		}

		return h
	}

	return f
}

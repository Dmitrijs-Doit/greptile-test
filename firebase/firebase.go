package firebase

import (
	"errors"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

var errNoAuthHeader = errors.New("No Authorization Header found")
var errInvalidAuthHeader = errors.New("Invalid Authorization Header found")

func tokenAuthTime(token *auth.Token) (*time.Time, error) {
	if authTime, prs := token.Claims["auth_time"]; prs {
		if v, ok := authTime.(float64); ok {
			t := time.Unix(int64(v), 0)
			return &t, nil
		}
	}

	return nil, errors.New("invalid auth token")
}

// VerifyIDToken : Verify auth header
func VerifyIDToken(ctx *gin.Context) (*auth.Token, *time.Time, error) {
	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		return nil, nil, errNoAuthHeader
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, nil, errInvalidAuthHeader
	}

	idToken := strings.Split(authHeader, " ")[1]

	auth, err := App.Auth(ctx)
	if err != nil {
		return nil, nil, err
	}

	demoAuth, err := DemoApp.Auth(ctx)
	if err != nil {
		return nil, nil, err
	}

	token, err := auth.VerifyIDToken(ctx, idToken)
	if err != nil {
		token, err = demoAuth.VerifyIDToken(ctx, idToken)
		if err != nil {
			return nil, nil, err
		}
	}

	authTime, err := tokenAuthTime(token)
	if err != nil {
		return nil, nil, err
	}

	return token, authTime, nil
}

// VerifyIDTokenAndCheckRevoked verify request authorization header
func VerifyIDTokenAndCheckRevoked(ctx *gin.Context, tenantID string) error {
	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		return errNoAuthHeader
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return errInvalidAuthHeader
	}

	auth, err := App.Auth(ctx)
	if err != nil {
		return err
	}

	tenantAuth, err := auth.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return err
	}

	idToken := strings.Split(authHeader, " ")[1]
	if _, err := tenantAuth.VerifyIDTokenAndCheckRevoked(ctx, idToken); err != nil {
		return err
	}

	return nil
}

package user

import (
	"net/http"
	"strings"

	"math/rand"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func randSeq(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func Exists(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	l.SetLabels(map[string]string{
		"uid":             ctx.GetString("uid"),
		logger.LabelEmail: ctx.GetString("email"),
	})

	type requestBody struct {
		Email string `json:"email"`
	}

	var body requestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	l.Infof("%#v", body)

	fs := common.GetFirestoreClient(ctx)
	customerID := ctx.Param("customerID")
	tenantAuth, err := tenant.GetTenantAuthClientByCustomer(ctx, fs, customerID)

	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(body.Email))

	// Check if email has a user record
	userExists, err := isUserExists(ctx, fs, body.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if userExists {
		ctx.String(http.StatusConflict, "%s already exists", email)
		return
	}

	// Check if email has an invite record
	userInvited, err := isUserInvited(ctx, fs, body.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if userInvited {
		ctx.String(http.StatusNotFound, "%s was already invited", email)
		return
	}

	// Check if email exists in auth
	{
		if user, err := tenantAuth.GetUserByEmail(ctx, email); err != nil {
			if !auth.IsUserNotFound(err) {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		} else {
			l.Infof("user already exists: %#v", user)
			ctx.String(http.StatusBadRequest, "%s already exists", email)

			return
		}
	}
}

func GetUIDByEmail(ctx *gin.Context) {
	email := ctx.Request.URL.Query().Get("email")
	if email == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	customerID := ctx.Param("customerID")
	fs := common.GetFirestoreClient(ctx)
	tenantAuth, err := tenant.GetTenantAuthClientByCustomer(ctx, fs, customerID)

	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if user, err := tenantAuth.GetUserByEmail(ctx, email); err != nil {
		if !auth.IsUserNotFound(err) {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	} else {
		ctx.String(http.StatusOK, "%s", user.UserInfo.UID)
		return
	}
}

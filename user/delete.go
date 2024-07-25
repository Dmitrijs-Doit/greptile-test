package user

import (
	"errors"
	"fmt"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Check if email has an invite record
func isUserInvited(ctx *gin.Context, fs *firestore.Client, email string) (bool, error) {
	docs, err := fs.Collection("invites").Where("email", "==", email).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	if len(docs) > 0 {
		return true, nil
	}

	return false, nil
}

// Check if email has a user record
func isUserExists(ctx *gin.Context, fs *firestore.Client, email string) (bool, error) {
	docs, err := fs.Collection("users").Where("email", "==", email).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return false, err
	}

	if len(docs) > 0 {
		return true, nil
	}

	return false, nil
}

func Delete(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	l.SetLabels(map[string]string{
		"uid":             ctx.GetString("uid"),
		logger.LabelEmail: ctx.GetString("email"),
	})

	if !ctx.GetBool(common.DoitEmployee) {
		ctx.AbortWithError(http.StatusForbidden, errors.New("user not authorized"))
		return
	}

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

	userInvited, err := isUserInvited(ctx, fs, body.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if userInvited {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("%s was already invited", body.Email))
		return
	}

	userExists, err := isUserExists(ctx, fs, body.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if userExists {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("%s already exists", body.Email))
		return
	}

	tenantAuth, err := tenant.GetTenantAuthClientByCustomer(ctx, fs, customerID)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	u, err := tenantAuth.GetUserByEmail(ctx, body.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if err := tenantAuth.DeleteUser(ctx, u.UID); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

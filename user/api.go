package user

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/api"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Response struct {
	AccessKey   string
	AccessToken string
}

var (
	externalAPIAuthService = auth.NewService()
)

func GenerateAPIToken(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	email := ctx.GetString(common.CtxKeys.Email)
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	l.Infof("email: %s", email)

	if email == "" {
		api.AbortMsg(ctx, 400, nil, api.ErrorEmailIsEmpty)
		return
	}

	userID := ctx.GetString(common.CtxKeys.UserID)
	l.Infof("userID: %s", userID)

	if userID == "" {
		api.AbortMsg(ctx, 400, nil, api.ErrorUserNotFound)
		return
	}

	uid := ctx.GetString(common.CtxKeys.UID)
	l.Infof("uid: %s", uid)

	if uid == "" {
		api.AbortMsg(ctx, 400, nil, api.ErrorUIDIsEmpty)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	userDoc, err := fs.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		api.AbortMsg(ctx, 400, err, api.ErrorUserNotFound)
		return
	}

	var curUser common.User
	if err := userDoc.DataTo(&curUser); err != nil {
		l.Error(err)
		ctx.AbortWithError(http.StatusBadRequest, err)

		return
	}
	// if there is already a key - do not create a new one
	if curUser.AccessKey != "" {
		l.Error(curUser.AccessKey)
		api.AbortMsg(ctx, 400, err, api.ErrorKeyExists)

		return
	}

	tokenString, claims, err := externalAPIAuthService.GenerateToken(ctx, uid)
	if err != nil {
		l.Error(err)

		if ctx.Writer.Status() == http.StatusOK {
			// if response status code is not error - return 500
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}

		return
	}

	l.Infof("expires: %#v", claims.ExpiresAt)
	l.Infof("email: %#v", claims.Subject)
	l.Infof("doitEmployee: %#v", claims.DoitEmployee)
	l.Infof("doitOwner: %#v", claims.DoitOwner)
	l.Infof("name: %#v", ctx.GetString("name"))

	// save key to firestore
	_, err = userDoc.Ref.Update(ctx, []firestore.Update{
		{FieldPath: []string{"accessKey"}, Value: claims.Key},
	})
	if err != nil {
		l.Errorf("Error update key for user %#v", claims.UserID)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	res := Response{
		AccessKey:   string(claims.Key),
		AccessToken: tokenString,
	}

	ctx.JSON(http.StatusOK, res)
}

func DeleteAPIKey(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	email := ctx.GetString(common.CtxKeys.Email)
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	l.Infof("email: %s", email)

	if email == "" {
		api.AbortMsg(ctx, 400, nil, api.ErrorUserNotFound)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	userID := ctx.GetString(common.CtxKeys.UserID)
	l.Infof("userID: %#v", userID)

	if userID == "" {
		api.AbortMsg(ctx, 400, nil, api.ErrorUserNotFound)
		return
	}

	userDoc, err := fs.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		api.AbortMsg(ctx, 400, err, api.ErrorUserNotFound)
		return
	}

	var curUser common.User
	if err := userDoc.DataTo(&curUser); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	// delete key from firebase
	_, err = userDoc.Ref.Update(ctx, []firestore.Update{
		{FieldPath: []string{"accessKey"}, Value: firestore.Delete},
	})
	if err != nil {
		l.Infof("Error deleting key for user %#v", userDoc.Ref.ID)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	res := Response{
		AccessKey:   "",
		AccessToken: "",
	}

	ctx.JSON(http.StatusOK, res)
}

func isDoiTEmployee(ctx *gin.Context) (bool, string) {
	return ctx.GetBool("doitEmployee"), ctx.GetString("uid")
}

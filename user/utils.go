package user

import (
	"fmt"
	"net/http"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
)

func UpdateUserDisplayName(ctx *gin.Context) {
	isDoitMember, _ := isDoiTEmployee(ctx)

	var userDetails doitemployees.UserDetails
	if err := ctx.ShouldBindJSON(&userDetails); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}

	currentUserEmail := ctx.GetString("email")

	if userDetails.Email == "" {
		ctx.AbortWithError(http.StatusBadRequest, fmt.Errorf("email is required"))
		return
	}

	fs := common.GetFirestoreClient(ctx)

	client, err := tenant.GetTenantAuthClientByCustomer(ctx, fs, userDetails.CustomerID)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	// if email is not of the current user then return
	if userDetails.Email != currentUserEmail && !isDoitMember {
		return
	}

	user, err := client.GetUserByEmail(ctx, userDetails.Email)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	uid := user.UserInfo.UID

	params := (&auth.UserToUpdate{}).
		DisplayName(userDetails.DisplayName)

	u, err := client.UpdateUser(ctx, uid, params)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"user": u,
	})
}

package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

func permissionsAuthorizer(ctx *gin.Context, conn *connection.Connection, requiredPermissions []common.Permission) error {
	if ctx.GetBool("doitEmployee") {
		return nil
	}

	userID := ctx.GetString("userId")
	if userID == "" {
		// this shouldn't happen as long as we withstand the assumption that doit employees are the only users without a userId
		return errors.New("expecting a valid userId on context")
	}

	fs := conn.Firestore(ctx)
	userRef := fs.Collection("users").Doc(userID)

	user, err := common.GetUser(ctx, userRef)
	if err != nil {
		return err
	}

	return user.HasRequiredPermissions(ctx, requiredPermissions)
}

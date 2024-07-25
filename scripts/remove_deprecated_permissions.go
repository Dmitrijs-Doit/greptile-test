package scripts

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type RemoveDeprecatedPermissionsInput struct {
	PermissionID string `json:"permission_id"`
}

// RemoveDeprecatedPermissions removes old deprecated legacy permissions IDs from "invites"
// TODO: add support to remove permissinos from users/roles as well.
func RemoveDeprecatedPermissions(ctx *gin.Context) []error {
	var params RemoveDeprecatedPermissionsInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	usersInvitesDocsnaps, err := fs.Collection("invites").
		Where("permissions", "array-contains", params.PermissionID).
		Documents(ctx).
		GetAll()
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if len(usersInvitesDocsnaps) > 0 {
		for _, docSnap := range usersInvitesDocsnaps {
			batch.Update(docSnap.Ref, []firestore.Update{
				{FieldPath: []string{"permissions"}, Value: firestore.ArrayRemove(params.PermissionID)},
			})
		}

		if errs := batch.Commit(ctx); len(errs) > 0 {
			return errs
		}
	}

	return nil
}

package flexsaveresold

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

const (
	FlexsaveSuperAdmin = "flexsave-super-admin"
	FlexsaveAdmin      = "flexsave-admin"
)

func assertPermissions(ctx context.Context, fs *firestore.Client, userID string) error {
	userRef := fs.Collection("users").Doc(userID)

	user, err := common.GetUser(ctx, userRef)
	if err != nil {
		return err
	}

	if !user.HasFlexSaveAdminPermission(ctx) {
		return web.ErrForbidden
	}

	return nil
}

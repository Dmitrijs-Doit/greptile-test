package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
)

type UserFirestore struct {
	userDAL *userDal.UserFirestoreDAL
}

func NewUserFirestore(conn *connection.Connection) *UserFirestore {
	return &UserFirestore{
		userDAL: userDal.NewUserFirestoreDALWithClient(conn.Firestore),
	}
}

func (d *UserFirestore) GetUser(ctx context.Context, userID string) (*common.User, error) {
	return d.userDAL.Get(ctx, userID)
}

func (d *UserFirestore) HasUsersPermission(ctx context.Context, user *common.User) bool {
	return user.HasUsersPermission(ctx)
}
func (d *UserFirestore) HasInvoicesPermission(ctx context.Context, user *common.User) bool {
	return user.HasInvoicesPermission(ctx)
}
func (d *UserFirestore) HasLicenseManagePermission(ctx context.Context, user *common.User) bool {
	return user.HasLicenseManagePermission(ctx)
}
func (d *UserFirestore) HasEntitiesPermission(ctx context.Context, user *common.User) bool {
	return user.HasEntitiesPermission(ctx)
}
func (d *UserFirestore) HasCloudAnalyticsPermission(ctx context.Context, user *common.User) bool {
	return user.HasCloudAnalyticsPermission(ctx)
}
func (d *UserFirestore) HasMetricsPermission(ctx context.Context, user *common.User) bool {
	return user.HasMetricsPermission(ctx)
}
func (d *UserFirestore) HasAttributionsPermission(ctx context.Context, user *common.User) bool {
	return user.HasAttributionsPermission(ctx)
}
func (d *UserFirestore) HasBudgetsPermission(ctx context.Context, user *common.User) bool {
	return user.HasBudgetsPermission(ctx)
}

package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
	"github.com/doitintl/hello/scheduled-tasks/user/dal"
)

type CAOwnerChecker struct {
	userDal *dal.UserFirestoreDAL
}

func NewCAOwnerChecker(conn *connection.Connection) *CAOwnerChecker {
	userDal := dal.NewUserFirestoreDALWithClient(conn.Firestore)

	return &CAOwnerChecker{
		userDal,
	}
}

func (c CAOwnerChecker) CheckCAOwner(ctx context.Context, es doitemployees.ServiceInterface, userID, email string) (bool, error) {
	if es.IsDoitEmployee(ctx) {
		return es.CheckDoiTEmployeeRole(ctx, permissionsDomain.DoitRoleCAOwnershipAssigner.String(), email)
	}

	if userID != "" {
		user, err := c.userDal.Get(ctx, userID)
		if err != nil {
			return false, err
		}

		return user.HasCAOwnerRoleAssignerPermission(ctx), nil
	}

	return false, nil
}

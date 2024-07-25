//go:generate mockery --output=./mocks --all
package permissions

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	permissionDAL "github.com/doitintl/hello/scheduled-tasks/iam/permission/dal"
	permissionDALIface "github.com/doitintl/hello/scheduled-tasks/iam/permission/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	"github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
)

type Service interface {
	AssertCacheEnableAccess(ctx *gin.Context) (*gin.Context, error)
	AssertCacheDisableAccess(ctx *gin.Context) (*gin.Context, error)
}

type service struct {
	doitemployees doitemployees.ServiceInterface
	userDAL       iface.IUserFirestoreDAL
	permissionDAL permissionDALIface.IPermissionFirestoreDAL
	isProduction  bool
}

func NewService(conn *connection.Connection) Service {
	return &service{
		doitemployees.NewService(conn),
		userDal.NewUserFirestoreDALWithClient(conn.Firestore),
		permissionDAL.NewPermissionFirestoreDALWithClient(conn.Firestore),
		common.Production,
	}
}

func (s *service) AssertCacheEnableAccess(ctx *gin.Context) (*gin.Context, error) {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	if doitEmployee {
		isFlexsaveSuperAdmin, err := s.doitemployees.CheckDoiTEmployeeRole(ctx, flexsaveresold.FlexsaveSuperAdmin, email)
		if err != nil {
			return nil, err
		}

		if s.isProduction && !isFlexsaveSuperAdmin {
			return nil, web.NewRequestError(errors.New("doit employee not allowed to modify Flexsave"), http.StatusForbidden)
		}
	}

	customerID := ctx.Param("customerID")

	userID := ctx.GetString("userId")
	if userID == "" && !doitEmployee {
		return nil, web.NewRequestError(errors.New(http.StatusText(http.StatusUnauthorized)), http.StatusUnauthorized)
	}

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		"userId":               userID,
	})

	return ctx, nil
}

func (s *service) AssertCacheDisableAccess(ctx *gin.Context) (*gin.Context, error) {
	doitEmployee := ctx.GetBool("doitEmployee")
	email := ctx.GetString("email")

	isFlexsaveAdmin, err := s.doitemployees.CheckDoiTEmployeeRole(ctx, flexsaveresold.FlexsaveAdmin, email)
	if err != nil {
		return nil, err
	}

	if doitEmployee && !isFlexsaveAdmin {
		return nil, web.NewRequestError(errors.New("doit employee not allowed to modify Flexsave"), http.StatusForbidden)
	}

	return ctx, nil
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/user"
	"github.com/doitintl/hello/scheduled-tasks/user/service"
)

type Users struct {
	loggerProvider logger.Provider
	service        *service.UserService
}

func NewUsers(loggerProvider logger.Provider, conn *connection.Connection) *Users {
	userService := service.NewUserService(loggerProvider, conn.Firestore)

	return &Users{
		loggerProvider,
		userService,
	}
}

func (h *Users) DoitMigration(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	email := ctx.GetString("email")
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	if err := h.service.DoitMigration(ctx, email); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func Exists(ctx *gin.Context) error {
	user.Exists(ctx)

	return nil
}

func StartImpersonate(ctx *gin.Context) error {
	user.StartImpersonate(ctx)

	return nil
}

func StopImpersonate(ctx *gin.Context) error {
	user.StopImpersonate(ctx)

	return nil
}

func GenerateAPIToken(ctx *gin.Context) error {
	user.GenerateAPIToken(ctx)

	return nil
}

func DeleteAPIKey(ctx *gin.Context) error {
	user.DeleteAPIKey(ctx)

	return nil
}

func GetUIDByEmail(ctx *gin.Context) error {
	user.GetUIDByEmail(ctx)

	return nil
}

func AssignAllBillingProfiles(ctx *gin.Context) error {
	user.AssignAllBillingProfiles(ctx)

	return nil
}

func UpdateFSUserDisplayName(ctx *gin.Context) error {
	user.UpdateUserDisplayName(ctx)

	return nil
}

func Delete(ctx *gin.Context) error {
	user.Delete(ctx)

	return nil
}

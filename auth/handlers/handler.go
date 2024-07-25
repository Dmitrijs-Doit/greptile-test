package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/auth/service"
	"github.com/doitintl/hello/scheduled-tasks/auth/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ValidateResponse struct {
	Domain string `json:"domain"`
	Email  string `json:"email"`
}

type Auth struct {
	loggerProvider logger.Provider
	service        iface.AuthService
}

func NewAuth(log logger.Provider, conn *connection.Connection) *Auth {
	s := service.NewAuthService(log, conn)

	return &Auth{
		log,
		s,
	}
}

func (a *Auth) Validate(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString("email")

	customerDomain, err := a.service.Validate(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusForbidden)
	}

	res := ValidateResponse{
		customerDomain,
		email,
	}

	return web.Respond(ctx, res, http.StatusOK)
}

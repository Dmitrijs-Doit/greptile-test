package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/announcekit"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AnnouncekitHandler struct {
	loggerProvider logger.Provider
	service        *announcekit.AnnounceKitService
}

func NewAnnouncekitHandler(loggerProvider logger.Provider) *AnnouncekitHandler {
	service, err := announcekit.NewAnnounceKitService(loggerProvider)
	if err != nil {
		panic(err)
	}

	return &AnnouncekitHandler{
		loggerProvider,
		service,
	}
}

func (h *AnnouncekitHandler) CreateJwtToken(ctx *gin.Context) error {
	type response struct {
		Token string `json:"token"`
	}

	userClaims := announcekit.JwtUserClaims{
		ID:    ctx.GetString("userId"),
		EMAIL: ctx.GetString("email"),
		NAME:  ctx.GetString("name"),
	}

	token, err := h.service.CreateAuthToken(ctx, &userClaims)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response{Token: token}, http.StatusOK)
}

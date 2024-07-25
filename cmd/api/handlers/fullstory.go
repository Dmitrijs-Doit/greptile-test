package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/fullstory"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Fullstory struct {
	*logger.Logging
	service *fullstory.FullstoryService
}

func NewFullstory(log *logger.Logging, conn *connection.Connection) *Fullstory {
	service := fullstory.NewFullstoryService(log, conn)

	return &Fullstory{
		log,
		service,
	}
}

func (h *Fullstory) GetUserHMAC(ctx *gin.Context) error {
	email := ctx.GetString("email")
	claims := ctx.GetStringMap("claims")

	data, err := h.service.GetUserHMAC(ctx, email, claims)
	if err != nil {
		return web.NewRequestError(err, http.StatusUnauthorized)
	}

	return web.Respond(ctx, data, http.StatusOK)
}

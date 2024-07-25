package handlers

import (
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/csmengagement/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/gin-gonic/gin"
)

func (h *Handler) SendAttributionEmails(ctx *gin.Context) error {
	s := service.NewService(ctx, h.conn.Firestore(ctx), h.log(ctx))

	if err := s.SendNewAttributionEmails(ctx); err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

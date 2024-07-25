package handlers

import (
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/csmengagement/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"

	"github.com/gin-gonic/gin"
)

func (h *Handler) SendNoCustomerEngagementNotifications(ctx *gin.Context) error {
	log := h.log(ctx)
	s := service.NewService(ctx, h.conn.Firestore(ctx), log)

	if err := s.SendNoCustomerEngagementNotifications(ctx); err != nil {
		log.Error(err)
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

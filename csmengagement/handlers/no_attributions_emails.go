package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/csmengagement/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func (h *Handler) SendNoAttributionsEmails(ctx *gin.Context) error {
	s := service.NewService(ctx, h.conn.Firestore(ctx), h.log(ctx))

	emailsToSend, err := s.GetNoAttributionsEmails(ctx)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	err = s.SendNoAttributionsEmails(ctx, emailsToSend)
	if err != nil {
		return web.Respond(ctx, nil, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

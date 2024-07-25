package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/scripts"
)

type Scripts struct {
	scripts *scripts.Scripts
}

func NewScripts(log logger.Provider, conn *connection.Connection) *Scripts {
	s := scripts.NewScripts(log, conn)

	return &Scripts{
		scripts: s,
	}
}

func (s *Scripts) HandleScript(ctx *gin.Context) error {
	scriptName := ctx.Param("scriptName")

	if err := s.scripts.HandleScript(ctx, scriptName); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

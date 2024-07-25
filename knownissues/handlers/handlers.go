package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/knownissues"
	"github.com/doitintl/hello/scheduled-tasks/knownissues/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type KnownIssues struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        iface.Service
}

func NewKnownIssues(loggerProvider logger.Provider, conn *connection.Connection) *KnownIssues {
	return &KnownIssues{
		loggerProvider,
		conn,
		knownissues.NewService(loggerProvider, conn),
	}
}

func (h *KnownIssues) UpdateKnownIssues(ctx *gin.Context) error {
	if err := h.service.UpdateKnownIssues(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

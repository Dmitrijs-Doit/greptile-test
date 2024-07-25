package handlers

import (
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func NewHandler(log logger.Provider, conn *connection.Connection) *Handler {
	return &Handler{
		log,
		conn,
	}
}

type Handler struct {
	log  logger.Provider
	conn *connection.Connection
}

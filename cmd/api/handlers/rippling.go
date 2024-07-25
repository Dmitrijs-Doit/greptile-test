package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/rippling"
	"github.com/doitintl/hello/scheduled-tasks/rippling/iface"
)

type RipplingHandler struct {
	loggerProvider logger.Provider
	service        iface.IRippling
}

func NewRipplingHandler(loggerProvider logger.Provider, conn *connection.Connection) *RipplingHandler {
	service, err := rippling.NewRipplingService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	return &RipplingHandler{
		loggerProvider,
		service,
	}
}

// SyncAccountManagers - for each account manager (from rippling) update direct manager, create new document if does not exist
func (h *RipplingHandler) SyncAccountManagers(ctx *gin.Context) error {
	err := h.service.SyncAccountManagers(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// AddAccountManager - creates account manager document using payload taken from rippling
func (h *RipplingHandler) AddAccountManager(ctx *gin.Context) error {
	email := ctx.GetString("email")

	err := h.service.AddAccountManager(ctx, email)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// SyncFieldSalesManagerRole - sync doitRole with managers
func (h *RipplingHandler) SyncFieldSalesManagerRole(ctx *gin.Context) error {
	err := h.service.SyncFieldSalesManagerRole(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *RipplingHandler) GetTerminated(ctx *gin.Context) error {
	err := h.service.GetTerminated(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

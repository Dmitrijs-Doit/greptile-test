package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Fixer struct {
	logger.Provider
	service *fixer.FixerService
}

func NewFixer(log logger.Provider, conn *connection.Connection) *Fixer {
	service, err := fixer.NewFixerService(log, conn)
	if err != nil {
		panic(err)
	}

	return &Fixer{
		log,
		service,
	}
}

func (h *Fixer) SyncHandler(ctx *gin.Context) error {
	err := h.service.SyncCurrencyExchangeRateHistory(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

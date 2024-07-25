package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Pricebook struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        service.Pricebook
}

func NewPricebook(log logger.Provider, conn *connection.Connection) *Pricebook {
	svc := service.NewPricebook(log, conn)

	return &Pricebook{
		loggerProvider: log,
		conn:           conn,
		service:        svc,
	}
}

func (h *Pricebook) SetEditionPricebook(ctx *gin.Context) error {
	err := h.service.SetEditionPrices(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	// TODO(CMP-21119): Retire this code when customers stop using these SKUs.
	err = h.service.SetLegacyFlatRatePrices(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

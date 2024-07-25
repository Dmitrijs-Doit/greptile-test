package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/hubspot"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Hubspot struct {
	*logger.Logging
	service *hubspot.HubspotService
}

func NewHubspot(log *logger.Logging, conn *connection.Connection) *Hubspot {
	service, err := hubspot.NewHubspotService(log, conn)
	if err != nil {
		panic(err)
	}

	return &Hubspot{
		log,
		service,
	}
}

func (h *Hubspot) SyncHandler(ctx *gin.Context) error {
	err := h.service.Sync(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Hubspot) SyncCompanyHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	err := h.service.SyncCompanyWorker(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Hubspot) SyncContactsHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	err := h.service.SyncContactsWorker(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

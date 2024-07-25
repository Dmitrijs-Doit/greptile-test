package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/salesforce/service"
	"github.com/doitintl/hello/scheduled-tasks/salesforce/sync"
	"github.com/doitintl/hello/scheduled-tasks/salesforce/sync/customer"
)

type SalesforceHandler struct {
	logging          *logger.Logging
	compositeService *service.CompositeService
	syncService      *sync.Service
	companiesService *customer.Service
}

func NewSalesforce(log *logger.Logging, conn *connection.Connection) *SalesforceHandler {
	sfComposite, err := service.NewCompositeService(log, nil, nil)

	if err != nil {
		panic(err)
	}

	syncService, err := sync.NewService(log, conn)
	if err != nil {
		panic(err)
	}

	return &SalesforceHandler{
		log,
		sfComposite,
		syncService,
		customer.NewService(log, conn),
	}
}

func (h *SalesforceHandler) CompositeRequestHandler(ctx *gin.Context) error {
	var body = service.CompositeRequest{}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.CompositeRequest) == 0 {
		return web.NewRequestError(errors.New("there is no payload to the composite request"), http.StatusBadRequest)
	}

	for i, c := range body.CompositeRequest {
		if c.URL == "" {
			return web.NewRequestError(fmt.Errorf("composite request %d does not have URL attribute", i), http.StatusBadRequest)
		}

		if c.Method == "" {
			return web.NewRequestError(fmt.Errorf("composite request %d does not have Method attribute", i), http.StatusBadRequest)
		}
	}

	resp, err := h.compositeService.CompositeRequest(ctx, body)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	for _, r := range resp.CompositeResponse {
		if r.HTTPStatusCode == 400 {
			s, _ := json.MarshalIndent(resp, "", " ")
			return web.NewRequestError(fmt.Errorf("composite request failed, response body: %s", string(s)), http.StatusBadRequest)
		}
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *SalesforceHandler) SyncHandler(ctx *gin.Context) error {
	err := h.syncService.Sync(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *SalesforceHandler) SyncCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	err := h.companiesService.SyncCompany(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

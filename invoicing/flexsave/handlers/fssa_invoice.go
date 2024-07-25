package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/domain/billing"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/service"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FlexsaveInvoicing struct {
	loggerProvider       logger.Provider
	fssaInvoicingService iface.FlexsaveStandalone
}

func NewFSSAInvoiceData(loggerProvider logger.Provider, conn *connection.Connection) *FlexsaveInvoicing {
	service, err := service.NewFlexsaveInvoiceService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	return &FlexsaveInvoicing{
		logger.FromContext,
		service,
	}
}

func (h *FlexsaveInvoicing) UpdateFlexsaveInvoicingData(ctx *gin.Context) error {
	var body billing.InvoicingTask
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if body.Provider != common.Assets.AmazonWebServicesStandalone && body.Provider != common.Assets.GoogleCloudStandalone {
		return web.NewRequestError(errors.New("invalid provider"), http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginInvoicingAws)

	err := h.fssaInvoicingService.UpdateFlexsaveInvoicingData(ctx, body.InvoiceMonth, body.Provider)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveInvoicing) UpdateFlexsaveInvoicingDataWorker(ctx *gin.Context) error {
	var body billing.InvoicingTask
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if body.Provider != common.Assets.AmazonWebServicesStandalone && body.Provider != common.Assets.GoogleCloudStandalone {
		return web.NewRequestError(errors.New("invalid provider"), http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginInvoicingAws)

	err := h.fssaInvoicingService.FlexsaveDataWorker(ctx, ctx.Param("customerID"), body.InvoiceMonth, body.Provider)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

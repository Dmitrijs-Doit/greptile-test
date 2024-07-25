package handlers

import (
	"net/http"
	"strconv"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type PLES struct {
	loggerProvider logger.Provider
	service        service.PLESIface
	billingData    invoicing.BillingData
}

func NewPLES(log logger.Provider, conn *connection.Connection) *PLES {
	s := service.NewPLESService(log, conn)

	return &PLES{
		log,
		s,
		invoicing.NewBillingDataService(conn),
	}
}

func (h *PLES) UpdatePLES(ctx *gin.Context) error {
	file, _, err := ctx.Request.FormFile("ples_accounts")
	if err != nil {
		return err
	}

	defer file.Close()

	invoiceMonth := ctx.Request.FormValue("invoice_month")

	if err := h.validateInvoiceMonth(ctx, invoiceMonth); err != nil {
		return web.Respond(ctx, []string{err.Error()}, http.StatusBadRequest)
	}

	forceUpdate, err := strconv.ParseBool(ctx.Request.FormValue("force_update"))
	if err != nil {
		forceUpdate = false // default value
	}

	updatePLESRequest, errs := parsePLESFile(file, invoiceMonth)
	if len(errs) > 0 {
		var errStrings []string

		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}

		return web.Respond(ctx, errStrings, http.StatusBadRequest)
	}

	if errs := h.service.UpdatePLESAccounts(ctx, updatePLESRequest, forceUpdate); len(errs) > 0 {
		var errStrings []string

		for _, err := range errs {
			errStrings = append(errStrings, err.Error())
		}

		return web.Respond(ctx, errStrings, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

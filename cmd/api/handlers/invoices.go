package handlers

import (
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/gin-gonic/gin"
)

func InvoicesMainHandler(ctx *gin.Context) error {
	invoices.MainHandler(ctx)

	return nil
}

func InvoicesCustomerWorker(ctx *gin.Context) error {
	invoices.CustomerWorker(ctx)

	return nil
}

func NotificationsHandler(ctx *gin.Context) error {
	invoices.NotificationsHandler(ctx)

	return nil
}

func NotificationsWorker(ctx *gin.Context) error {
	invoices.NotificationsWorker(ctx)

	return nil
}

func NoticeToRemedy(ctx *gin.Context) error {
	invoices.NoticeToRemedy(ctx)

	return nil
}

func CustomerHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return ctx.AbortWithError(http.StatusBadRequest, nil)
	}

	invoices.CustomerHandler(ctx, customerID)

	return web.Respond(ctx, nil, http.StatusOK)
}

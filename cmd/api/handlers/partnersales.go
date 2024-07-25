package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud/partnersales"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type PartnerSales struct {
	loggerProvider logger.Provider
	service        *partnersales.GoogleChannelService
}

func NewPartnerSales(loggerProvider logger.Provider, conn *connection.Connection) *PartnerSales {
	service, err := partnersales.NewGoogleChannelService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	return &PartnerSales{
		loggerProvider,
		service,
	}
}

func (h *PartnerSales) SyncCustomersHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	if errs := h.service.SyncCustomers(ctx); len(errs) > 0 {
		for err := range errs {
			l.Error(err)
		}

		return web.NewRequestError(errs[0], h.httpCode(errs[0]))
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// CreateBillingAccountHandler creates new billing account in Partner Sales Console
// using Channel API
func (h *PartnerSales) CreateBillingAccountHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var body googlecloud.CreateBillingAccountBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, h.httpCode(err))
	}

	l.Info(body)

	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")
	email := ctx.GetString("email")

	data := partnersales.CreateBillingAccountRequestData{
		CustomerID: customerID,
		EntityID:   entityID,
		Email:      email,
		ReqBody:    &body,
	}

	respData, err := h.service.CreateBillingAccount(ctx, &data)
	if err != nil {
		return web.NewRequestError(err, h.httpCode(err))
	}

	return web.Respond(ctx, respData, http.StatusOK)
}

func (h *PartnerSales) BillingAccountsListHandler(ctx *gin.Context) error {
	if err := h.service.BillingAccountsList(ctx); err != nil {
		return web.NewRequestError(err, h.httpCode(err))
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *PartnerSales) httpCode(err error) int {
	switch status.Code(err) {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

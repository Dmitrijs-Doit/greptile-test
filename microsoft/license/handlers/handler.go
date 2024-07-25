package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/service"
)

type License struct {
	log     *logger.Logging
	service service.ILicenseService
}

func NewLicenseHandler(log *logger.Logging, conn *connection.Connection) *License {
	s, err := service.NewLicenseService(log, conn)
	if err != nil {
		panic(err)
	}

	return &License{
		service: s,
		log:     log,
	}
}

func (h *License) LicenseChangeQuantityHandler(ctx *gin.Context) error {
	l := h.log.Logger(ctx)

	licenseCustomerID := ctx.Param("licenseCustomerID")
	subscriptionID := ctx.Param("subscriptionID")
	email := ctx.GetString("email")

	doitEmployee := ctx.GetBool("doitEmployee")
	claims := ctx.GetStringMap("claims")

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: licenseCustomerID,
		"subscriptionId":       subscriptionID,
	})

	l.Infof("calling LicenseChangeQuantityHandler customer ID %s and subscription ID %s", licenseCustomerID, subscriptionID)

	if licenseCustomerID == "" {
		return web.NewRequestError(service.ErrMissingCustomerID, http.StatusBadRequest)
	}

	if subscriptionID == "" {
		return web.NewRequestError(service.ErrMissingSubscriptionID, http.StatusBadRequest)
	}

	var requestBody service.ChangeSeatsRequest

	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	l.Infof("change microsoft subscription request: %+v ", requestBody)

	if requestBody.AssetType != "office-365" {
		return web.NewRequestError(service.ErrWrongAssetType, http.StatusBadRequest)
	}

	if requestBody.Quantity < 0 {
		return web.NewRequestError(service.ErrWrongQuantity, http.StatusBadRequest)
	}

	props := &service.ChangeQuantityProps{
		Email:             email,
		DoitEmployee:      doitEmployee,
		SubscriptionID:    subscriptionID,
		LicenseCustomerID: licenseCustomerID,
		RequestBody:       requestBody,
		Claims:            claims,
		EnableLog:         true,
	}
	httpResponse, err := h.service.ChangeQuantity(ctx, props)

	if err != nil {
		return web.NewRequestError(err, httpResponse)
	}

	return web.Respond(ctx, nil, httpResponse)
}

func (h *License) LicenseOrderHandler(ctx *gin.Context) error {
	var err error

	var email string

	var doitEmployee bool

	customerID := ctx.Param("customerID")

	if customerID == "" {
		return web.NewRequestError(service.ErrMissingCustomerID, http.StatusBadRequest)
	}

	var requestBody service.SubscriptionsOrderRequest
	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	log.Printf("%+v", requestBody)

	if requestBody.Quantity < 0 {
		return web.NewRequestError(service.ErrQuantity, http.StatusBadRequest)
	}

	claims := ctx.GetStringMap("claims")

	val, ok := claims["email"]

	if !ok {
		//FIX
		return web.NewRequestError(service.ErrBadRequest, http.StatusBadRequest)
	}

	email, ok = val.(string)
	if !ok {
		return web.NewRequestError(service.ErrBadRequest, http.StatusBadRequest)
	}

	val, ok = claims["doitEmployee"]
	if !ok {
		return web.NewRequestError(service.ErrBadRequest, http.StatusBadRequest)
	}

	doitEmployee, ok = val.(bool)

	if !ok {
		return web.NewRequestError(service.ErrBadRequest, http.StatusBadRequest)
	}

	props := &service.CreateOrderProps{
		CustomerID:   customerID,
		Email:        email,
		DoitEmployee: doitEmployee,
		RequestBody:  requestBody,
		Claims:       claims,
		EnableLog:    true,
	}

	err = h.service.CreateOrder(ctx, props)

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *License) LicenseSyncHandler(ctx *gin.Context) error {
	var err error

	customerID := ctx.Param("customerID")

	if customerID == "" {
		return web.NewRequestError(service.ErrMissingCustomerID, http.StatusBadRequest)
	}

	var requestBody microsoft.SubscriptionSyncRequest

	if err := ctx.ShouldBindJSON(&requestBody); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err = h.service.SyncQuantity(ctx, requestBody)

	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

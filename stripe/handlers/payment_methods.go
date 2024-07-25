package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

// GetPaymentMethodsHandler fetches all billing profiles payment methods
func (h *Stripe) GetPaymentMethodsHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return err
	}

	if err := h.validatePaymentMethodRequest(ctx, customerID, entity); err != nil {
		return err
	}

	result, err := h.GetStripeAccountByCurrency(entity.Currency).service.GetPaymentMethods(ctx, entity)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, result, http.StatusOK)
}

// PatchPaymentMethodHandler updates a billing profile payment method
func (h *Stripe) PatchPaymentMethodHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var input service.PaymentMethodBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	l.Infoln(input)

	if input.PaymentType == common.EntityPaymentTypeCard && input.PaymentMethodID == "" {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return err
	}

	if err := h.validatePaymentMethodRequest(ctx, customerID, entity); err != nil {
		return err
	}

	input.Email = ctx.GetString("email")
	input.Name = ctx.GetString("name")
	input.CustomerID = customerID
	input.EntityID = entityID

	if err := h.GetStripeAccountByCurrency(entity.Currency).service.PatchPaymentMethod(ctx, input, entity); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// DetachPaymentMethodHandler deletes a payment method from a billing profile
func (h *Stripe) DetachPaymentMethodHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var input service.PaymentMethodBody
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	l.Infoln(input)

	if input.PaymentMethodID == "" {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return err
	}

	if err := h.validatePaymentMethodRequest(ctx, customerID, entity); err != nil {
		return err
	}

	input.Email = ctx.GetString("email")
	input.Name = ctx.GetString("name")
	input.CustomerID = customerID
	input.EntityID = entityID

	if err := h.GetStripeAccountByCurrency(entity.Currency).service.DetachPaymentMethod(ctx, input, entity); err != nil {
		if errors.Is(err, service.ErrDetachPaymentMethod) {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Stripe) validatePaymentMethodRequest(ctx *gin.Context, customerID string, entity *common.Entity) error {
	if customerID == "" {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	if !ctx.GetBool(common.DoitEmployee) {
		userID := ctx.GetString("userId")

		ok, err := h.GetStripeAccountByCurrency(entity.Currency).service.ValidateUserPermissions(ctx, customerID, userID)
		if err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}

		if !ok {
			return web.NewRequestError(web.ErrAuthenticationFailure, http.StatusUnauthorized)
		}
	}

	return nil
}

type SetupHandlerBody struct {
	NewEntity bool `json:"newEntity"`
}

// CreateSetupIntentHandler creates a payment method setup intent, returns client secret
func (h *Stripe) CreateSetupIntentHandler(ctx *gin.Context) error {
	entityID := ctx.Param("entityID")

	var setupHandlerBody SetupHandlerBody
	if err := ctx.ShouldBindJSON(&setupHandlerBody); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	response, err := h.GetStripeAccountByCurrency(entity.Currency).service.CreatePMSetupIntentForEntity(ctx, entity, setupHandlerBody.NewEntity)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) || errors.Is(err, service.ErrInvalidPaymentMethodType) {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(service.ErrCreateSetupIntent, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *Stripe) CreateSetupSessionHandler(ctx *gin.Context) error {
	entityID := ctx.Param("entityID")

	var body service.SetupSessionURLs
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	sessionURL, err := h.GetStripeAccountByCurrency(entity.Currency).service.CreateSetupSessionForEntity(ctx, entity, body)
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) || errors.Is(err, service.ErrInvalidPaymentMethodType) {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(service.ErrCreateSession, http.StatusInternalServerError)
	}

	return web.Respond(ctx, sessionURL, http.StatusOK)
}

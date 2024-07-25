package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

// PaymentsDigestHandler sends a digest email of payments from the last day
func (h *Stripe) PaymentsDigestHandler(ctx *gin.Context) error {
	for _, account := range h.Accounts() {
		if err := account.service.PaymentsDigest(ctx); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// AutomaticPaymentsHandler create payments for invoices on their due date
func (h *Stripe) AutomaticPaymentsHandler(ctx *gin.Context) error {
	var input service.AutomaticPaymentsInput

	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	for _, account := range h.Accounts() {
		if err := account.service.AutomaticPayments(ctx, input); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// AutomaticPaymentsEntityWorker create payments for invoices on their due date
func (h *Stripe) AutomaticPaymentsEntityWorker(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var input service.AutomaticPaymentsEntityWorkerInput

	if err := ctx.ShouldBindJSON(&input); err != nil {
		l.Errorf("failed to bind input with error: %s", err)
		return web.RespondError(ctx, web.NewRequestError(err, http.StatusOK))
	}

	stripeAccount, err := h.GetStripeAccount(input.StripeAccountID)
	if err != nil {
		l.Errorf("failed to get stripe account %s with error: %s", input.StripeAccountID, err)
		return web.RespondError(ctx, web.NewRequestError(err, http.StatusOK))
	}

	if stripeAccount == nil {
		err := fmt.Errorf("stripe account id %s not found", input.StripeAccountID)
		l.Error(err)

		return web.RespondError(ctx, web.NewRequestError(err, http.StatusOK))
	}

	if err := stripeAccount.service.AutomaticPaymentsEntityWorker(ctx, input); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// PayInvoiceHandler manually pay an invoice
func (h *Stripe) PayInvoiceHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var input service.PayInvoiceInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	l.Infoln(input)

	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")
	invoiceID := ctx.Param("invoiceID")

	if invoiceID == "" {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return err
	}

	if err := h.validatePaymentMethodRequest(ctx, customerID, entity); err != nil {
		return err
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEntityID:   entityID,
		"invoiceId":            invoiceID,
	})

	input.UserID = ctx.GetString("userId")
	input.Email = ctx.GetString("email")
	input.CustomerID = customerID
	input.EntityID = entityID
	input.InvoiceID = invoiceID

	if err := h.GetStripeAccountByCurrency(entity.Currency).service.PayInvoice(ctx, input, entity); err != nil {
		l.Errorf("invoice payment failed with error: %s", err)

		if errors.Is(err, service.ErrPaymentMethodDisabled) {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		stripeErr, ok := err.(*stripe.Error)
		if ok {
			return web.NewRequestError(errors.New(stripeErr.Msg), stripeErr.HTTPStatusCode)
		}

		return web.NewRequestError(errors.New("payment failed"), http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Stripe) GetCreditCardProcessingFee(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")

	var input service.ProcessingFeeInput
	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEntityID:   entityID,
		"amount":               strconv.FormatInt(input.Amount, 10),
	})

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return err
	}

	processingFee, err := h.GetStripeAccountByCurrency(entity.Currency).service.GetCreditCardProcessingFee(ctx, customerID, entity, input.Amount)
	if err != nil {
		l.Errorf("get credit card processing fee failed with error: %s", err)
		return web.NewRequestError(ErrGetCreditCardProcessingFee, http.StatusInternalServerError)
	}

	return web.Respond(ctx, processingFee, http.StatusOK)
}

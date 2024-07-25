package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

// WebhookHandler handles events from stripe
func (h *Stripe) WebhookHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	acct := ctx.Request.Header.Get("Stripe-Account")
	l.Debugf("Stripe-Account header: %s", acct)

	signature := ctx.Request.Header.Get("Stripe-Signature")
	if signature == "" {
		return web.NewRequestError(web.ErrBadRequest, http.StatusBadRequest)
	}

	apiVersion := ctx.Query("api_version")
	account := ctx.Query("account")

	l.SetLabels(map[string]string{
		"apiVersion": apiVersion,
		"account":    account,
	})

	var webhookService *service.StripeWebhookService

	switch account {
	case string(domain.StripeAccountUKandI):
		webhookService = h.stripeUKandI.webhookService
	case string(domain.StripeAccountUS):
		webhookService = h.stripeUS.webhookService
	case string(domain.StripeAccountDE):
		webhookService = h.stripeDE.webhookService
	default:
		return web.NewRequestError(fmt.Errorf("Invalid account: %s", account), http.StatusBadRequest)
	}

	if err := webhookService.HandleEvent(ctx, body, signature, apiVersion); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service"
)

type StandaloneApprovePayload struct {
	CustomerID       string `json:"customerId"`
	BillingAccountID string `json:"billingAccountId"`
}

func (h *MarketplaceGCP) StandaloneApprove(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	var payload StandaloneApprovePayload

	if err := ctx.ShouldBindJSON(&payload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if payload.CustomerID == "" || payload.BillingAccountID == "" {
		return web.NewRequestError(ErrInvalidPayload, http.StatusBadRequest)
	}

	l.SetLabels(map[string]string{
		logger.LabelCustomerID:  payload.CustomerID,
		"billingAccountId": payload.BillingAccountID,
	})

	err := h.service.StandaloneApprove(ctx, payload.CustomerID, payload.BillingAccountID)
	if err != nil {
		switch err {
		case service.ErrBillingAccountMismatch, service.ErrCustomerNotStandalone:
			return web.NewRequestError(err, http.StatusUnprocessableEntity)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

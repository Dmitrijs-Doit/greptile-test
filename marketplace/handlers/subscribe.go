package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service"
)

type HTTPError struct {
	Error      string     `json:"error"`
	StatusCode StatusCode `json:"statusCode"`
}

type StatusCode string

const (
	StatusCodeCustomerIsNotEligibleFlexsave StatusCode = "100"
)

func (h *MarketplaceGCP) Subscribe(ctx *gin.Context) error {
	var subscribePayload domain.SubscribePayload

	if err := ctx.ShouldBindJSON(&subscribePayload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.Subscribe(ctx, subscribePayload)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCustomerAlreadySubscribed):
			return web.Respond(ctx, nil, http.StatusOK)
		case errors.Is(err, service.ErrFlexsaveProductIsDisabled):
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusCreated)
}

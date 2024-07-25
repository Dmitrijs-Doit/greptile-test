package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func (h *Stripe) SyncCustomerData(ctx *gin.Context) error {
	entityID := ctx.Param("entityID")

	entity, err := h.entitiesDAL.GetEntity(ctx, entityID)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.GetStripeAccountByCurrency(entity.Currency).service.SyncCustomerData(ctx, entity); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

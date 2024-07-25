package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/validator"
)

func (h *MarketplaceGCP) HandleCmpEvent(ctx *gin.Context) error {
	log := h.loggerProvider(ctx)

	var m domain.PubSubMessage

	if err := ctx.ShouldBindJSON(&m); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	log.Infof("original data message: %s", string(m.Message.Data))

	var cmpBaseEvent domain.CmpInboundBaseEvent

	if err := validator.UnmarshalJSON(m.Message.Data, &cmpBaseEvent); err != nil {
		log.Infof("error unmarshalling cmp base event %+v, data: %s", err, m.Message.Data)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	switch cmpBaseEvent.EventType {
	case domain.CmpInboundEventTypeEntitlementApproveRequested:
		if err := h.handleCmpEntitlementApproveRequestedEvent(ctx, m); err != nil {
			if err == ErrUnmarshalCmpEntitlementApproveRequestedEvent {
				return web.NewRequestError(err, http.StatusBadRequest)
			}

			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	case domain.CmpInboundEventTypeEntitlementCancelled:
		if err := h.handleCmpEntitlementCancelledEvent(ctx, m); err != nil {
			if err == ErrUnmarshalCmpEntitlementCancelledEvent {
				return web.NewRequestError(err, http.StatusBadRequest)
			}

			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	default:
		log.Errorf("unknown cmp base event type: %s", cmpBaseEvent.EventType)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *MarketplaceGCP) handleCmpEntitlementApproveRequestedEvent(ctx context.Context, m domain.PubSubMessage) error {
	log := h.loggerProvider(ctx)

	var event domain.CmpEntitlementApproveRequestedEvent

	if err := validator.UnmarshalJSON(m.Message.Data, &event); err != nil {
		log.Errorf("error unmarshalling cmp entitlement approve event: %s, data: %s", err, m.Message.Data)

		return ErrUnmarshalCmpEntitlementApproveRequestedEvent
	}

	entitlementID := event.Entitlement.ID
	if err := h.service.ApproveEntitlement(
		ctx,
		entitlementID,
		"",
		false,
		true,
	); err != nil {
		return err
	}

	return nil
}

func (h *MarketplaceGCP) handleCmpEntitlementCancelledEvent(ctx context.Context, m domain.PubSubMessage) error {
	log := h.loggerProvider(ctx)

	var event domain.CmpEntitlementCancelledEvent

	if err := validator.UnmarshalJSON(m.Message.Data, &event); err != nil {
		log.Errorf("error unmarshalling cmp entitlement cancelled event: %s, data: %s", err, m.Message.Data)

		return ErrUnmarshalCmpEntitlementCancelledEvent
	}

	entitlementID := event.Entitlement.ID

	if err := h.service.HandleCancelledEntitlement(ctx, entitlementID); err != nil {
		return err
	}

	log.Infof("cancelled entitlementID: %s", entitlementID)

	return nil
}

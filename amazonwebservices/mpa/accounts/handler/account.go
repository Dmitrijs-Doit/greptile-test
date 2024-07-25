package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AccountHandler struct {
	notifier iface.Notifier
	*logger.Logging
}

func NewAccountHandler(
	notifier iface.Notifier,
	logger *logger.Logging,
) *AccountHandler {
	return &AccountHandler{
		notifier,
		logger,
	}
}

func (h *AccountHandler) HandleNotifyAccountUpdate(ctx *gin.Context) error {
	data, err := extractDataFromMessage(ctx.Request.Body)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	var changes domain.AccountChanges
	if err = json.Unmarshal(data, &changes); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err = h.notifier.NotifyIfNecessary(ctx, convert(changes), h.eventType(ctx, changes)); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func convert(changes domain.AccountChanges) iface.AccountMove {
	before := changes.Before.Payer
	after := changes.After.Payer

	fromPayer := iface.NewPayer(before.ID, before.DisplayName)
	toPayer := iface.NewPayer(after.ID, after.DisplayName)

	return iface.NewAccountMove(changes.Before.ID, changes.Before.Name, fromPayer, toPayer)
}

func (h *AccountHandler) eventType(ctx *gin.Context, changes domain.AccountChanges) iface.EventType {
	logger := h.Logger(ctx)

	if changes.After == (domain.Account{}) {
		logger.Infof("Account deleted: %+v", changes.Before)
		return iface.LeftAccount
	}

	logger.Infof("Account changes: %+v", changes)

	return iface.MovedAccount
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zapier/service"
)

type WebhookHandler struct {
	l   logger.Provider
	svc service.WebhookSubscriptionService
}

func NewWebhookHandler(l logger.Provider, conn *connection.Connection) *WebhookHandler {
	svc := service.NewWebhookSubscriptionService(l, conn)

	return &WebhookHandler{
		l:   l,
		svc: svc,
	}
}

func (ws *WebhookHandler) Create(ctx *gin.Context) error {
	var req service.CreateWebhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	req.CustomerID = ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	req.UserID = ctx.GetString(common.CtxKeys.UserID)
	req.UserEmail = ctx.GetString(common.CtxKeys.Email)

	resp, err := ws.svc.CreateSubscription(ctx, &req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusCreated)
}

func (ws *WebhookHandler) Delete(ctx *gin.Context) error {
	var req service.DeleteWebhookRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	req.CustomerID = ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	req.UserEmail = ctx.GetString(common.CtxKeys.Email)
	req.UserID = ctx.GetString(common.CtxKeys.UserID)

	err := ws.svc.DeleteSubscription(ctx, &req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (ws *WebhookHandler) GetAlertsMock(ctx *gin.Context) error {
	return web.Respond(ctx, ws.svc.GetAlertsMock(), http.StatusOK)
}

func (ws *WebhookHandler) GetBudgetsMock(ctx *gin.Context) error {
	return web.Respond(ctx, ws.svc.GetBudgetsMock(), http.StatusOK)
}

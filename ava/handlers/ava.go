package handlers

import (
	"context"
	"net/http"

	avaservice "github.com/doitintl/hello/scheduled-tasks/ava"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type AvaEmbeddingsHandler struct {
	loggerProvider logger.Provider
	avaService     *avaservice.Service
}

func NewAvaEmbeddingsHandler(ctx context.Context, log logger.Provider, conn *connection.Connection) *AvaEmbeddingsHandler {
	return &AvaEmbeddingsHandler{
		log,
		avaservice.NewAvaService(ctx, log, conn),
	}
}

func (h *AvaEmbeddingsHandler) UpsertFirestoreDocumentEmbeddings(ctx *gin.Context) error {
	var req avaservice.Request
	if err := ctx.BindJSON(&req); err != nil {
		return web.Respond(ctx, err, http.StatusBadRequest)
	}

	if err := h.avaService.PopulateCustomerFilterValues(ctx, req.CustomerID); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AvaEmbeddingsHandler) UpsertPresetAttributionsEmbeddings(ctx *gin.Context) error {
	if err := h.avaService.PopulateAttributions(ctx); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AvaEmbeddingsHandler) CreateMetadataTaskHandler(ctx *gin.Context) error {
	var req avaservice.Request
	if err := ctx.BindJSON(&req); err != nil {
		return web.Respond(ctx, err, http.StatusBadRequest)
	}

	if !h.avaService.IsAllowToUpdate(ctx, req.CustomerID) {
		return web.Respond(ctx, nil, http.StatusConflict)
	}

	if err := h.avaService.CreateMetadataTask(ctx, req.CustomerID); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

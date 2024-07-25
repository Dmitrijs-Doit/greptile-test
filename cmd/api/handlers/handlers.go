package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/handlers"
)

func FirestoreExportHandler(ctx *gin.Context) error {
	if err := handlers.FirestoreExportHandler(ctx); err != nil {
		return err
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func FirestoreImportBigQueryHandler(ctx *gin.Context) error {
	if err := handlers.FirestoreImportBigQueryHandler(ctx); err != nil {
		return err
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

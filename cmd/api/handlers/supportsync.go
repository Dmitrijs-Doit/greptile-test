package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/supportsync"
)

type SupportSync struct {
	loggerProvider logger.Provider
	service        *supportsync.SupportSyncService
}

// NewSupportSync handler
func NewSupportSync(log logger.Provider, conn *connection.Connection) *SupportSync {
	service, err := supportsync.NewSupportSyncService(log, conn.Firestore, conn.CloudStorageClient)
	if err != nil {
		panic(err)
	}

	return &SupportSync{
		log,
		service,
	}
}

// Sync syncs services from gcs to firestore
func (h *SupportSync) Sync(ctx *gin.Context) error {
	if err := h.service.Sync(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

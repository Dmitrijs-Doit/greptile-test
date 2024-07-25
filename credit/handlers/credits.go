package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/credit/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Credits cloud analytics handlers
type Credits struct {
	logger.Provider
	service *service.CreditsService
}

// NewCredits init new Credits handlers
func NewCredits(loggerProvider logger.Provider, conn *connection.Connection) *Credits {
	service, err := service.NewCreditsService(loggerProvider, conn.Firestore, conn.Bigquery)
	if err != nil {
		panic(err)
	}

	return &Credits{
		loggerProvider,
		service,
	}
}

func (h *Credits) UpdateCustomerCreditsTable(ctx *gin.Context) error {
	err := h.service.LoadCreditsToBQ(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

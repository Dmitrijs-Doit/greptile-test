package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/service"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/billing-explainer/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BillingExplainerHandler struct {
	loggerProvider logger.Provider
	service        serviceIface.BillingExplainerService
}

func NewBillingExplainerHandler(log logger.Provider, conn *connection.Connection) *BillingExplainerHandler {
	s := service.NewBillingExplainerService(log, conn)

	return &BillingExplainerHandler{
		log,
		s,
	}
}

func (h *BillingExplainerHandler) GetDataFromBigQueryAndStoreInFirestore(ctx *gin.Context) error {
	var billingExplainerInput domain.BillingExplainerInputStruct

	if err := ctx.BindJSON(&billingExplainerInput); err != nil {
		return web.Respond(ctx, "Invalid request data", http.StatusBadRequest)
	}

	validate := validator.New()

	if err := validate.Struct(billingExplainerInput); err != nil {
		return web.Respond(ctx, "Missing required fields", http.StatusBadRequest)
	}

	err := h.service.GetBillingExplainerSummaryAndStoreInFS(ctx, billingExplainerInput.CustomerID, billingExplainerInput.BillingMonth, billingExplainerInput.EntityID, billingExplainerInput.IsBackfill)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Data transferred successfully", http.StatusOK)
}

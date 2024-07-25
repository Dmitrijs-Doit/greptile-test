package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	reportsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/contract/service"
	contractIface "github.com/doitintl/hello/scheduled-tasks/contract/service/iface"
	customersDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const invalidRequestDataErr = "Invalid request data"

type ContractHandler struct {
	loggerProvider logger.Provider
	service        contractIface.ContractService
}

func NewContractHandler(log logger.Provider, conn *connection.Connection) *ContractHandler {
	reportDAL := reportsDAL.NewReportsFirestoreWithClient(conn.Firestore)

	customerDAL := customersDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	cloudAnalyticsService, err := cloudanalytics.NewCloudAnalyticsService(log, conn, reportDAL, customerDAL)
	if err != nil {
		panic(err)
	}

	s := service.NewContractService(log, conn, cloudAnalyticsService)

	return &ContractHandler{
		log,
		s,
	}
}

func (h *ContractHandler) AddContract(ctx *gin.Context) error {
	var ContractInput domain.ContractInputStruct

	if err := ctx.BindJSON(&ContractInput); err != nil {
		return web.Respond(ctx, invalidRequestDataErr, http.StatusBadRequest)
	}

	validate := validator.New()

	if err := validate.Struct(ContractInput); err != nil {
		return web.Respond(ctx, "Missing required fields", http.StatusBadRequest)
	}

	err := h.service.CreateContract(ctx, ContractInput)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Contract created successfully", http.StatusOK)
}

func (h *ContractHandler) CancelContract(ctx *gin.Context) error {
	contractID := ctx.Param("id")

	if err := h.service.CancelContract(ctx, contractID); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Contract canceled successfully", http.StatusOK)
}

func (h *ContractHandler) InternalCancelContract(ctx *gin.Context) error {
	contractID := ctx.Param("id")

	err := h.service.CancelContract(ctx, contractID)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Contract canceled successfully", http.StatusOK)
}

func (h *ContractHandler) AggregateAllInvoiceData(ctx *gin.Context) error {
	var AggregatedInvoiceInput domain.AggregatedInvoiceInputStruct

	if err := ctx.BindJSON(&AggregatedInvoiceInput); err != nil {
		return web.Respond(ctx, invalidRequestDataErr, http.StatusBadRequest)
	}

	err := h.service.AggregateInvoiceData(ctx, AggregatedInvoiceInput.InvoiceMonth, "")
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Aggregated invoice data successfully", http.StatusOK)
}

func (h *ContractHandler) AggregateInvoiceData(ctx *gin.Context) error {
	contractID := ctx.Param("id")

	var AggregatedInvoiceInput domain.AggregatedInvoiceInputStruct

	if err := ctx.BindJSON(&AggregatedInvoiceInput); err != nil {
		return web.Respond(ctx, invalidRequestDataErr, http.StatusBadRequest)
	}

	err := h.service.AggregateInvoiceData(ctx, AggregatedInvoiceInput.InvoiceMonth, contractID)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Aggregated invoice data successfully", http.StatusOK)
}

func (h *ContractHandler) Refresh(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	err := h.service.RefreshCustomerTiers(ctx, customerID)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Customer contracts and tiers refreshed succesfully", http.StatusOK)
}

func (h *ContractHandler) RefreshAll(ctx *gin.Context) error {
	err := h.service.RefreshAllCustomerTiers(ctx)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Customer contracts and tiers refreshed succesfully", http.StatusOK)
}

func (h *ContractHandler) ExportContracts(ctx *gin.Context) error {
	if err := h.service.ExportContracts(ctx); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Contracts exported successfully", http.StatusOK)
}

func (h *ContractHandler) UpdateContract(ctx *gin.Context) error {
	contractID := ctx.Param("id")

	var ContractInput domain.ContractUpdateInputStruct

	email := ctx.GetString("email")

	userName := ctx.GetString("name")

	if err := ctx.BindJSON(&ContractInput); err != nil {
		return web.Respond(ctx, invalidRequestDataErr, http.StatusBadRequest)
	}

	err := h.service.UpdateContract(ctx, contractID, ContractInput, email, userName)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Contract updated successfully", http.StatusOK)
}

func (h *ContractHandler) UpdateGoogleCloudContractsSupport(ctx *gin.Context) error {
	err := h.service.UpdateGoogleCloudContractsSupport(ctx)
	if err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "UpdateGoogleCloudContractsSupport finished successfully", http.StatusOK)
}

func (h *ContractHandler) DeleteContract(ctx *gin.Context) error {
	contractID := ctx.Param("id")

	if err := h.service.DeleteContract(ctx, contractID); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, "Contract deleted successfully", http.StatusOK)
}

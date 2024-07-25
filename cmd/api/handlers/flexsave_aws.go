package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Flexsave struct {
	loggerProvider logger.Provider
	service        *flexsaveresold.Service
}

type customerRequest struct {
	CustomerIDsToExclude []string `json:"customerIDsToExclude"`
}

func NewFlexSaveAWS(log logger.Provider, conn *connection.Connection) *Flexsave {
	service := flexsaveresold.NewService(log, conn)

	return &Flexsave{
		log,
		service,
	}
}

func (h *Flexsave) ActivateFlexsaveOrderHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	orderID := ctx.Param("orderID")
	if orderID == "" {
		return web.NewRequestError(errors.New("missing orderID parameter"), http.StatusBadRequest)
	}

	h.loggerProvider(ctx).SetLabels(map[string]string{
		"orderID":              orderID,
		logger.LabelCustomerID: customerID,
	})

	err := h.service.ActivateFlexRIOrder(ctx, customerID, orderID, false)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) UpdateFlexSaveAutopilotOrdersHandler(ctx *gin.Context) error {
	err := h.service.UpdateFlexRIAutopilotOrders(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) UpdateFlexSaveAutopilotOrdersByCustomerHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	err := h.service.UpdateFlexRIAutopilotOrdersByCustomer(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) BackfillFlexsaveInvoiceAdjustment(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customer id parameter"), http.StatusBadRequest)
	}

	type payload struct {
		Month        string             `json:"month" validate:"required,datetime=2006-01-02"`
		TotalSavings float64            `json:"totalSavings" validate:"required"`
		Entities     map[string]float64 `json:"entities"`
	}

	var body payload

	err := ctx.ShouldBindJSON(&body)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	validate := validator.New()

	err = validate.Struct(body)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err = h.service.UpdateInvoiceAdjustment(ctx, customerID, body.Month, body.TotalSavings, body.Entities)
	if err != nil {
		if strings.Contains(err.Error(), "validateEntities()") {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err = h.service.CreateBillingRows(ctx, customerID, body.Month, body.TotalSavings)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) UpdateInstancesPricingHandler(ctx *gin.Context) error {
	err := h.service.UpdateInstancesPricing(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) AcceptAutopilotOrdersHandler(ctx *gin.Context) error {
	err := h.service.AcceptAutopilotOrders(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) RegenerateAutopilotOrdersForAllCustomersHandler(ctx *gin.Context) error {
	offset, err := getOffset(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	customerOffset := ""
	customerOffset = ctx.Query("customerOffset")

	var req customerRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err != io.EOF {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err = h.service.RegenerateAutopilotOrdersForAllCustomers(ctx, offset, customerOffset, req.CustomerIDsToExclude)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) RegenerateAutopilotOrdersForCustomerHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	offset, err := getOffset(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err = h.service.RegenerateAutopilotOrdersForCustomer(ctx, customerID, offset)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) ReactivateOrdersHandler(ctx *gin.Context) error {
	var orderIds []int

	if err := ctx.ShouldBindJSON(&orderIds); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.ReactivateOrders(ctx, orderIds)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) AutopilotOrdersEndDateUpdateHandler(ctx *gin.Context) error {
	customerID, month, endDate := ctx.Param("customerID"), ctx.Query("month"), ctx.Query("endTime")
	if month == "" || endDate == "" {
		return web.NewRequestError(errors.New("missing customerID, and/or month(format 2006-01) and/or endTime (format 2006-01-02_Hr) parameters"), http.StatusBadRequest)
	}

	err := h.service.UpdateFlexsaveDailyOrderEndTimeByCustomerAndMonth(ctx, customerID, month, endDate)
	if err != nil {
		return web.NewRequestError(err, 500)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) ReactivateAllOrdersForCustomerHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	err := h.service.ReactivateAllOrdersForCustomer(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func getOffset(ctx *gin.Context) (int, error) {
	offset := 1
	offsetRequest := ctx.Query("monthOffset")

	if offsetRequest != "" {
		var err error

		offset, err = strconv.Atoi(offsetRequest)
		if err != nil {
			return 0, err
		}
	}

	return offset, nil
}

func handleServiceError(err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*flexsaveresold.ServiceError); ok {
		switch e.Err {
		case web.ErrBadRequest:
			return web.NewRequestError(err, http.StatusBadRequest)
		case web.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case web.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		}
	}

	if err := web.TranslateError(err); err != nil {
		return err
	}

	return web.NewRequestError(err, http.StatusInternalServerError)
}

func (h *Flexsave) FanoutImportPotentialHandler(ctx *gin.Context) error {
	err := h.service.FanoutImportPotential(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) ImportPotentialForCustomerHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("missing customerID parameter"), http.StatusBadRequest)
	}

	err := h.service.ImportPotentialForCustomer(ctx, customerID)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Flexsave) CreateFlexSaveOrdersHandler(ctx *gin.Context) error {
	err := h.service.CreateFlexsaveOrders(ctx)
	if err != nil {
		return handleServiceError(err)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

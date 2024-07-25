package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

const (
	customerScope = "customer"
	asgScope      = "asg"
)

type SpotZero struct {
	*logger.Logging
	service *api.SpotZeroService
}

func (h *SpotZero) GetService() api.SpotScalingServiceInterface {
	return h.service
}

func NewSpotZero(log *logger.Logging, conn *connection.Connection) *SpotZero {
	service := api.NewSpotZeroService(log, conn)

	return &SpotZero{
		log,
		service,
	}
}

func (h *SpotZero) executeSpotScaling(ctx *gin.Context, forceManagedMode bool) error {
	var req model.ApplyConfigurationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	req.CustomerID = ctx.Param("customerID")
	if req.CustomerID == "" {
		return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
	}

	h.Logger(ctx).SetLabels(map[string]string{
		logger.LabelCustomerID: req.CustomerID,
		"accountId":            req.AccountID,
		"region":               req.Region,
		"asg":                  req.ASGName,
	})

	if req.Scope == "" {
		req.Scope = customerScope
	}

	if forceManagedMode {
		req.Scope = asgScope
	}

	req.ForceManagedMode = forceManagedMode

	resp, err := h.GetService().ExecuteSpotScaling(ctx, &req)
	if err != nil {
		switch err {
		case api.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case api.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *SpotZero) ApplyConfiguration(ctx *gin.Context) error {
	return h.executeSpotScaling(ctx, true)
}

func (h *SpotZero) RefreshASGs(ctx *gin.Context) error {
	return h.executeSpotScaling(ctx, false)
}

func (h *SpotZero) AveragePrices(ctx *gin.Context) error {
	var req model.AveragePricesRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	h.Logger(ctx).SetLabels(map[string]string{
		"accountId": req.AccountID,
		"region":    req.Region,
		"asg":       req.ASGName,
	})

	resp, err := h.GetService().AveragePrices(ctx, &req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *SpotZero) CheckFallbackOnDemandConfig(ctx *gin.Context) error {
	req := model.FallbackOnDemandRequest{
		CustomerID: ctx.Param("customerID"),
		AccountID:  ctx.Param("accountID"),
		Region:     ctx.Param("region"),
		Action:     "check",
	}

	if req.CustomerID == "" {
		return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
	}

	if req.AccountID == "" {
		return web.NewRequestError(errors.New("missing account id"), http.StatusBadRequest)
	}

	if req.Region == "" {
		return web.NewRequestError(errors.New("missing region"), http.StatusBadRequest)
	}

	h.setLoggerLabels(ctx, &req)

	resp, err := h.GetService().UpdateFallbackOnDemandConfig(ctx, &req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *SpotZero) UpdateAsgConfig(ctx *gin.Context) error {
	var req model.UpdateAsgConfigRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.CustomerID == "" {
		req.CustomerID = ctx.Param("customerID")
	}

	if req.AccountID == "" {
		return web.NewRequestError(errors.New("missing account id"), http.StatusBadRequest)
	}

	if req.Region == "" {
		return web.NewRequestError(errors.New("missing region"), http.StatusBadRequest)
	}

	if req.AsgName == "" {
		return web.NewRequestError(errors.New("missing asg name"), http.StatusBadRequest)
	}

	if req.Configuration == nil {
		return web.NewRequestError(errors.New("missing configuration"), http.StatusBadRequest)
	}

	h.Logger(ctx).SetLabels(map[string]string{
		logger.LabelCustomerID: req.CustomerID,
		"accountId":            req.AccountID,
		"region":               req.Region,
		"asgName":              req.AsgName,
	})

	resp, err := h.GetService().UpdateAsgConfig(ctx, &req)
	if err != nil {
		if ue, ok := err.(*model.UpdateAsgConfigError); ok {
			if ue.Code == "400" {
				return web.NewRequestError(err, http.StatusBadRequest)
			} else {
				return web.NewRequestError(err, http.StatusInternalServerError)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *SpotZero) AddFallbackOnDemandConfig(ctx *gin.Context) error {
	var req model.FallbackOnDemandRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	req.CustomerID = ctx.Param("customerID")
	if req.CustomerID == "" {
		return web.NewRequestError(errors.New("missing customer id"), http.StatusBadRequest)
	}

	req.Action = "add"

	h.setLoggerLabels(ctx, &req)

	resp, err := h.service.UpdateFallbackOnDemandConfig(ctx, &req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp, http.StatusOK)
}

func (h *SpotZero) setLoggerLabels(ctx *gin.Context, req *model.FallbackOnDemandRequest) {
	h.Logger(ctx).SetLabels(map[string]string{
		logger.LabelCustomerID: req.CustomerID,
		"accountId":            req.AccountID,
		"region":               req.Region,
		"action":               req.Action,
	})
}

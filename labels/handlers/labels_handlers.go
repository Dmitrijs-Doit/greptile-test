package handlers

import (
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	"github.com/doitintl/hello/scheduled-tasks/labels/service"
	"github.com/doitintl/hello/scheduled-tasks/labels/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type Labels struct {
	loggerProvider logger.Provider
	service        iface.LabelsIface
}

func NewLabels(log logger.Provider, conn *connection.Connection) *Labels {
	s := service.NewLabelsService(log, conn)

	return &Labels{
		log,
		s,
	}
}

func (h *Labels) CreateLabel(ctx *gin.Context) error {
	var body service.CreateLabelRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	body.CustomerID = ctx.Param("customerID")
	body.UserEmail = ctx.GetString(common.CtxKeys.Email)

	if err := isValidCreateLabelRequest(body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	label, err := h.service.CreateLabel(ctx, body)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, label, http.StatusCreated)
}

func (h *Labels) UpdateLabel(ctx *gin.Context) error {
	var req service.UpdateLabelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	req.LabelID = ctx.Param("labelID")

	if err := isValidUpdateLabelRequest(req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	label, err := h.service.UpdateLabel(ctx, req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, label, http.StatusOK)
}

func (h *Labels) DeleteLabel(ctx *gin.Context) error {
	labelID := ctx.Param("labelID")

	if labelID == "" {
		return web.NewRequestError(labels.ErrInvalidLabelID, http.StatusBadRequest)
	}

	err := h.service.DeleteLabel(ctx, labelID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Labels) AssignLabels(ctx *gin.Context) error {
	var req service.AssignLabelsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	req.CustomerID = ctx.Param("customerID")

	if err := validateAssignObjectLabelsRequest(req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.service.AssignLabels(ctx, req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

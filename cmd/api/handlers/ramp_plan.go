package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/contract/rampplan"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type RampPlan struct {
	*logger.Logging
	service *rampplan.Service
}
type Request struct {
	PlanID string `json:"planId"`
}

type CreateRampPlanRequest struct {
	ContractID string `json:"contract-id"`
	Name       string `json:"name"`
}

func NewRampPlan(log *logger.Logging, conn *connection.Connection) *RampPlan {
	service, err := rampplan.NewRampPlanService(log, conn)
	if err != nil {
		panic(err)
	}

	return &RampPlan{
		log,
		service,
	}
}

func (h *RampPlan) UpdateAllRampPlans(ctx *gin.Context) error {
	logger := h.Logger(ctx)

	rampPlanDocs, err := h.service.RampPlansDal.GetAllActiveRampPlans(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	for _, rp := range rampPlanDocs {
		body, err := json.Marshal(Request{
			PlanID: rp.Ref.ID,
		})
		if err != nil {
			logger.Errorf("failed to marshal ramp plan %s with error: %s", rp.Ref.ID, err)
			continue
		}

		config := common.CloudTaskConfig{
			Method: 1,
			Path:   "/tasks/dashboard/update-ramp-plan",
			Queue:  common.TaskQueueUpdateRampPlan,
			Body:   body,
		}
		if _, err := common.CreateCloudTask(ctx, &config); err != nil {
			logger.Errorf("failed to create ramp plan performance update task for ramp plan %s with error: %s", rp.Ref.ID, err)
			continue
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *RampPlan) UpdateRampPlanByID(ctx *gin.Context) error {
	var body Request
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	rampPlanDoc, err := h.service.RampPlansDal.GetRampPlan(ctx, body.PlanID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.UpdateUsage(ctx, rampPlanDoc); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *RampPlan) CreateRampPlan(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var body CreateRampPlanRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.CreateRampPlan(ctx, customerID, body.ContractID, body.Name); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *RampPlan) CreateRampPlans(ctx *gin.Context) error {
	if err := h.service.CreateRampPlans(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager"
	computeManager "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute"
	rdsManager "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/rds"
	sagemakerManager "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/sagemaker"
	payermanagerutils "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Handler interface {
	ProcessOpsUpdates(ctx *gin.Context) error
}

type handler struct {
	payerManager          payermanager.Service
	computeStateService   computeManager.Service
	rdsStateService       rdsManager.Service
	sagemakerStateService sagemakerManager.Service
}

func NewHandler(log logger.Provider, conn *connection.Connection) Handler {
	return &handler{
		payerManager:          payermanager.NewService(log, conn),
		computeStateService:   computeManager.NewService(log, conn),
		rdsStateService:       rdsManager.NewService(log, conn),
		sagemakerStateService: sagemakerManager.NewService(log, conn),
	}
}

func validateTransitions(compute, rds, sagemaker string) error {
	if compute == payermanagerutils.PendingState && sagemaker == payermanagerutils.ActiveState {
		return errors.New("sagemaker cannot be enabled when compute is not")
	}

	if compute == payermanagerutils.PendingState && rds == payermanagerutils.ActiveState {
		return errors.New("rds cannot be enabled when compute is not")
	}

	if compute == payermanagerutils.DisabledState && sagemaker != payermanagerutils.DisabledState {
		return errors.New("sagemaker must be disabled before can disable compute")
	}

	if compute == payermanagerutils.DisabledState && rds != payermanagerutils.DisabledState {
		return errors.New("rds must be disabled before can disable compute")
	}

	return nil
}

func (h *handler) ProcessOpsUpdates(ctx *gin.Context) error {
	payerID := ctx.Param("payerId")

	var body payermanager.FormEntry

	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	payer, err := h.payerManager.GetPayer(ctx, payerID)
	if err != nil {
		return web.NewRequestError(
			errors.Wrapf(err, "GetPayer() failed for accountId '%s'", payerID),
			http.StatusInternalServerError)
	}

	compute := payer.Status
	if body.Status != "" {
		compute = body.Status
	}

	rds := payer.RDSStatus
	if body.RDSStatus != nil {
		rds = *body.RDSStatus
	}

	sagemaker := payer.SageMakerStatus
	if body.SagemakerStatus != nil {
		sagemaker = *body.SagemakerStatus
	}

	if err := validateTransitions(compute, rds, sagemaker); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	// If we have it, set the reason for a status change on the context, so it can be processed later
	if body.StatusChangeReason != nil {
		ctx.Set(utils.StatusChangeReasonContextKey, *body.StatusChangeReason)
	}

	err = h.payerManager.UpdateNonStatusPayerConfigFields(ctx, payer, body)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err = h.computeStateService.ProcessPayerStatusTransition(ctx, payerID, payer.CustomerID, payer.Status, compute)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err = h.rdsStateService.ProcessPayerStatusTransition(ctx, payerID, payer.CustomerID, payer.RDSStatus, rds)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err = h.sagemakerStateService.ProcessPayerStatusTransition(ctx, payerID, payer.CustomerID, payer.SageMakerStatus, sagemaker)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

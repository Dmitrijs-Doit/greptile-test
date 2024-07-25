package handlers

import (
	"errors"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/application"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/application/task"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/rows_validator"
	tableanalytics "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/table_analytics"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/gin-gonic/gin"
)

type FlexsaveStandaloneGCPBilling struct {
	loggerProvider logger.Provider
	*connection.Connection
	internalManager                 *application.InternalManager
	internalAccountUpdate           *task.InternalAccountBillingUpdateTask
	externalManager                 *application.ExternalManager
	alternativeManager              *application.AlternativeManager
	externalToBucketAccountUpdate   *task.ExternalToBucketTask
	externalFromBucketAccountUpdate *task.ExternalFromBucketTask
	onborading                      *application.Onboarding
	sanity                          *application.Sanity
	rowsValidator                   *rows_validator.RowsValidator
	tableAnalytics                  *tableanalytics.TableAnalytics
}

func NewFlexsaveStandaloneGCPBilling(log logger.Provider, conn *connection.Connection) *FlexsaveStandaloneGCPBilling {
	return &FlexsaveStandaloneGCPBilling{
		log,
		conn,
		application.NewInternalManager(log, conn),
		task.NewInternalAccountBillingUpdateTask(log, conn),
		application.NewExternalManager(log, conn),
		application.NewAlternativeManager(log, conn),
		task.NewExternalToBucketTask(log, conn),
		task.NewExternalFromBucketBillingUpdateTask(log, conn),
		application.NewOnboarding(log, conn),
		application.NewSanity(log, conn),
		rows_validator.NewRowsValidator(log, conn),
		tableanalytics.NewTableAnalytics(log, conn),
	}
}

func (h *FlexsaveStandaloneGCPBilling) RunSanity(ctx *gin.Context) error {
	defer ctx.Done()

	err := h.sanity.RunSanity(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RunInternalManager(ctx *gin.Context) error {
	defer ctx.Done()

	err := h.internalManager.RunInternalManager(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RunInternalTask(ctx *gin.Context) error {
	var body dataStructures.UpdateRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.internalAccountUpdate.RunInternalTask(ctx, &body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveStandaloneGCPBilling) RunExternalManager(ctx *gin.Context) error {
	if err := h.externalManager.RunExternalManager(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveStandaloneGCPBilling) RunAlternativeManager(ctx *gin.Context) error {
	if err := h.alternativeManager.RunAlternativeManager(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveStandaloneGCPBilling) RunToBucketExternalTask(ctx *gin.Context) error {
	defer ctx.Done()

	var body dataStructures.UpdateRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		h.loggerProvider(ctx).Errorf("unable to run RunToBucketExternalTask. Caused by %s", err)
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.externalToBucketAccountUpdate.RunExternalToBucketTask(ctx, &body); err != nil {
		h.loggerProvider(ctx).Errorf("execution RunToBucketExternalTask failed. Caused by %s", err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *FlexsaveStandaloneGCPBilling) RunFromBucketExternalTask(ctx *gin.Context) error {
	defer ctx.Done()

	var body dataStructures.UpdateRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.externalFromBucketAccountUpdate.RunFromBucketExternalTask(ctx, &body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RemoveAll(ctx *gin.Context) error {
	if err := h.onborading.RemoveAll(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) Onboaring(ctx *gin.Context) error {
	var body dataStructures.OnboardingRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.onborading.Onboard(ctx, &body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RemoveBilling(ctx *gin.Context) error {
	var body dataStructures.DeleteBillingRequestBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.onborading.RemoveBilling(ctx, &body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) ValidateRows(ctx *gin.Context) error {
	if err := h.rowsValidator.ValidateRows(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RunMonitor(ctx *gin.Context) error {
	if err := h.rowsValidator.RunMonitor(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RunMonitorBillingAccount(ctx *gin.Context) error {
	billingAccoutnID := ctx.Param("billingAccountId")

	if billingAccoutnID == "" {
		return web.NewRequestError(errors.New("missing billing account id"), http.StatusBadRequest)
	}

	if err := h.rowsValidator.RunMonitorForBilling(ctx, billingAccoutnID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) ValidateCustomerRows(ctx *gin.Context) error {
	billingAccoutnID := ctx.Param("billingAccountId")

	if billingAccoutnID == "" {
		return web.NewRequestError(errors.New("missing billing account id"), http.StatusBadRequest)
	}

	if err := h.rowsValidator.ValidateCustomerRows(ctx, billingAccoutnID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RunDetailedTableRewritesMapping(ctx *gin.Context) error {
	if err := h.tableAnalytics.RunDetailedTableRewritesMapping(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *FlexsaveStandaloneGCPBilling) RunDataFreshnessReport(ctx *gin.Context) error {
	if err := h.tableAnalytics.RunDataFreshnessReport(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

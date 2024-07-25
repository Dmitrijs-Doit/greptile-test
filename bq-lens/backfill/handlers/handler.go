package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BackfillHandler struct {
	loggerProvider    logger.Provider
	backfillScheduler serviceIface.BackfillSchedulerService
	backfillService   serviceIface.BackfillService
}

func NewBackfillHandler(
	log logger.Provider,
	backfillScheduler serviceIface.BackfillSchedulerService,
	backfillService serviceIface.BackfillService,
) *BackfillHandler {
	return &BackfillHandler{
		loggerProvider:    log,
		backfillScheduler: backfillScheduler,
		backfillService:   backfillService,
	}
}

func (h *BackfillHandler) ScheduleBackfill(ctx *gin.Context) error {
	var request ScheduleBackfillRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validator.New().Struct(&request); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.backfillScheduler.ScheduleBackfill(ctx, request.SinkID, request.TestMode); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *BackfillHandler) Backfill(ctx *gin.Context) error {

	var request BackfillRequest

	if err := ctx.ShouldBindJSON(&request); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validator.New().Struct(&request); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	backfillDate, err := time.Parse("2006-01-02", request.BackfillDate)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	minCreationTime, err := time.Parse(time.RFC3339, request.DateBackfillInfo.BackfillMinCreationTime)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	maxCreationTime, err := time.Parse(time.RFC3339, request.DateBackfillInfo.BackfillMaxCreationTime)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	backfillInfo := domain.DateBackfillInfo{
		BackfillMinCreationTime: minCreationTime,
		BackfillMaxCreationTime: maxCreationTime,
	}

	if err := h.backfillService.Backfill(ctx, request.DocID, request.CustomerID, request.BackfillProject, backfillDate, backfillInfo); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

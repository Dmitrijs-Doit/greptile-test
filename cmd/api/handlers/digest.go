package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	budgetsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/digest"
	highchartsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/highcharts/service"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type Digest struct {
	loggerProvider logger.Provider
	service        digest.IDigestService
}

func NewDigest(loggerProvider logger.Provider, conn *connection.Connection) *Digest {
	budgetService, err := budgetsService.NewBudgetsService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	highcharts, err := highchartsService.NewHighcharts(loggerProvider, conn, budgetService)
	if err != nil {
		panic(err)
	}

	service, err := digest.NewDigestService(loggerProvider, conn, highcharts)
	if err != nil {
		panic(err)
	}

	return &Digest{
		loggerProvider,
		service,
	}
}

func (d *Digest) ScheduleDaily(ctx *gin.Context) error {
	err := d.service.ScheduleDaily(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (d *Digest) ScheduleWeekly(ctx *gin.Context) error {
	l := d.loggerProvider(ctx)

	if d := times.CurrentDayUTC().Day(); d <= 7 {
		l.Info("Weekly digest is not sent on the first week of the month")
		return web.Respond(ctx, nil, http.StatusOK)
	}

	err := d.service.ScheduleWeekly(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (d *Digest) GenerateDigest(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginDigest)

	var req digest.GenerateTaskRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if req.CustomerID == "" {
		err := errors.New("invalid digest request - missing customer id")
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.AttributionID == "" {
		err := errors.New("invalid digest request - missing attribution id")
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if req.Frequency != digest.FrequencyDaily && req.Frequency != digest.FrequencyWeekly {
		err := errors.New("invalid digest request - invalid frequency")
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	dayParam, err := d.parseHandleCustomerQueryParams(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := d.service.Generate(ctx, dayParam, &req); err != nil {
		return web.Respond(ctx, err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (d *Digest) parseHandleCustomerQueryParams(ctx *gin.Context) (int, error) {
	dayParamString := ctx.Query("day")
	if dayParamString == "" {
		return 0, nil
	}

	dayParam, err := strconv.Atoi(dayParamString)
	if err != nil {
		return 0, err
	}

	if dayParam < 1 {
		return 0, errors.New("day query param must be a positive integer")
	}

	year, month, _ := time.Now().UTC().Date()
	nextMonth := time.Date(year, month, 0, 0, 0, 0, 0, time.UTC)

	lastDayOfCurrentMonth := nextMonth.AddDate(0, 0, -1)
	if dayParam > lastDayOfCurrentMonth.Day() {
		return 0, errors.New("day query parameter must be a valid date in current month")
	}

	return dayParam, nil
}

func (d *Digest) HandleMonthlyDigest(ctx *gin.Context) error {
	l := d.loggerProvider(ctx)

	l.Info("Handling monthly digest")

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginDigest)

	if err := d.service.GetMonthlyDigest(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (d *Digest) SendDigest(ctx *gin.Context) error {
	var req digest.Data
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := d.service.Send(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

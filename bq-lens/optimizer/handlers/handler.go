package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Optimizer struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        iface.OptimizerService
}

func NewOptimizer(ctx context.Context, log logger.Provider, conn *connection.Connection) *Optimizer {

	svc := service.NewOptimizer(ctx, log, conn)

	return &Optimizer{
		loggerProvider: log,
		conn:           conn,
		service:        svc,
	}
}

func (h *Optimizer) SingleCustomerOptimizer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":    "adoption",
		"feature":  "bq-lens",
		"module":   "discovery",
		"service":  "discovery",
		"customer": customerID,
	})

	var input domain.Payload

	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validator.New().Struct(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	startTimestamp := time.Now()

	l.Infof("Optimizer for customer '%s' started", customerID)

	err := h.service.SingleCustomerOptimizer(ctx, customerID, input)

	l.Infof("Optimizer for customer '%s' completed with total duration of '%v' seconds", customerID, time.Since(startTimestamp).Seconds())

	if err != nil {
		l.Error(err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Optimizer) AllCustomersOptimizer(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "optimizer",
		"service": "optimizer",
	})

	errs, err := h.service.Schedule(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(errs) > 0 {
		l.Errorf(FailedToOptimizeErrFormat, errs)
		return web.Respond(ctx, errs, http.StatusMultiStatus)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

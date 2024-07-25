package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service"
	serviceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Metric struct {
	loggerProvider logger.Provider
	service        serviceIface.IMetricsService
}

func NewMetric(log logger.Provider, conn *connection.Connection) *Metric {
	s := service.NewMetricsService(log, conn)
	return &Metric{
		log,
		s,
	}
}

func (h *Metric) DeleteMetricsHandler(ctx *gin.Context) error {
	var body service.DeleteMetricsRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	validate := validator.New()

	if err := validate.Struct(body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.service.DeleteMany(ctx, body); err != nil {
		switch {
		case errors.As(err, &service.CustomMetricNotFoundError{}):
			return web.NewRequestError(err, http.StatusNotFound)

		case errors.As(err, &service.PresetMetricsCannotBeDeletedError{}),
			errors.As(err, &service.MetricIsInUseError{}):
			return web.NewRequestError(err, http.StatusForbidden)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

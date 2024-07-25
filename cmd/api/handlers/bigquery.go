package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BigQueryHandler struct {
	loggerProvider logger.Provider
	service        *bigquery.Service
}

var ErrUnableToProvideQuery = errors.New("unable to provide query")

func NewBigQueryHandler(loggerProvider logger.Provider, conn *connection.Connection) *BigQueryHandler {
	getQueryForJob := bigquery.NewService(loggerProvider, conn)

	return &BigQueryHandler{
		loggerProvider,
		getQueryForJob,
	}
}

func (h *BigQueryHandler) GetQuery(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	jobID := ctx.Param("jobId")
	location := ctx.Param("location")
	project := ctx.Param("project")
	customerID := ctx.Param("customerID")

	if customerID == "" || location == "" || project == "" {
		return web.ErrBadRequest
	}

	job, err := h.service.GetCustomerJob(ctx, bigquery.GetJobByIDParams{
		JobID:      jobID,
		CustomerID: customerID,
		Project:    project,
		Location:   location,
	})
	if err != nil {
		l.Errorf("failed to get customer job with error: %s", err)
		return web.NewRequestError(ErrUnableToProvideQuery, http.StatusBadRequest)
	}

	query, err := h.service.GetQueryFromJob(ctx, job)
	if err != nil {
		l.Errorf("failed to get query from job with error: %s", err)
		return web.NewRequestError(ErrUnableToProvideQuery, http.StatusBadRequest)
	}

	type Body struct {
		Query string `json:"query"`
	}

	return web.Respond(ctx, &Body{
		Query: query,
	}, http.StatusOK)
}

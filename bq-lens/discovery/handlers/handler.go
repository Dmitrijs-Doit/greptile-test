package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	doitBQ "github.com/doitintl/bigquery"
	discoveryBigqueryDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/service"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TableDiscovery struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        iface.DiscoveryService
}

func NewTableDiscovery(log logger.Provider, conn *connection.Connection) *TableDiscovery {
	cloudConnectService := cloudconnect.NewCloudConnectService(log, conn)
	discoveryBigqueryDal := discoveryBigqueryDal.NewBigquery(log, &doitBQ.QueryHandler{})
	svc := service.NewDiscovery(log, conn, discoveryBigqueryDal, cloudConnectService)

	return &TableDiscovery{
		loggerProvider: log,
		conn:           conn,
		service:        svc,
	}
}

func (h *TableDiscovery) AllCustomersTablesDiscovery(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":   "adoption",
		"feature": "bq-lens",
		"module":  "discovery",
		"service": "discovery",
	})

	errs, err := h.service.Schedule(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(errs) > 0 {
		l.Errorf(FailedToDiscoverTablesErrFormat, errs)
		return web.Respond(ctx, errs, http.StatusMultiStatus)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *TableDiscovery) SingleCustomerTablesDiscovery(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":    "adoption",
		"feature":  "bq-lens",
		"module":   "discovery",
		"service":  "discovery",
		"customer": customerID,
	})

	var input service.TablesDiscoveryPayload

	if err := ctx.ShouldBindJSON(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validator.New().Struct(&input); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.service.TablesDiscovery(ctx, customerID, input)
	if err != nil {
		l.Error(err)
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

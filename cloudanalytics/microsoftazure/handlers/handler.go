package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	azureTablemgmt "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure/tablemanagement"
	azureTablemgmtIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure/tablemanagement/iface"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	customersDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	queryParamAllPartitions = "allPartitions"
	queryParamFrom          = "from"
	queryParamTo            = "to"
	queryParamNumPartitions = "numPartitions"
	queryParamInterval      = "interval"
	pathParamCustomerID     = "customerID"
)

type AnalyticsMicrosoftAzure struct {
	loggerProvider logger.Provider
	tableMgmt      azureTablemgmtIface.BillingTableManagementService
}

func NewAnalyticsMicrosoftAzure(loggerProvider logger.Provider, conn *connection.Connection) *AnalyticsMicrosoftAzure {
	customerDAL := customersDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	tableMgmt, err := azureTablemgmt.NewBillingTableManagementService(loggerProvider, conn, customerDAL)
	if err != nil {
		panic(err)
	}

	return &AnalyticsMicrosoftAzure{
		loggerProvider,
		tableMgmt,
	}
}

func (h *AnalyticsMicrosoftAzure) UpdateAggregateAllAzureCustomers(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAzure)

	allPartitions := ctx.Query(queryParamAllPartitions) == "true"

	if errs := h.tableMgmt.UpdateAllAggregatedTablesAllCustomers(ctx, allPartitions); errs != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": errs})
		return nil
	}

	return nil
}

func (h *AnalyticsMicrosoftAzure) UpdateAggregatedTableAzure(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAzure)

	interval := ctx.Query(queryParamInterval)
	if interval == "" {
		return web.NewRequestError(errors.New("interval query param is required"), http.StatusBadRequest)
	}

	suffix := ctx.Param(pathParamCustomerID)
	if suffix == "" {
		return web.NewRequestError(azureTablemgmt.ErrSuffixIsEmpty, http.StatusBadRequest)
	}

	// allPartitions == "true" -> Update all table partitions
	// allPartitions == "" (or any other value) -> (default) Update last partition based on current time
	allPartitions := ctx.Query(queryParamAllPartitions) == "true"
	if err := h.tableMgmt.UpdateAggregatedTable(ctx, suffix, interval, allPartitions); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *AnalyticsMicrosoftAzure) UpdateAllAzureAggregatedTables(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAzure)

	suffix := ctx.Param(pathParamCustomerID)
	if suffix == "" {
		return web.NewRequestError(azureTablemgmt.ErrSuffixIsEmpty, http.StatusBadRequest)
	}

	allPartitions := ctx.Query(queryParamAllPartitions) == "true"
	if errs := h.tableMgmt.UpdateAllAggregatedTables(ctx, suffix, allPartitions); errs != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"errors": errs})
		return nil
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMicrosoftAzure) UpdateCSPTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAzure)

	startDate := ctx.Query(queryParamFrom)
	endDate := ctx.Query(queryParamTo)

	if err := h.tableMgmt.UpdateCSPTable(ctx, startDate, endDate); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMicrosoftAzure) UpdateCSPAggregatedTable(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginTablesMgmtAzure)

	// allPartitions == "true" -> Update all table partitions
	// allPartitions == "" (or any other value) -> (default) Update last partition based on current time
	allPartitions := ctx.Query(queryParamAllPartitions) == "true"

	fromDate := ctx.Query(queryParamFrom)

	var numPartitions int

	if v := ctx.Query(queryParamNumPartitions); v != "" {
		var err error

		numPartitions, err = strconv.Atoi(v)
		if err != nil || numPartitions < 0 {
			numPartitions = 0
		}
	}

	if err := h.tableMgmt.UpdateCSPAggregatedTable(
		ctx,
		allPartitions,
		fromDate,
		numPartitions,
	); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

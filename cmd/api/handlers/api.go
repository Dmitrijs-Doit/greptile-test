package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/api"
	"github.com/doitintl/hello/scheduled-tasks/api/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/reporttier"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/service/stats"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

const (
	maxResultsLimit = 250
)

type APIv1 struct {
	loggerProvider logger.Provider
	*connection.Connection
	service          *api.APIV1Service
	reportAPIService iface.ReportAPIService
}

func NewAPIv1(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) *APIv1 {
	service, err := api.NewAPIV1Service(ctx, loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	reportDAL := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	reportStatsService, err := stats.NewReportStatsService(loggerProvider, reportDAL)
	if err != nil {
		panic(err)
	}

	tierService := tier.NewTiersService(conn.Firestore)

	attributionDAL := attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	attrGroupDAL := dal.NewAttributionGroupsFirestoreWithClient(conn.Firestore)

	customerDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	reportTierService := reporttier.NewReportTierService(
		loggerProvider,
		reportDAL,
		customerDAL,
		attributionDAL,
		attrGroupDAL,
		tierService,
		doitEmployeesService,
	)

	reportAPIService, err := api.NewReportAPIService(
		loggerProvider,
		conn,
		reportStatsService,
		reportTierService,
	)
	if err != nil {
		panic(err)
	}

	return &APIv1{
		loggerProvider,
		conn,
		service,
		reportAPIService,
	}
}

func (h *APIv1) ListKnownIssues(ctx *gin.Context) error {
	api.ListKnownIssues(ctx, h.Connection)

	return nil
}

func (h *APIv1) GetKnownIssue(ctx *gin.Context) error {
	api.GetKnownIssue(ctx, h.Connection)

	return nil
}

func (h *APIv1) ListInvoices(ctx *gin.Context) error {
	api.ListInvoices(ctx, h.Connection)

	return nil
}

func (h *APIv1) GetInvoice(ctx *gin.Context) error {
	api.GetInvoice(ctx, h.Connection)

	return nil
}

func (h *APIv1) CreateAsset(ctx *gin.Context) error {
	h.service.CreateAsset(ctx)

	return nil
}

func (h *APIv1) ListAnomalies(ctx *gin.Context) error {
	api.ListAnomalies(ctx, h.Connection)

	return nil
}

func (h *APIv1) GetAnomaly(ctx *gin.Context) error {
	api.GetAnomaly(ctx, h.Connection)

	return nil
}

func (h *APIv1) ListReports(ctx *gin.Context) error {
	h.reportAPIService.ListReports(ctx, h.Connection)

	return nil
}

func (h *APIv1) RunReport(ctx *gin.Context) error {
	h.reportAPIService.RunReport(ctx, h.Connection)

	return nil
}

func (h *APIv1) CreateBudgetHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	email := ctx.GetString("email")

	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	var budget api.Budget

	if err := ctx.ShouldBindJSON(&budget); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validateBudget(&budget); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginReportsAPI)

	responseBudget, err := h.service.CreateBudget(ctx, &api.BudgetRequestData{
		Budget:     &budget,
		Email:      email,
		CustomerID: customerID,
	})
	if err != nil {
		return web.NewRequestError(err.Error, http.StatusInternalServerError)
	}

	return web.Respond(ctx, responseBudget, http.StatusCreated)
}

func validateBudget(b *api.Budget) error {
	if !common.IsNil(b.Type) && *b.Type == "recurring" && !common.IsNil(b.EndPeriod) {
		return errors.New(api.ErrorBudgetRecurringWithEndPeriod)
	}

	return nil
}

func (h *APIv1) UpdateBudgetHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	email := ctx.GetString("email")

	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	budgetID := ctx.Param("id")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	var budget api.Budget
	if err := ctx.ShouldBindJSON(&budget); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := validateBudget(&budget); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginReportsAPI)

	responseBudget, err := h.service.UpdateBudget(ctx, &api.BudgetRequestData{
		BudgetID:   budgetID,
		Budget:     &budget,
		Email:      email,
		CustomerID: customerID,
	})
	if err != nil {
		return web.NewRequestError(err.Error, err.Code)
	}

	return web.Respond(ctx, responseBudget, http.StatusOK)
}

func (h *APIv1) DeleteBudgetHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	email := ctx.GetString("email")

	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	budgetID := ctx.Param("id")

	if err := h.service.DeleteBudget(ctx, &api.BudgetRequestData{
		BudgetID: budgetID,
		Email:    email,
	}); err != nil {
		return web.NewRequestError(errors.New(err.Msg), err.Code)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

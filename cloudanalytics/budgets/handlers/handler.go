package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AnalyticsBudgets struct {
	loggerProvider logger.Provider
	service        service.IBudgetsService
	customerDAL    customerDAL.Customers
}

func NewAnalyticsBudgets(loggerProvider logger.Provider, conn *connection.Connection) *AnalyticsBudgets {
	s, err := service.NewBudgetsService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	customerDAL := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)

	return &AnalyticsBudgets{
		loggerProvider,
		s,
		customerDAL,
	}
}

func (h *AnalyticsBudgets) UpdateBudgetSharingHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	budgetID := ctx.Param("budgetID")
	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)

	if budgetID == "" {
		return web.NewRequestError(service.ErrMissingBudgetID, http.StatusInternalServerError)
	}

	customer, err := h.customerDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if customer.PresentationMode != nil && customer.PresentationMode.Enabled {
		budget, err := h.service.GetBudget(ctx, budgetID)
		if err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}

		if budget.Customer.ID == customer.PresentationMode.CustomerID {
			return web.NewRequestError(service.ErrUnauthorized, http.StatusForbidden)
		}
	}

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"budgetId":             budgetID,
	})

	var body service.ShareBudgetRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.Collaborators) == 0 {
		return web.NewRequestError(service.ErrNoCollaborators, http.StatusBadRequest)
	}

	if err := h.service.ShareBudget(ctx, body, budgetID, userID, email); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsBudgets) UpdateBudgetEnforcedByMeteringHandler(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	customerID := ctx.Param("customerID")
	budgetID := ctx.Param("budgetID")
	email := ctx.GetString(common.CtxKeys.Email)

	if budgetID == "" {
		return web.NewRequestError(service.ErrMissingBudgetID, http.StatusInternalServerError)
	}

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"budgetId":             budgetID,
	})

	budget, err := h.service.GetBudget(ctx, budgetID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.UpdateEnforcedByMeteringField(ctx, budgetID, budget.Collaborators, budget.Recipients, budget.Public); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsBudgets) ExternalAPIGetBudget(ctx *gin.Context) error {
	budgetID := ctx.Param("id")

	if budgetID == "" {
		return web.NewRequestError(service.ErrMissingBudgetID, http.StatusBadRequest)
	}

	l := h.loggerProvider(ctx)
	email := ctx.GetString(common.CtxKeys.Email)
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"budgetId":             budgetID,
	})

	responseBudget, err := h.service.GetBudgetExternal(ctx, budgetID, email, customerID)

	if err != nil {
		switch err {
		case web.ErrUnauthorized:
			return web.NewRequestError(err, http.StatusForbidden)
		case web.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		}

		return web.NewRequestError(web.ErrInternalServerError, http.StatusInternalServerError)
	}

	return web.Respond(ctx, responseBudget, http.StatusOK)
}

func (h *AnalyticsBudgets) ExternalAPIListBudgets(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	email := ctx.GetString(common.CtxKeys.Email)

	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	isDoitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)
	query := ctx.Request.URL.Query()

	budgetsRequest := service.BudgetsRequest{
		MaxResults:      query.Get("maxResults"),
		MinCreationTime: query.Get("minCreationTime"),
		MaxCreationTime: query.Get("maxCreationTime"),
		PageToken:       query.Get("pageToken"),
		Filter:          query.Get("filter"),
	}

	budgetList, paramErr, internalErr := h.service.ListBudgets(ctx, &service.ExternalAPIListArgsReq{
		BudgetRequest:  &budgetsRequest,
		Email:          email,
		CustomerID:     customerID,
		IsDoitEmployee: isDoitEmployee,
	})

	if paramErr != nil {
		return web.NewRequestError(paramErr, http.StatusBadRequest)
	}

	if internalErr != nil {
		return web.NewRequestError(web.ErrInternalServerError, http.StatusInternalServerError)
	}

	return web.Respond(ctx, budgetList, http.StatusOK)
}

func (h *AnalyticsBudgets) DeleteManyHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")
	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEmail:      email,
		"action":               "deleteManyBudgets",
	})

	var body struct {
		IDs []string `json:"ids"`
	}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		l.Errorf(err.Error())
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.IDs) == 0 {
		return web.NewRequestError(errors.New("no budget ids provided"), http.StatusBadRequest)
	}

	if err := h.service.DeleteMany(ctx, email, body.IDs); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

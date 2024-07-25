package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service"
	alerttier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier/iface"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type AnalyticsAlerts struct {
	loggerProvider   logger.Provider
	service          service.AlertsService
	alertTierService iface.AlertTierService
}

func NewAnalyticsAlerts(
	ctx context.Context,
	log logger.Provider,
	conn *connection.Connection,
) *AnalyticsAlerts {
	alertsService, err := service.NewAnalyticsAlertsService(ctx, log, conn, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	tierService := tier.NewTiersService(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	alertTierService := alerttier.NewAlertTierService(
		log,
		tierService,
		doitEmployeesService,
	)

	return &AnalyticsAlerts{
		log,
		alertsService,
		alertTierService,
	}
}

// UpdateAlertSharingHandler ShareAlerts updates alert collaborators to share with users.
func (h *AnalyticsAlerts) UpdateAlertSharingHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	alertID := ctx.Param("alertID")
	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"alertId":              alertID,
	})

	var body service.ShareAlertRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if alertID == "" {
		return web.NewRequestError(service.ErrNoAlertID, http.StatusBadRequest)
	}

	if len(body.Collaborators) == 0 {
		return web.NewRequestError(service.ErrNoCollaborators, http.StatusBadRequest)
	}

	switch err := h.service.ShareAlert(ctx, body.Collaborators, body.PublicAccess, alertID, email, userID, customerID); true {
	case err == service.ErrNoAuthorization:
		return web.NewRequestError(err, http.StatusForbidden)
	case err != nil:
		return web.NewRequestError(err, http.StatusInternalServerError)
	default:
		return web.Respond(ctx, nil, http.StatusOK)
	}
}

// RefreshAlerts creates a worker task per alert that checks if user should be alerted.
func (h *AnalyticsAlerts) RefreshAlerts(ctx *gin.Context) error {
	err := h.service.RefreshAlerts(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// RefreshAlerts worker that checks if a specific alert needs to be alerted.
func (h *AnalyticsAlerts) RefreshAlert(ctx *gin.Context) error {
	alertID := ctx.Param("alertID")

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginAlerts)

	if err := h.service.RefreshAlert(ctx, alertID); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// SendEmails creates a worker task per customer to send alert emails.
func (h *AnalyticsAlerts) SendEmails(ctx *gin.Context) error {
	if err := h.service.SendEmails(ctx); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// SendEmailsToCustomer worker that sends alert emails to customer's users.
func (h *AnalyticsAlerts) SendEmailsToCustomer(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("invalid customer id"), http.StatusBadRequest)
	}

	if err := h.service.SendEmailsToCustomer(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsAlerts) DeleteAlert(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	alertID := ctx.Param("alertID")
	email := ctx.GetString(common.CtxKeys.Email)

	return h.deleteAlert(ctx, customerID, alertID, email)
}

func (h *AnalyticsAlerts) deleteAlert(ctx *gin.Context, customerID, alertID, email string) error {
	if alertID == "" {
		return web.NewRequestError(domain.ErrMissingAlertID, http.StatusBadRequest)
	}

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"alertId":              alertID,
	})

	if err := h.service.DeleteAlert(ctx, customerID, email, alertID); err != nil {
		switch err {
		case domain.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case domain.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case domain.ErrorUnAuthorized:
			return web.NewRequestError(err, http.StatusUnauthorized)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// DeleteMany deletes the list of alert ids from the request in a transaction
func (h *AnalyticsAlerts) DeleteManyHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	email := ctx.GetString(common.CtxKeys.Email)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"action":               "deleteManyAlerts",
	})

	var body struct {
		IDs []string `json:"ids"`
	}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.IDs) == 0 {
		return web.NewRequestError(errors.New("no alert ids provided"), http.StatusBadRequest)
	}

	if err := h.service.DeleteMany(ctx, email, body.IDs); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

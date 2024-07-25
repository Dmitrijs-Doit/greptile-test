package handlers

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func (h *AnalyticsAlerts) ExternalAPIGetAlert(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	alertID := ctx.Param("id")

	if alertID == "" {
		return web.NewRequestError(domain.ErrMissingAlertID, http.StatusBadRequest)
	}

	l.SetLabels(map[string]string{
		"alertId": alertID,
	})

	a, err := h.service.GetAlert(ctx, alertID)
	if err != nil {
		l.Errorf("Failed to get alert %s \n Error: %v", alertID, err)

		if err == doitFirestore.ErrNotFound {
			return web.NewRequestError(domain.ErrNotFound, http.StatusNotFound)
		}

		return web.NewRequestError(domain.ErrGetAlert, http.StatusInternalServerError)
	}

	return web.Respond(ctx, a, http.StatusOK)
}

func (h *AnalyticsAlerts) ExternalAPIListAlerts(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString(common.CtxKeys.Email)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	r := domain.AlertRequestData{
		SortBy:     ctx.Request.URL.Query().Get("sortBy"),
		SortOrder:  ctx.Request.URL.Query().Get("sortOrder"),
		MaxResults: ctx.Request.URL.Query().Get("maxResults"),
		PageToken:  ctx.Request.URL.Query().Get("pageToken"),
		Filter:     ctx.Request.URL.Query().Get("filter"),
		Email:      email,
		CustomerID: customerID,
	}

	reqData, err := customerapi.NewAPIRequest(&r)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	alerts, err := h.service.ListAlerts(ctx, service.ExternalAPIListArgsReq{
		CustomerID:    customerID,
		Email:         email,
		SortBy:        reqData.SortBy,
		SortOrder:     reqData.SortOrder,
		Filters:       reqData.Filters,
		MaxResults:    reqData.MaxResults,
		NextPageToken: reqData.NextPageToken,
	})

	if err != nil {
		l.Errorf("Failed to get list of alerts\n Error: %v", err)
		return web.NewRequestError(domain.ErrGetAlert, http.StatusBadRequest)
	}

	return web.Respond(ctx, alerts, http.StatusOK)
}

func (h *AnalyticsAlerts) ExternalAPIDeleteAlert(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	alertID := ctx.Param("id")
	emptyJSON := struct{}{}

	if alertID == "" {
		return web.NewRequestError(domain.ErrMissingAlertID, http.StatusBadRequest)
	}

	email := ctx.GetString("email")

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

	return web.Respond(ctx, emptyJSON, http.StatusOK)
}

func (h *AnalyticsAlerts) ExternalAPICreateAlert(ctx *gin.Context) error {
	email := ctx.GetString(common.CtxKeys.Email)
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	isDoitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	accessDeniedErr, err := h.alertTierService.CheckAccessToAlerts(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	var alertRequest service.AlertRequest

	if err := ctx.ShouldBindJSON(&alertRequest); err != nil {
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": errormsg.MapTagValidationErrors(err, false)})
		return nil
	}

	resp := h.service.CreateAlert(ctx, service.ExternalAPICreateUpdateArgsReq{
		AlertRequest:   &alertRequest,
		CustomerID:     customerID,
		UserID:         userID,
		Email:          email,
		IsDoitEmployee: isDoitEmployee,
	})

	if resp.Error != nil {
		if errors.Is(resp.Error, domain.ErrValidationErrors) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": resp.ValidationErrors})
			return nil
		}

		return web.NewRequestError(resp.Error, http.StatusInternalServerError)
	}

	return web.Respond(ctx, resp.Alert, http.StatusCreated)
}

func (h *AnalyticsAlerts) ExternalAPIUpdateAlert(ctx *gin.Context) error {
	alertID := ctx.Param("id")

	if alertID == "" {
		ctx.Abort()
		return web.NewRequestError(domain.ErrMissingAlertID, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	isDoitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	accessDeniedErr, err := h.alertTierService.CheckAccessToAlerts(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	var alertRequest service.AlertRequest

	if err := ctx.ShouldBindJSON(&alertRequest); err != nil {
		errors := errormsg.MapTagValidationErrors(err, true)
		if len(errors) > 0 {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": errors})
			return nil
		}
	}

	if (reflect.DeepEqual(alertRequest, service.AlertRequest{})) {
		return web.NewRequestError(domain.ErrEmptyBody, http.StatusBadRequest)
	}

	resp := h.service.UpdateAlert(ctx, alertID, service.ExternalAPICreateUpdateArgsReq{
		AlertRequest:   &alertRequest,
		CustomerID:     customerID,
		UserID:         userID,
		Email:          email,
		IsDoitEmployee: isDoitEmployee,
	})

	if resp.Error != nil {
		switch {
		case errors.Is(resp.Error, domain.ErrValidationErrors):
			ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"errors": resp.ValidationErrors})
			return nil
		case errors.Is(resp.Error, domain.ErrForbidden):
			return web.NewRequestError(domain.ErrorUnAuthorized, http.StatusForbidden)
		case errors.Is(resp.Error, doitFirestore.ErrNotFound):
			return web.NewRequestError(domain.ErrNotFound, http.StatusNotFound)
		default:
			return web.NewRequestError(resp.Error, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, resp.Alert, http.StatusOK)
}

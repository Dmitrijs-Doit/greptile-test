package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func (h *Attributions) ListAttributionsExternalHandler(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString(common.CtxKeys.Email)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	r := attribution.AttributionsListRequestData{}
	r.MaxResults = ctx.Request.URL.Query().Get("maxResults")
	r.PageToken = ctx.Request.URL.Query().Get("pageToken")
	r.Filter = ctx.Request.URL.Query().Get("filter")
	r.SortBy = ctx.Request.URL.Query().Get("sortBy")
	r.SortOrder = ctx.Request.URL.Query().Get("sortOrder")
	r.CustomerID = customerID
	r.Email = email
	reqData, err := customerapi.NewAPIRequest(&r)

	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	attributionsList, err := h.service.ListAttributions(ctx, reqData)
	if err != nil {
		switch {
		case errors.Is(err, customerapi.ErrQueryParam):
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	ctx.JSON(http.StatusOK, attributionsList)

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Attributions) GetAttributionExternalHandler(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	attributionID := ctx.Param("id")
	if attributionID == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	accessDeniedErr, err := h.attributionTierService.CheckAccessToAttributionID(ctx, customerID, attributionID)
	if err != nil {
		if errors.Is(err, attribution.ErrNotFound) {
			return web.NewRequestError(err, http.StatusNotFound)
		}

		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	return h.GetAttribution(ctx, customerID)
}

func (h *Attributions) DeleteAttributionExternalHandler(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	return h.DeleteAttribution(ctx, customerID)
}

func (h *Attributions) CreateAttributionExternalHandler(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	accessDeniedErr, err := h.attributionTierService.CheckAccessToCustomAttribution(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	return h.CreateAttribution(ctx, customerID)
}

func (h *Attributions) UpdateAttributionExternalHandler(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	accessDeniedErr, err := h.attributionTierService.CheckAccessToCustomAttribution(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	return h.UpdateAttribution(ctx, customerID)
}

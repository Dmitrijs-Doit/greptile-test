package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AnalyticsMetadata struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	service        iface.MetadataIface
}

type DimensionsExternalAPIGetResponse iface.ExternalAPIGetRes

func NewAnalyticsMetadata(ctx context.Context, log logger.Provider, conn *connection.Connection) *AnalyticsMetadata {
	svc := service.NewMetadataService(ctx, log, conn)

	return &AnalyticsMetadata{
		log,
		conn,
		svc,
	}
}

func (h *AnalyticsMetadata) ExternalAPIListDimensions(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString(common.CtxKeys.Email)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	r := domain.DimensionsListRequestData{
		MaxResults: ctx.Query("maxResults"),
		PageToken:  ctx.Query("pageToken"),
		Filter:     ctx.Query("filter"),
		SortOrder:  ctx.Query("sortOrder"),
	}

	sortBy := ctx.Query("sortBy")
	if sortBy == "" {
		sortBy = "id"
	}

	r.SortBy = sortBy

	reqData, err := customerapi.NewAPIRequest(&r)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	res, err := h.service.ExternalAPIListWithFilters(iface.ExternalAPIListArgs{
		Ctx:             ctx,
		CustomerID:      customerID,
		IsDoitEmployee:  ctx.GetBool(common.CtxKeys.DoitEmployee),
		UserID:          ctx.GetString(common.CtxKeys.UserID),
		UserEmail:       email,
		OmitGkeTypes:    true,
		OmitByTimestamp: true,
	}, reqData)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *AnalyticsMetadata) ExternalAPIGetDimensions(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString(common.CtxKeys.Email)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	idFilter := ctx.Query("id")

	typeFilter := ctx.Query("type")
	if typeFilter == "" {
		return web.NewRequestError(metadata.ErrHandlerGetInvalidFilters, http.StatusBadRequest)
	}

	if idFilter == "" {
		return web.NewRequestError(metadata.ErrHandlerGetInvalidFilters, http.StatusBadRequest)
	}

	res, err := h.service.ExternalAPIGet(iface.ExternalAPIGetArgs{
		Ctx:            ctx,
		CustomerID:     customerID,
		IsDoitEmployee: ctx.GetBool(common.CtxKeys.DoitEmployee),
		UserID:         ctx.GetString(common.CtxKeys.UserID),
		UserEmail:      email,
		KeyFilter:      idFilter,
		TypeFilter:     typeFilter,
	})

	if err != nil {
		switch err {
		case metadata.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *AnalyticsMetadata) AttributionGroupsMetadata(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")

	md, err := h.service.AttributionGroupsMetadata(ctx, customerID, email)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, md, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateAzureAllCustomersMetadata(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataAzure)

	errs, err := h.service.UpdateAzureAllCustomersMetadata(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(errs) > 0 {
		l.Errorf(FailedToCreateMetadataErrFormat, errs)
		return web.Respond(ctx, errs, http.StatusMultiStatus)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateAzureCustomerMetadata(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataAzure)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	err := h.service.UpdateAzureCustomerMetadata(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateBQLensAllCustomersMetadata(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataBqLens)

	errs, err := h.service.UpdateBQLensAllCustomersMetadata(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(errs) > 0 {
		l.Errorf(FailedToCreateMetadataErrFormat, errs)
		return web.Respond(ctx, errs, http.StatusMultiStatus)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateBQLensCustomerMetadata(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataBqLens)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	err := h.service.UpdateBQLensCustomerMetadata(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateGCPBillingAccountMetadata(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataGcp)

	var body metadata.MetadataUpdateInput
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	billingAccountID := ctx.Param("billingAccountID")

	err := h.service.UpdateGCPBillingAccountMetadata(ctx, body.AssetID, billingAccountID, nil)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateAWSAllCustomersMetadata(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataAws)

	errs, err := h.service.UpdateAWSAllCustomersMetadata(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(errs) > 0 {
		l.Errorf(FailedToCreateMetadataErrFormat, errs)
		return web.Respond(ctx, errs, http.StatusMultiStatus)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateAWSCustomerMetadata(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataAws)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(ErrMissingCustomerID, http.StatusBadRequest)
	}

	if err := h.service.UpdateAWSCustomerMetadata(ctx, customerID, nil); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsMetadata) UpdateDataHubMetadata(ctx *gin.Context) error {
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginMetadataDataHub)

	if err := h.service.UpdateDataHubMetadata(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

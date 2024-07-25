package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	attrGroupDeleteValidation "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups/attributiongroupdeletevalidation"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service"
	attributionGroupTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/attributiongrouptier"
	attributionGroupIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/attributiongrouptier/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service/iface"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attributionsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier"
	attributionServiceface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type AnalyticsAttributionGroups struct {
	loggerProvider              logger.Provider
	service                     iface.AttributionGroupsIface
	attributionService          attributionServiceface.AttributionsIface
	attributionGroupTierService attributionGroupIface.AttributionGroupTierService
}

type ShareAttributionGroupRequest struct {
	Collaborators []collab.Collaborator `json:"collaborators"`
	PublicAccess  *collab.PublicAccess  `json:"public"`
}

type DeleteAttributionGroupsRequest struct {
	AttributionGroupIDs []string `json:"attributionGroupIDs"`
}

type CreateAttributionGroupResponse struct {
	ID string `json:"id"`
}

func NewCreateAttributionGroupResponse(attributionGroupID string) *CreateAttributionGroupResponse {
	return &CreateAttributionGroupResponse{
		ID: attributionGroupID,
	}
}

func NewAnalyticsAttributionGroups(ctx context.Context, log logger.Provider, conn *connection.Connection) *AnalyticsAttributionGroups {
	s := service.NewAttributionGroupsService(ctx, log, conn)

	attrService := attributionsService.NewAttributionsService(ctx, log, conn)

	attributionGroupDal := dal.NewAttributionGroupsFirestoreWithClient(conn.Firestore)

	tierService := tier.NewTiersService(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	attributionDal := attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	attributionTierService := attributiontier.NewAttributionTierService(
		log,
		attributionDal,
		tierService,
		doitEmployeesService,
	)

	attributionGroupTierService := attributionGroupTier.NewAttributionGroupTierService(
		log,
		attributionGroupDal,
		tierService,
		attributionTierService,
		doitEmployeesService,
	)

	return &AnalyticsAttributionGroups{
		log,
		s,
		attrService,
		attributionGroupTierService,
	}
}

// ShareAttributionGroup updates alert collaborators to share with users.
func (h *AnalyticsAttributionGroups) ShareAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	attributionGroupID := ctx.Param("attributionGroupID")
	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)
	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"attributionGroupId":   attributionGroupID,
	})

	accessDeniedErr, err := h.attributionGroupTierService.CheckAccessToCustomAttributionGroup(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	var body ShareAttributionGroupRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.Collaborators) == 0 {
		return web.NewRequestError(attributiongroups.ErrNoCollaborators, http.StatusBadRequest)
	}

	err = h.service.ShareAttributionGroup(ctx, body.Collaborators, body.PublicAccess, attributionGroupID, userID, email)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsAttributionGroups) CreateAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	return h.createAttributionGroup(ctx, customerID, false)
}

func (h *AnalyticsAttributionGroups) createAttributionGroup(
	ctx *gin.Context,
	customerID string,
	isExternalAPI bool,
) error {
	email := ctx.GetString("email")
	fromDraftAttributions := ctx.Query("fromDraftAttributions") == "true"

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var attributionGroupRequest attributiongroups.AttributionGroupRequest

	if err := ctx.ShouldBindJSON(&attributionGroupRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if fromDraftAttributions {
		var attributionsToUpdate []*attribution.Attribution

		for _, attrID := range attributionGroupRequest.Attributions {
			draft := false
			attributionsToUpdate = append(attributionsToUpdate, &attribution.Attribution{ID: attrID, Draft: &draft, ExpireBy: nil})
		}

		_, err := h.attributionService.UpdateAttributions(ctx, customerID, attributionsToUpdate, ctx.GetString(common.CtxKeys.UserID))
		if err != nil {
			switch err {
			case attributionsService.ErrForbidden:
				return web.NewRequestError(err, http.StatusForbidden)
			case attributionsService.ErrNotFound:
				return web.NewRequestError(err, http.StatusNotFound)
			case query.ErrInvalidChars:
				return web.NewRequestError(err, http.StatusBadRequest)
			default:
				return web.NewRequestError(err, http.StatusInternalServerError)
			}
		}
	}

	accessDeniedErr, err := h.attributionGroupTierService.CheckAccessToExternalAttributionGroup(
		ctx,
		customerID,
		attributionGroupRequest.Attributions,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		if isExternalAPI {
			return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
		}

		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	attributionGroupID, err := h.service.CreateAttributionGroup(ctx, customerID, email, &attributionGroupRequest)
	if err != nil {
		switch err {
		case attributiongroups.ErrForbiddenAttribution:
			return web.NewRequestError(err, http.StatusForbidden)
		case attribution.ErrNotFound:
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response := NewCreateAttributionGroupResponse(attributionGroupID)

	return web.Respond(ctx, response, http.StatusCreated)
}

func (h *AnalyticsAttributionGroups) UpdateAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	attributionGroupID := ctx.Param("attributionGroupID")

	return h.updateAttributionGroup(ctx, customerID, attributionGroupID, false)
}

func (h *AnalyticsAttributionGroups) updateAttributionGroup(
	ctx *gin.Context,
	customerID string,
	attributionGroupID string,
	isExternalApi bool,
) error {
	email := ctx.GetString("email")
	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"attributionGroupId":   attributionGroupID,
	})

	var attributionGroupUpdateRequest attributiongroups.AttributionGroupUpdateRequest
	if err := ctx.ShouldBindJSON(&attributionGroupUpdateRequest); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	accessDeniedErr, err := h.attributionGroupTierService.CheckAccessToExternalAttributionGroup(
		ctx,
		customerID,
		attributionGroupUpdateRequest.Attributions,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		if isExternalApi {
			return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
		}

		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	if err := h.service.UpdateAttributionGroup(ctx, customerID, attributionGroupID, email, &attributionGroupUpdateRequest); err != nil {
		switch err {
		case attributiongroups.ErrForbidden, attributiongroups.ErrForbiddenAttribution:
			return web.NewRequestError(err, http.StatusForbidden)
		case attributiongroups.ErrNotFound, attribution.ErrNotFound:
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsAttributionGroups) deleteAttributionGroup(ctx *gin.Context, customerID, attributionGroupID string) error {
	email := ctx.GetString("email")
	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
		"attributionGroupId":   attributionGroupID,
	})

	if attributionGroupID == "" {
		return web.NewRequestError(attributiongroups.ErrNoAttributionGroupID, http.StatusBadRequest)
	}

	blockingResources, err := h.service.DeleteAttributionGroup(ctx, customerID, email, attributionGroupID)
	if err != nil {
		if errors.Is(err, attributiongroups.ErrForbidden) {
			return web.NewRequestError(err, http.StatusForbidden)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(blockingResources) > 0 {
		resources := make(map[domainResource.ResourceType][]domainResource.Resource)
		resources[domainResource.Reports] = blockingResources

		response := []attrGroupDeleteValidation.AttributionGroupDeleteValidation{
			{
				ID:        attributionGroupID,
				Error:     attributiongroups.ErrAttrGroupIsInUse.Error(),
				Resources: resources,
			},
		}

		return web.Respond(ctx, response, http.StatusConflict)
	}

	emptyJSON := struct{}{}

	return web.Respond(ctx, emptyJSON, http.StatusOK)
}

func (h *AnalyticsAttributionGroups) DeleteAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	attributionGroupID := ctx.Param("attributionGroupID")

	return h.deleteAttributionGroup(ctx, customerID, attributionGroupID)
}

func (h *AnalyticsAttributionGroups) DeleteAttributionGroups(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	email := ctx.GetString("email")

	var body DeleteAttributionGroupsRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if len(body.AttributionGroupIDs) == 0 {
		return web.NewRequestError(attributiongroups.ErrNoAttributionGroupID, http.StatusBadRequest)
	}

	var errs []error

	var attrGroupDeleteValidations []attrGroupDeleteValidation.AttributionGroupDeleteValidation

	for _, attributionGroupID := range body.AttributionGroupIDs {
		blockingResources, err := h.service.DeleteAttributionGroup(ctx, customerID, email, attributionGroupID)
		if err != nil {
			errs = append(errs, err)
		}

		if len(blockingResources) > 0 {
			resources := make(map[domainResource.ResourceType][]domainResource.Resource)
			resources[domainResource.Reports] = blockingResources

			attrGroupDeleteValidations = append(
				attrGroupDeleteValidations,
				attrGroupDeleteValidation.AttributionGroupDeleteValidation{
					ID:        attributionGroupID,
					Error:     attributiongroups.ErrAttrGroupIsInUse.Error(),
					Resources: resources,
				},
			)
		}
	}

	if len(errs) > 0 {
		var err *multierror.Error

		for _, e := range errs {
			err = multierror.Append(err, e)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if len(attrGroupDeleteValidations) > 0 {
		return web.Respond(ctx, attrGroupDeleteValidations, http.StatusConflict)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsAttributionGroups) ExternalAPIDeleteAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	attributionGroupID := ctx.Param("id")

	return h.deleteAttributionGroup(ctx, customerID, attributionGroupID)
}

// APIListAttributionGroups lists all attributions groups for a user. Used by customers in external API
func (h *AnalyticsAttributionGroups) ExternalAPIListAttributionGroups(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)

	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString("email")

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	r := attributiongroups.AttributionGroupsRequestData{
		MaxResults: ctx.Query("maxResults"),
		PageToken:  ctx.Query("pageToken"),
		SortBy:     ctx.Query("sortBy"),
		SortOrder:  ctx.Query("sortOrder"),
		CustomerID: customerID,
		Email:      email,
	}

	parsedRequest, err := customerapi.NewAPIRequest(r)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	aGroups, err := h.service.ListAttributionGroupsExternal(ctx, parsedRequest)
	if err != nil {
		switch {
		case errors.Is(err, customerapi.ErrQueryParam):
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, aGroups, http.StatusOK)
}

func (h *AnalyticsAttributionGroups) ExternalAPIGetAttributionGroup(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	email := ctx.GetString("email")
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)

	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	attributionGroupID := ctx.Param("id")
	attributionGroup, err := h.service.GetAttributionGroupExternal(ctx, attributionGroupID)

	if err != nil {
		return web.NewRequestError(err, http.StatusNotFound)
	}

	if attributionGroup.Type == "custom" && (attributionGroup.Customer == nil || attributionGroup.Customer.ID != customerID) {
		return web.NewRequestError(errors.New("unauthorized operation"), http.StatusUnauthorized)
	}

	accessDeniedErr, err := h.attributionGroupTierService.CheckAccessToAttributionGroupID(
		ctx,
		customerID,
		attributionGroupID,
	)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr.PublicError(), http.StatusForbidden)
	}

	ctx.JSON(http.StatusOK, attributionGroup)

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AnalyticsAttributionGroups) ExternalAPICreateAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	return h.createAttributionGroup(ctx, customerID, true)
}

func (h *AnalyticsAttributionGroups) ExternalAPIUpdateAttributionGroup(ctx *gin.Context) error {
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	attributionGroupID := ctx.Param("id")

	return h.updateAttributionGroup(ctx, customerID, attributionGroupID, true)
}

func (h *AnalyticsAttributionGroups) SyncEntityInvoiceAttributions(ctx *gin.Context) error {
	entityID := ctx.Param("entityID")
	customerID := ctx.Param("customerID")

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEntityID:   entityID,
	})

	err := h.service.SyncEntityInvoiceAttributions(ctx, service.SyncEntityInvoiceAttributionsRequest{
		CustomerID: customerID,
		EntityID:   entityID,
	})
	if err != nil {
		switch err {
		case attributiongroups.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

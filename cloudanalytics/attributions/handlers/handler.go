package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	attributionDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier"
	attributionTierServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	tier "github.com/doitintl/tiers/service"
)

type Attributions struct {
	loggerProvider         logger.Provider
	service                iface.AttributionsIface
	attributionTierService attributionTierServiceIface.AttributionTierService
}

const (
	//amount of attributions that can be deleted at once per request (to prevent abuse)
	maxAttributionsToDelete = 30
)

func NewAttributions(ctx context.Context, log logger.Provider, conn *connection.Connection) *Attributions {
	s := service.NewAttributionsService(ctx, log, conn)

	tierService := tier.NewTiersService(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	attributionDAL := attributionDal.NewAttributionsFirestoreWithClient(conn.Firestore)

	attributionTierService := attributiontier.NewAttributionTierService(
		log,
		attributionDAL,
		tierService,
		doitEmployeesService,
	)

	return &Attributions{
		log,
		s,
		attributionTierService,
	}
}

func (h *Attributions) DeleteAttributions(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var body service.DeleteAttributionsRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	body.UserID = ctx.GetString("userId")
	body.Email = ctx.GetString("email")
	body.CustomerID = customerID

	if len(body.AttributionsIDs) == 0 {
		return web.NewRequestError(service.ErrNotFound, http.StatusBadRequest)
	}

	if len(body.AttributionsIDs) > maxAttributionsToDelete {
		return web.NewRequestError(ErrTooManyAttributionsToDelete, http.StatusBadRequest)
	}

	validations, err := h.service.DeleteAttributions(ctx, &body)
	if err != nil {
		return web.RespondError(ctx, err)
	}

	return web.Respond(ctx, validations, http.StatusOK)
}

func (h *Attributions) UpdateAttributionSharingHandler(ctx *gin.Context) error {
	var body service.ShareAttributionRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if body.AttributionID == "" || len(body.Collaborators) == 0 {
		return web.NewRequestError(service.ErrBadRequest, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)

	if err := h.service.ShareAttributions(ctx, &body, email, userID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Attributions) GetAttributionHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

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

func (h *Attributions) GetAttribution(ctx *gin.Context, customerID string) error {
	idParam := ctx.Param("id")
	if idParam == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	isDoitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)

	if !isDoitEmployee && email == "" {
		return web.NewRequestError(ErrEmailMissing, http.StatusBadRequest)
	}

	attr, err := h.service.GetAttribution(ctx, idParam, isDoitEmployee, customerID, email)
	if err != nil {
		switch err {
		case attribution.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case service.ErrWrongCustomer, service.ErrMissingPermissions:
			return web.NewRequestError(err, http.StatusForbidden)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, attr, http.StatusOK)
}

func (h *Attributions) UpdateAttribution(ctx *gin.Context, customerID string) error {
	attributionID := ctx.Param("id")
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		"attributionId":        attributionID,
	})

	var attribution *attribution.AttributionAPI
	if err := ctx.ShouldBindJSON(&attribution); err != nil {
		return web.NewRequestError(ErrNoAttributionFieldInRequest, http.StatusBadRequest)
	}

	if attribution.Filters != nil {
		if err := validateComponents(attribution.Filters); err != nil {
			return web.NewRequestError(err, http.StatusBadRequest)
		}
	}

	attribution.ID = attributionID

	attr, err := toInternalAttribution(attribution)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	updatedAttr, err := h.service.UpdateAttribution(ctx, &service.UpdateAttributionRequest{
		CustomerID:  customerID,
		Attribution: *attr,
		UserID:      userID,
	})
	if err != nil {
		switch err {
		case service.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case service.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case query.ErrInvalidChars:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, updatedAttr, http.StatusOK)
}

func (h *Attributions) UpdateAttributionInternalHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	accessDeniedErr, err := h.attributionTierService.CheckAccessToCustomAttribution(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	return h.UpdateAttributionInternal(ctx, customerID)
}

func (h *Attributions) UpdateAttributionInternal(ctx *gin.Context, customerID string) error {
	attributionID := ctx.Param("id")
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		"attributionId":        attributionID,
	})

	accessDeniedErr, err := h.attributionTierService.CheckAccessToCustomAttribution(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	var attribution *attribution.Attribution
	if err := ctx.ShouldBindJSON(&attribution); err != nil {
		return web.NewRequestError(ErrNoAttributionFieldInRequest, http.StatusBadRequest)
	}

	if attribution.Filters != nil {
		if err := validateFilters(attribution.Filters); err != nil {
			return web.NewRequestError(err, http.StatusBadRequest)
		}
	}

	attribution.ID = attributionID

	updatedAttr, err := h.service.UpdateAttribution(ctx, &service.UpdateAttributionRequest{
		CustomerID:  customerID,
		Attribution: *attribution,
		UserID:      userID,
	})
	if err != nil {
		switch err {
		case service.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case service.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case query.ErrInvalidChars:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, updatedAttr, http.StatusOK)
}

func (h *Attributions) CreateAttribution(ctx *gin.Context, customerID string) error {
	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var attribution *attribution.AttributionAPI
	if err := ctx.ShouldBindJSON(&attribution); err != nil {
		return web.NewRequestError(ErrNoAttributionInRequest, http.StatusBadRequest)
	}

	if err := validateAttribution(attribution); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	attr, err := toInternalAttribution(attribution)
	if err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	att, err := h.service.CreateAttribution(ctx, &service.CreateAttributionRequest{
		CustomerID:  customerID,
		Attribution: *attr,
		UserID:      userID,
		Email:       email,
	})

	if err != nil {
		switch err {
		case service.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case service.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, att, http.StatusCreated)
}

func (h *Attributions) CreateAttributionInternalHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	accessDeniedErr, err := h.attributionTierService.CheckAccessToCustomAttribution(ctx, customerID)
	if err != nil {
		return ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	if accessDeniedErr != nil {
		return web.Respond(ctx, accessDeniedErr, http.StatusForbidden)
	}

	return h.CreateAttributionInternal(ctx, customerID)
}

func (h *Attributions) CreateAttributionInternal(ctx *gin.Context, customerID string) error {
	email := ctx.GetString(common.CtxKeys.Email)
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	var attribution *attribution.Attribution
	if err := ctx.ShouldBindJSON(&attribution); err != nil {
		return web.NewRequestError(ErrNoAttributionInRequest, http.StatusBadRequest)
	}

	if err := validateAttributionInternal(attribution); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	att, err := h.service.CreateAttribution(ctx, &service.CreateAttributionRequest{
		CustomerID:  customerID,
		Attribution: *attribution,
		UserID:      userID,
		Email:       email,
	})

	if err != nil {
		switch err {
		case service.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case service.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, att, http.StatusCreated)
}

func (h *Attributions) DeleteAttributionHandler(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	return h.DeleteAttribution(ctx, customerID)
}

func (h *Attributions) DeleteAttribution(ctx *gin.Context, customerID string) error {
	attributionID := ctx.Param("id")
	if attributionID == "" {
		return web.NewRequestError(web.ErrNotFound, http.StatusBadRequest)
	}

	email := ctx.GetString(common.CtxKeys.Email)
	doitEmployee := ctx.GetBool(common.CtxKeys.DoitEmployee)
	userID := ctx.GetString(common.CtxKeys.UserID)

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail:      email,
		logger.LabelCustomerID: customerID,
	})

	if !doitEmployee && userID == "" {
		return web.NewRequestError(ErrUserIDMissing, http.StatusBadRequest)
	}

	requestData := service.DeleteAttributionsRequest{
		AttributionsIDs: []string{attributionID},
		CustomerID:      customerID,
		UserID:          userID,
		Email:           email,
	}

	validations, err := h.service.DeleteAttributions(ctx, &requestData)
	if err != nil {
		return web.RespondError(ctx, err)
	}

	return web.Respond(ctx, validations, http.StatusOK)
}

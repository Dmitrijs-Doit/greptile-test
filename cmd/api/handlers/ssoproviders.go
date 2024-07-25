package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/sso"
)

type SsoProviders struct {
	*logger.Logging
	*sso.ProviderService
}

func NewSsoProviders(log *logger.Logging, conn *connection.Connection) *SsoProviders {
	ssoService := sso.NewSSOProviderService(log, conn)

	return &SsoProviders{
		log,
		ssoService,
	}
}

func (h *SsoProviders) GetAllProvidersHandler(ctx *gin.Context) error {
	if authErr := h.authorize(ctx); authErr != nil {
		return web.NewRequestError(authErr, http.StatusUnauthorized)
	}

	customerID := ctx.Param("customerID")

	result, err := h.GetAllProviders(ctx, customerID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, result, http.StatusOK)
}

func (h *SsoProviders) CreateProviderHandler(ctx *gin.Context) error {
	if authErr := h.authorize(ctx); authErr != nil {
		return web.NewRequestError(authErr, http.StatusUnauthorized)
	}

	customerID := ctx.Param("customerID")

	var providerConfig sso.ProviderConfig

	if err := ctx.ShouldBindJSON(&providerConfig); err != nil {
		return err
	}

	if err := sso.ValidateSAMLConfigOnCreate(providerConfig.SAML); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := sso.ValidateOIDCConfigOnCreate(providerConfig.OIDC); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	config, err := h.CreateProvider(ctx, customerID, &providerConfig)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, config, http.StatusOK)
}

func (h *SsoProviders) UpdateProviderHandler(ctx *gin.Context) error {
	if authErr := h.authorize(ctx); authErr != nil {
		return web.NewRequestError(authErr, http.StatusUnauthorized)
	}

	customerID := ctx.Param("customerID")

	var providerConfig sso.ProviderConfig

	if err := ctx.ShouldBindJSON(&providerConfig); err != nil {
		return err
	}

	if err := sso.ValidateSAMLConfigOnUpdate(providerConfig.SAML); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	config, err := h.UpdateProvider(ctx, customerID, &providerConfig)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, config, http.StatusOK)
}

func (h *SsoProviders) authorize(ctx *gin.Context) error {
	return permissionsAuthorizer(ctx, h.Connection, []common.Permission{common.PermissionUsersManager})
}

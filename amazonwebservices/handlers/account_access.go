package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/doitintl/googleadmin"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type AccountAccess struct {
	accessor *amazonwebservices.AWSAccountAccessor
}

func NewAccountAccess(log logger.Provider, conn *connection.Connection) *AccountAccess {
	ctx := context.Background()

	goolgeAdmin, err := googleadmin.NewGoogleAdmin(ctx, common.ProjectID)
	if err != nil {
		panic(err)
	}

	// NewAWSAuth can only be initialized with gcp service account env credentials
	accessor, _ := amazonwebservices.NewAWSAccountAccessor(ctx, conn.Firestore(ctx), goolgeAdmin)

	return &AccountAccess{accessor}
}

func (h *AccountAccess) GetCreds(ctx *gin.Context) error {
	if h.accessor == nil {
		return web.NewRequestError(errors.New("accessor failed to init"), http.StatusInternalServerError)
	}

	var req amazonwebservices.GetCredsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	creds, err := h.accessor.GetCreds(ctx, req)
	if err != nil {
		if errors.Is(err, amazonwebservices.ErrUnauthorized) {
			return web.NewRequestError(err, http.StatusUnauthorized)
		}

		if errors.Is(err, amazonwebservices.ErrBadRequest) {
			return web.NewRequestError(err, http.StatusBadRequest)
		}

		return web.NewRequestError(fmt.Errorf(""), http.StatusInternalServerError)
	}

	return web.Respond(ctx, creds, http.StatusOK)
}

func (h *AccountAccess) GetRoles(ctx *gin.Context) error {
	if h.accessor == nil {
		return web.NewRequestError(errors.New("accessor failed to init"), http.StatusInternalServerError)
	}

	customerID := ctx.Param("customerID")

	allowedRoles, err := h.accessor.GetRoles(ctx, customerID)
	if err != nil {
		return web.NewRequestError(fmt.Errorf(""), http.StatusInternalServerError)
	}

	return web.Respond(ctx, allowedRoles, http.StatusOK)
}

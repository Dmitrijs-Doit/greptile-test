package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type IAMOrganizations struct {
	*logger.Logging
	service *organizations.OrgsIAMService
}

func NewIAMOrganizations(log *logger.Logging, conn *connection.Connection) *IAMOrganizations {
	service := organizations.NewIAMOrganizationService(log, conn)

	return &IAMOrganizations{
		log,
		service,
	}
}

func (h *IAMOrganizations) DeleteIAMOrganizations(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	var body organizations.RemoveIAMOrgsRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	body.CustomerID = customerID
	body.UserID = ctx.GetString("userId")

	if body.OrgIDs == nil || len(body.OrgIDs) == 0 {
		return web.NewRequestError(organizations.ErrNotFound, http.StatusBadRequest)
	}

	if err := h.service.DeleteIAMOrgs(ctx, &body); err != nil {
		switch err {
		case organizations.ErrForbidden:
			return web.NewRequestError(err, http.StatusForbidden)
		case organizations.ErrNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

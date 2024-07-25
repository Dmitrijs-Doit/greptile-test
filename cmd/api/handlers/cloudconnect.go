package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CloudConnect struct {
	loggerProvider logger.Provider
	service        *cloudconnect.CloudConnectService
}

func NewCloudConnect(loggerProvider logger.Provider, conn *connection.Connection) *CloudConnect {
	service := cloudconnect.NewCloudConnectService(loggerProvider, conn)

	return &CloudConnect{
		loggerProvider,
		service,
	}
}

func (h *CloudConnect) AddGcpServiceAccount(ctx *gin.Context) error {
	h.service.AddGcpServiceAccount(ctx)

	return nil
}

func (h *CloudConnect) AddPartialGcpServiceAccount(ctx *gin.Context) error {
	var form cloudconnect.RequestServiceAccount
	if err := ctx.ShouldBind(&form); err != nil {
		return err
	}

	if err := h.service.AddPartialGcpServiceAccount(ctx, form); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *CloudConnect) CheckWorkloadIdentityFederationConnection(ctx *gin.Context) error {
	var payload cloudconnect.WorkloadIdentityFederationConnectionCheckRequest
	if err := ctx.ShouldBind(&payload); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	connectionStatus, err := h.service.CheckWorkloadIdentityFederationConnection(ctx, payload)
	if err != nil {
		return web.Respond(ctx, err.Error(), http.StatusInternalServerError)
	}

	connectionError := ""
	connectionErrorDescription := ""

	if connectionStatus.ConnectionDetails != nil {
		connectionError = connectionStatus.ConnectionDetails.Error.Status
		connectionErrorDescription = connectionStatus.ConnectionDetails.Error.Message
	}

	response := cloudconnect.WorkloadIdentityFederationConnectionCheckResponse{
		IsConnectionEstablished: connectionStatus.IsConnectionEstablished,
		Error:                   connectionError,
		ErrorDescription:        connectionErrorDescription,
	}

	return web.Respond(ctx, &response, http.StatusOK)
}

func RemoveGcpServiceAccount(ctx *gin.Context) error {
	cloudconnect.RemoveGcpServiceAccount(ctx)

	return nil
}

func AWSAddRoleHandler(ctx *gin.Context) error {
	cloudconnect.AWSAddRoleHandler(ctx)

	return nil
}

func GetMissingPermissions(ctx *gin.Context) error {
	cloudconnect.GetMissingPermissions(ctx)

	return nil
}

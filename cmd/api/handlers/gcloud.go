package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type GoogleCloud struct {
	loggerProvider logger.Provider
	service        *googlecloud.GoogleCloudService
}

func NewGoogleCloud(loggerProvider logger.Provider, conn *connection.Connection) *GoogleCloud {
	service := googlecloud.NewGoogleCloudService(loggerProvider, conn)

	return &GoogleCloud{
		loggerProvider,
		service,
	}
}

func (h *GoogleCloud) UpdateCustomerLimitGCP(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if err := h.service.UpdateCustomerLimit(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func ListOrganizationFolders(ctx *gin.Context) error {
	googlecloud.ListOrganizationFolders(ctx)

	return nil
}

func CreateSandbox(ctx *gin.Context) error {
	googlecloud.CreateSandbox(ctx)

	return nil
}

func CreateServiceAccountForCustomer(ctx *gin.Context) error {
	googlecloud.CreateServiceAccountForCustomer(ctx)

	return nil
}

func TransferProjects(ctx *gin.Context) error {
	googlecloud.TransferProjects(ctx)

	return nil
}

func CheckServiceAccountPermissions(ctx *gin.Context) error {
	googlecloud.CheckServiceAccountPermissions(ctx)

	return nil
}

func (h *GoogleCloud) UpdateCustomerRecommendations(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")

	if customerID == "" {
		// Task
		var taskBody googlecloud.TaskBody
		if err := ctx.ShouldBindJSON(&taskBody); err != nil {
			return web.NewRequestError(err, http.StatusInternalServerError)
		}

		customerID = taskBody.CustomerID
	}

	if err := h.service.UpdateCustomerRecommendations(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *GoogleCloud) ChangeMachineType(ctx *gin.Context) error {
	req := googlecloud.ReqBody{
		CustomerID:     ctx.Param("customerID"),
		IsDoitEmployee: ctx.GetBool("doitEmployee"),
		UserID:         ctx.GetString("userId"),
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response, err := h.service.ChangeMachineType(ctx, req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *GoogleCloud) GetInstanceStatus(ctx *gin.Context) error {
	req := googlecloud.ReqBody{
		CustomerID:     ctx.Param("customerID"),
		IsDoitEmployee: ctx.GetBool("doitEmployee"),
		UserID:         ctx.GetString("userId"),
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response, err := h.service.GetInstanceStatus(ctx, req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *GoogleCloud) StopInstance(ctx *gin.Context) error {
	req := googlecloud.ReqBody{
		CustomerID:     ctx.Param("customerID"),
		IsDoitEmployee: ctx.GetBool("doitEmployee"),
		UserID:         ctx.GetString("userId"),
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response, err := h.service.StopInstance(ctx, req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func (h *GoogleCloud) StartInstance(ctx *gin.Context) error {
	req := googlecloud.ReqBody{
		CustomerID:     ctx.Param("customerID"),
		IsDoitEmployee: ctx.GetBool("doitEmployee"),
		UserID:         ctx.GetString("userId"),
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	response, err := h.service.StartInstance(ctx, req)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, response, http.StatusOK)
}

func SetBillingAccountAdmins(ctx *gin.Context) error {
	googlecloud.SetBillingAccountAdmins(ctx)

	return nil
}

func SendBillingAccountInstructionsHandler(ctx *gin.Context) error {
	googlecloud.SendBillingAccountInstructionsHandler(ctx)

	return nil
}

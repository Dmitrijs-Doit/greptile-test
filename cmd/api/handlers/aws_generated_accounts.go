package handlers

import (
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/generatedaccounts"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type AwsGeneratedAccounts struct {
	awsGeneratedAccountsService generatedaccounts.IGeneratedAccountsService
}

func NewAwsGeneratedAccounts(log logger.Provider, conn *connection.Connection) *AwsGeneratedAccounts {
	awsGeneratedAccountsService, err := generatedaccounts.NewGeneratedAccountsService(log, conn)
	if err != nil {
		panic(err)
	}

	return &AwsGeneratedAccounts{awsGeneratedAccountsService: awsGeneratedAccountsService}
}

// CreateAwsAccounts creates AWS account docs in Firestore and schedules a command to sign up these accounts with AWS
func (h *AwsGeneratedAccounts) CreateAccountsBatch(ctx *gin.Context) error {
	if common.Production {
		return web.Respond(ctx, nil, http.StatusNotAcceptable)
	}

	var req generatedaccounts.CreateAccountsBatchRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.awsGeneratedAccountsService.CreateAccountsBatch(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

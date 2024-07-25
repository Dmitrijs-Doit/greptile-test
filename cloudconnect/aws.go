package cloudconnect

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/aws"
)

func AWSPermissionsHandler(ctx *gin.Context) {
	p, err := aws.NewPermissions(ctx)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer p.Close()

	customerID := ctx.Param("customerID")
	accountID := ctx.Param("accountID")

	err = p.UpdateAWSPermissions(ctx, customerID, accountID)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.Status(http.StatusOK)
}

func AWSAddRoleHandler(ctx *gin.Context) {
	p, err := aws.NewPermissions(ctx)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer p.Close()

	var form RequestARN

	err = ctx.ShouldBind(&form)
	if err != nil {
		AbortWithError(ctx, http.StatusBadRequest, err)
		return
	}

	req := aws.RoleRequest{
		CustomerID: ctx.Param("customerID"),
		Arn:        form.Arn,
	}

	if len(req.CustomerID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing customer id"))
		return
	}

	if len(req.Arn) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing arn"))
		return
	}

	err = p.AddRole(ctx, &req)
	if err != nil {
		switch err {
		case aws.ErrArnUnauthorized, aws.ErrArnNotValid, aws.ErrNotExistsInAssets:
			AbortWithError(ctx, http.StatusBadRequest, err)
			return
		case aws.ErrAccountAlreadyExists:
			AbortWithError(ctx, http.StatusConflict, err)
			return
		default:
			AbortWithError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	err = p.UpdateAWSPermissions(ctx, req.CustomerID, req.AccountID)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, errors.New("could not update aws permissions"))
		return
	}

	ctx.Status(http.StatusOK)
}

func AWSUpdateRoleHandler(ctx *gin.Context) {
	p, err := aws.NewPermissions(ctx)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer p.Close()

	var req aws.RoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		AbortWithError(ctx, http.StatusBadRequest, err)
		return
	}

	if len(req.AccountID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing account id"))
		return
	}

	if len(req.CustomerID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing external id"))
		return
	}

	if len(req.Arn) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing arn"))
		return
	}

	if len(req.StackID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing stack id"))
		return
	}

	err = p.UpdateRoleAndSendNotification(ctx, &req)
	if err != nil {
		switch err {
		case aws.ErrArnUnauthorized, aws.ErrArnNotValid, aws.ErrNotExistsInAssets:
			AbortWithError(ctx, http.StatusBadRequest, err)
			return
		case aws.ErrAccountAlreadyExists:
			AbortWithError(ctx, http.StatusConflict, err)
			return
		default:
			AbortWithError(ctx, http.StatusInternalServerError, err)
			return
		}
	}

	err = p.UpdateAWSPermissions(ctx, req.CustomerID, req.AccountID)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, fmt.Errorf("could not update aws permissions. error %s", err))
		return
	}

	err = p.SendWelcomeEmail(ctx, &req)
	if err != nil {
		if err != aws.ErrEmailAlreadySent {
			AbortWithError(ctx, http.StatusInternalServerError, fmt.Errorf("error sending welcome email: %s", err))
			return
		}
	}

	ctx.Status(http.StatusOK)
}

func AWSDeleteRoleHandler(ctx *gin.Context) {
	p, err := aws.NewPermissions(ctx)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer p.Close()

	var req aws.RoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		AbortWithError(ctx, http.StatusBadRequest, err)
		return
	}

	if len(req.AccountID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing account id"))
		return
	}

	if len(req.CustomerID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing external id"))
		return
	}

	if len(req.Arn) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing arn"))
		return
	}

	if len(req.StackID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing stack id"))
		return
	}

	err = p.DeleteRole(ctx, &req)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, fmt.Errorf("could not delete aws role. error %s", err))
		return
	}

	err = p.UpdateAWSPermissions(ctx, req.CustomerID, req.AccountID)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, errors.New("could not update aws permissions"))
		return
	}

	ctx.Status(http.StatusOK)
}

// AWSUpdateFeature sends a notification into channels collection,
// in order to update client when aws-lambda function was finished.
// it also executes update aws permissions to check the health of the account.
func AWSUpdateFeature(ctx *gin.Context) {
	p, err := aws.NewPermissions(ctx)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer p.Close()

	var req aws.RoleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		AbortWithError(ctx, http.StatusBadRequest, err)
		return
	}

	if len(req.AccountID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing account id"))
		return
	}

	if len(req.Arn) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing arn"))
		return
	}

	if len(req.StackID) == 0 {
		AbortWithError(ctx, http.StatusBadRequest, errors.New("missing stack id"))
		return
	}

	err = p.UpdateChannelNotification(ctx, &req, "")
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, errors.New("could not update channel notification"))
		return
	}

	err = p.UpdateAWSPermissions(ctx, req.CustomerID, req.AccountID)
	if err != nil {
		AbortWithError(ctx, http.StatusInternalServerError, errors.New("could not update aws permissions"))
		return
	}

	ctx.Status(http.StatusOK)
}

func AbortWithError(ctx *gin.Context, statusCode int, err error) {
	ctx.JSON(statusCode, gin.H{
		"error": err.Error(),
	})
}

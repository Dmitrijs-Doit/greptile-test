package handlers

import (
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type AWS struct {
	loggerProvider   logger.Provider
	awsService       amazonwebservices.IAWSService
	awsAssetsService assets.IAWSAssetsService
	awsServiceLimits *amazonwebservices.ServiceLimits
}

func NewAWS(log logger.Provider, conn *connection.Connection) *AWS {
	awsService, err := amazonwebservices.NewAWSService(log, conn)
	if err != nil {
		panic(err)
	}

	awsAssetsService, err := assets.NewAWSAssetsService(log, conn, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	awsServiceLimits := amazonwebservices.NewServiceLimits(log, conn)

	return &AWS{
		log,
		awsService,
		awsAssetsService,
		awsServiceLimits,
	}
}

func (h *AWS) CreatePricebook(ctx *gin.Context) error {
	amazonwebservices.CreatePricebook(ctx)

	return nil
}

func (h *AWS) UpdatePricebook(ctx *gin.Context) error {
	amazonwebservices.UpdatePricebook(ctx)

	return nil
}

func (h *AWS) AssignPricebook(ctx *gin.Context) error {
	amazonwebservices.AssignPricebook(ctx)

	return nil
}

func (h *AWS) GetCustomerServicesLimitsAWS(ctx *gin.Context) error {
	if err := h.awsServiceLimits.GetCustomerServicesLimits(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWS) UpdateCustomerLimitAWS(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return web.NewRequestError(errors.New("invalid empty customer id"), http.StatusBadRequest)
	}

	if err := h.awsServiceLimits.UpdateCustomerLimit(ctx, customerID); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWS) GetSupportRole(ctx *gin.Context) error {
	amazonwebservices.GetSupportRole(ctx)

	return nil
}

func (h *AWS) UpdateAccounts(ctx *gin.Context) error {
	if err := h.awsService.UpdateAccounts(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *AWS) UpdateHandshakes(ctx *gin.Context) error {
	if err := h.awsService.UpdateHandshakes(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func (h *AWS) InviteAccount(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")
	email := ctx.GetString("email")

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEntityID:   entityID,
		logger.LabelEmail:      email,
	})

	var body amazonwebservices.InviteAccountBody

	if err := ctx.ShouldBindJSON(&body); err != nil {
		return err
	}

	l.Infof("%+v", body)

	statusCode, err := h.awsService.InviteAccount(ctx, customerID, entityID, email, &body)
	if err != nil {
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			var message string

			statusCode = http.StatusBadRequest

			switch aerr.Code() {
			case organizations.ErrCodeConstraintViolationException,
				organizations.ErrCodeHandshakeConstraintViolationException:
				message = aerr.Message()
			case organizations.ErrCodeDuplicateHandshakeException:
				message = "There is already an active invite for the requested account"
			default:
				message = "Invite account operation failed, try again later"
				statusCode = http.StatusInternalServerError
			}

			// set the error on the context onlt if we have internal server error
			if statusCode >= http.StatusInternalServerError {
				ctx.Error(aerr) // nolint: errcheck
			}

			data := map[string]interface{}{
				"code":    aerr.Code(),
				"message": message,
			}

			return web.Respond(ctx, data, statusCode)
		}

		// Other kind of error
		return web.NewRequestError(err, statusCode)
	}

	return web.Respond(ctx, nil, statusCode)
}

func (h *AWS) CreateAccount(ctx *gin.Context) error {
	customerID := ctx.Param("customerID")
	entityID := ctx.Param("entityID")
	email := ctx.GetString("email")

	l := h.loggerProvider(ctx)
	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		logger.LabelEntityID:   entityID,
		logger.LabelEmail:      email,
	})

	var body amazonwebservices.CreateAccountBody

	if err := ctx.ShouldBindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	l.Infof("%+v", body)

	if _, err := h.awsService.CreateAccount(ctx, customerID, entityID, email, &body); err != nil {
		switch err {
		case amazonwebservices.ErrAccountAlreadyExist:
			return web.NewRequestError(err, http.StatusConflict)
		case amazonwebservices.ErrEmailIsNotValid:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWS) UpdateAWSAssetsDedicated(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	l.Info("Asset Discovery UpdateAWSAssetsDedicated - started")

	err := h.awsAssetsService.UpdateAssetsAllMPA(ctx)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWS) UpdateAWSAssetDedicated(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	l.Info("Asset Discovery UpdateAWSAssetDedicated - started")

	mpaID := ctx.Param("accountID")
	err := h.awsAssetsService.UpdateAssetsMPA(ctx, mpaID)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWS) UpdateManualAsset(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	l.Infof("Asset Discovery UpdateManualAsset - started")

	mpaID := ctx.Param("accountID")
	err := h.awsAssetsService.UpdateManualAssetsMPA(ctx, mpaID)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AWS) GetAssetFromAccountNumber(ctx *gin.Context) error {
	accountNumber := ctx.Param("accountNumber")
	asset, err := h.awsAssetsService.GetAssetFromAccountNumber(ctx, accountNumber)

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "AccessDeniedException" {
				return web.NewRequestError(err, http.StatusForbidden)
			}
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, asset, http.StatusOK)
}

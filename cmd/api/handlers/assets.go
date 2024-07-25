package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	awsassets "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets"
	"github.com/doitintl/hello/scheduled-tasks/assets"
	assetsdal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	assetHandlers "github.com/doitintl/hello/scheduled-tasks/assets/handlers"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/googleclouddirect"
	"github.com/doitintl/hello/scheduled-tasks/gsuite"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

type AssetHandler struct {
	loggerProvider   logger.Provider
	conn             *connection.Connection
	service          *assets.AssetService
	gcpDirectService *googleclouddirect.AssetService
	awsService       *amazonwebservices.AWSService
	awsAssetService  *awsassets.AWSAssetsService
}

func NewAssetHandler(loggerProvider logger.Provider, conn *connection.Connection) *AssetHandler {
	assetService := assets.NewAssetService(loggerProvider, conn)
	gcpDirectService := googleclouddirect.NewAssetService(loggerProvider, conn)

	awsService, err := amazonwebservices.NewAWSService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	awsAssetsService, err := awsassets.NewAWSAssetsService(loggerProvider, conn, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	return &AssetHandler{
		loggerProvider,
		conn,
		assetService,
		gcpDirectService,
		awsService,
		awsAssetsService,
	}
}

func (h *AssetHandler) AssetsDigestHandler(ctx *gin.Context) error {
	if err := assetHandlers.DailyDigestHandler(ctx, h.conn); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return nil
}

func StandaloneBillingAccountsListHandler(ctx *gin.Context) error {
	googlecloud.StandaloneBillingAccountsListHandler(ctx)

	return nil
}

func BillingAccountsPageHandler(ctx *gin.Context) error {
	googlecloud.BillingAccountsPageHandler(ctx)

	return nil
}

func SubscriptionsListHandlerGSuite(ctx *gin.Context) error {
	gsuite.SubscriptionsListHandler(ctx)

	return nil
}

func SubscriptionsListHandlerMicrosoft(ctx *gin.Context) error {
	microsoft.SubscriptionsListHandler(ctx)

	return nil
}

func (h *AssetHandler) UpdateAWSAssetsShared(ctx *gin.Context) error {
	l := h.loggerProvider(ctx)
	l.Info("Asset Discovery UpdateAWSAssetsShared - started")

	if err := h.awsService.UpdateAssetsSharedPayers(ctx); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func TagCustomers(ctx *gin.Context) error {
	amazonwebservices.TagCustomers(ctx)

	return nil
}

func DeleteServiceAccount(ctx *gin.Context) error {
	googlecloud.DeleteServiceAccount(ctx)

	return nil
}

func ImportCloudServicesToFS(ctx *gin.Context) error {
	googlecloud.ImportCloudServicesToFS(ctx)

	return nil
}

func ImportAWSCloudServicesToFS(ctx *gin.Context) error {
	amazonwebservices.ImportAWSCloudServicesToFS(ctx)

	return nil
}

type CreateGoogleCloudDirectAssetHandlerBody struct {
	BillingAccountId string `json:"billingAccountId" binding:"required"`
	Dataset          string `json:"dataset" binding:"required"`
	Project          string `json:"project" binding:"required"`
}

func (h *AssetHandler) CreateGoogleCloudDirectAssetHandler(ctx *gin.Context) error {
	var body CreateGoogleCloudDirectAssetHandlerBody
	if err := ctx.BindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := h.gcpDirectService.CreateGoogleCloudDirectAssetService(ctx, &googleclouddirect.CreateAssetParams{
		CustomerID:       ctx.Param("customerID"),
		Project:          body.Project,
		Dataset:          body.Dataset,
		BillingAccountID: body.BillingAccountId,
	})
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AssetHandler) UpdateGoogleCloudDirectAssetHandler(ctx *gin.Context) error {
	id := ctx.Param("id")

	findParams := googleclouddirect.ModifyAssetParams{
		CustomerId: ctx.Param("customerID"),
		AssetId:    id,
	}

	if err := h.gcpDirectService.CanModifyAsset(ctx, &findParams); err != nil {
		switch err {
		case googleclouddirect.ErrorAssetsNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case googleclouddirect.ErrorAssetStateIsOtherThanError:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	var body googleclouddirect.AssetDataToUpdate
	if err := ctx.BindJSON(&body); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if err := h.gcpDirectService.Update(ctx, id, &body); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AssetHandler) DeleteGoogleCloudDirectAssetHandler(ctx *gin.Context) error {
	params := googleclouddirect.ModifyAssetParams{
		CustomerId: ctx.Param("customerID"),
		AssetId:    ctx.Param("id"),
	}

	if err := h.gcpDirectService.CanModifyAsset(ctx, &params); err != nil {
		switch err {
		case googleclouddirect.ErrorAssetsNotFound:
			return web.NewRequestError(err, http.StatusNotFound)
		case googleclouddirect.ErrorAssetStateIsOtherThanError:
			return web.NewRequestError(err, http.StatusBadRequest)
		default:
			return web.NewRequestError(err, http.StatusInternalServerError)
		}
	}

	err := h.gcpDirectService.Delete(ctx, params.AssetId)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AssetHandler) UpdateAssetSettingsHandler(ctx *gin.Context) error {
	assetID := ctx.Param("assetID")
	if assetID == "" {
		return web.NewRequestError(assets.ErrInvalidAssetID, http.StatusBadRequest)
	}

	email := ctx.GetString("email")

	var settings assets.Settings
	if err := ctx.ShouldBindJSON(&settings); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	if settings.Payment == "" && settings.Plan == nil && settings.Currency == "" {
		return web.NewRequestError(assets.ErrInvalidRequestBody, http.StatusBadRequest)
	}

	if err := h.service.UpdateAssetSettings(ctx, assetID, email, &settings); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AssetHandler) DeleteAssetSettingsHandler(ctx *gin.Context) error {
	assetID := ctx.Param("assetID")
	if assetID == "" {
		return web.NewRequestError(assets.ErrInvalidAssetID, http.StatusBadRequest)
	}

	email := ctx.GetString("email")

	if err := h.service.UpdateAssetSettings(ctx, assetID, email, nil); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *AssetHandler) GetAWSAsset(ctx *gin.Context) error {
	accountNumber := ctx.Query("accountNumber")
	if accountNumber == "" {
		return web.NewRequestError(fmt.Errorf("accountNumber is required"), http.StatusBadRequest)
	}

	// TODO: support WhereQuery?
	if len(ctx.Request.URL.Query()) > 1 {
		return web.NewRequestError(fmt.Errorf("only accountNumber is supported"), http.StatusBadRequest)
	}

	asset, err := h.awsAssetService.GetAssetFromAccountNumber(ctx, accountNumber)

	switch {
	case errors.Is(err, assetsdal.ErrFoundNoAssets):
		return web.NewRequestError(err, http.StatusNotFound)
	case errors.Is(err, assetsdal.ErrFoundMoreThanOneAsset):
		return web.NewRequestError(err, http.StatusInternalServerError)
	case err == nil:
		return web.Respond(ctx, asset, http.StatusOK)
	default:
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

}

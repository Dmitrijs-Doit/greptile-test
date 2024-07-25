package handlers

import (
	"context"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/firestore"
	assetDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	marketplaceDal "github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	httpPkg "github.com/doitintl/hello/scheduled-tasks/marketplace/http"
	marketplaceService "github.com/doitintl/hello/scheduled-tasks/marketplace/service"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service/slack"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	doitPubsub "github.com/doitintl/pubsub"
	doitPubsubIface "github.com/doitintl/pubsub/iface"
)

type MarketplaceGCP struct {
	loggerProvider logger.Provider
	service        iface.MarketplaceIface
}

type TopicHandlerProviderIface func(isProduction bool) (doitPubsubIface.TopicHandler, error)

func TopicHandlerProvider(isProduction bool) (doitPubsubIface.TopicHandler, error) {
	ctx := context.Background()
	procurementProjectID := marketplaceDal.GetProcurementProjectID(isProduction)

	client, err := pubsub.NewClient(ctx, procurementProjectID)
	if err != nil {
		return nil, err
	}

	topicHandler, err := doitPubsub.NewTopicHandler(
		ctx,
		client,
		marketplaceDal.ProcurementEventsTopic,
	)
	if err != nil {
		return nil, err
	}

	return topicHandler, nil
}

func NewMarketplace(log logger.Provider, conn *connection.Connection, topicHandlerProvider TopicHandlerProviderIface) *MarketplaceGCP {
	assetDAL := assetDal.NewAssetsFirestoreWithClient(conn.Firestore)
	customerDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	entitlementDAL := marketplaceDal.NewEntitlementFirestoreDALWithClient(conn.Firestore)
	accountDAL := marketplaceDal.NewAccountFirestoreDALWithClient(conn.Firestore)
	userFirestoreDAL := userDal.NewUserFirestoreDALWithClient(conn.Firestore)

	procurementClient, err := httpPkg.NewProcurementClient()
	if err != nil {
		panic(err)
	}

	topicHandler, err := topicHandlerProvider(common.Production)
	if err != nil {
		panic(err)
	}

	procurementDAL, err := marketplaceDal.NewProcurementDAL(
		procurementClient,
		topicHandler,
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	integrationDAL := firestore.NewIntegrationsDALWithClient(conn.Firestore(ctx))

	flexsaveResoldService := flexsaveresold.NewGCPService(log, conn)

	customerTypeDal := firestore.NewCustomerTypeDALWithClient(conn.Firestore(ctx))

	slackService := slack.NewSlackService(log)

	service, err := marketplaceService.NewMarketplaceService(
		log,
		accountDAL,
		assetDAL,
		customerDAL,
		entitlementDAL,
		integrationDAL,
		procurementDAL,
		userFirestoreDAL,
		flexsaveResoldService,
		customerTypeDal,
		slackService,
	)
	if err != nil {
		panic(err)
	}

	return &MarketplaceGCP{
		log,
		service,
	}
}

func (h *MarketplaceGCP) ApproveAccount(ctx *gin.Context) error {
	accountID := ctx.Param("accountID")
	email := ctx.GetString("email")

	err := h.service.ApproveAccount(ctx, accountID, email)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *MarketplaceGCP) RejectAccount(ctx *gin.Context) error {
	accountID := ctx.Param("accountID")
	email := ctx.GetString("email")

	if err := h.service.RejectAccount(ctx, accountID, email); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *MarketplaceGCP) ApproveEntitlement(ctx *gin.Context) error {
	entitlementID := ctx.Param("entitlementID")

	email := ctx.GetString("email")
	doitEmployee := ctx.GetBool("doitEmployee")

	if err := h.service.ApproveEntitlement(
		ctx,
		entitlementID,
		email,
		doitEmployee,
		true,
	); err != nil {
		if err == marketplaceService.ErrCustomerIsNotEligibleFlexsave {
			return web.Respond(
				ctx,
				HTTPError{
					Error:      err.Error(),
					StatusCode: StatusCodeCustomerIsNotEligibleFlexsave,
				}, http.StatusUnprocessableEntity)
		}

		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *MarketplaceGCP) RejectEntitlement(ctx *gin.Context) error {
	entitlementID := ctx.Param("entitlementID")
	email := ctx.GetString("email")

	if err := h.service.RejectEntitlement(ctx, entitlementID, email); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *MarketplaceGCP) PopulateBillingAccounts(ctx *gin.Context) error {
	var populateBillingAccounts domain.PopulateBillingAccounts

	if err := ctx.ShouldBindJSON(&populateBillingAccounts); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	populateBillingAccountsResults, err := h.service.PopulateBillingAccounts(ctx, populateBillingAccounts)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	for _, result := range populateBillingAccountsResults {
		if result.Error != "" {
			return web.Respond(ctx, populateBillingAccountsResults, http.StatusMultiStatus)
		}
	}

	return web.Respond(ctx, populateBillingAccountsResults, http.StatusOK)
}

func (h *MarketplaceGCP) PopulateSingleBillingAccount(ctx *gin.Context) error {
	accountID := ctx.Param("accountID")

	populateBillingAccounts := domain.PopulateBillingAccounts{
		{
			ProcurementAccountID: accountID,
		},
	}

	populateBillingAccountsResults, err := h.service.PopulateBillingAccounts(ctx, populateBillingAccounts)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	for _, result := range populateBillingAccountsResults {
		if result.ProcurementAccountID == accountID && result.Error != "" {
			return web.Respond(ctx, populateBillingAccountsResults, http.StatusMultiStatus)
		}
	}

	return web.Respond(ctx, populateBillingAccountsResults, http.StatusOK)
}

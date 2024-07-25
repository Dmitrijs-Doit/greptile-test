package handlers

import (
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/aws-marketplace/service"
	"github.com/doitintl/hello/scheduled-tasks/aws-marketplace/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type MarketplaceAWS struct {
	loggerProvider logger.Provider
	service        iface.MarketplaceServiceIface
}

type IntegrationSvcPayload struct {
	AwsMpSubscriptionDocID string `json:"awsMpSubscriptionDocId,omitempty"`
	CustomerID             string `json:"customerId,omitempty"`
}

func NewMarketplaceAWS(log logger.Provider, conn *connection.Connection) *MarketplaceAWS {
	s, err := service.NewAWSMarketplaceService(log, conn)
	if err != nil {
		panic(err)
	}

	return &MarketplaceAWS{
		log,
		s,
	}
}

func (a MarketplaceAWS) ResolveCustomer(ctx *gin.Context) error {
	var resolveCustomerPayload struct {
		AwsMpSubscriptionDocID string `json:"awsMpSubscriptionDocId"`
	}

	if err := ctx.ShouldBindJSON(&resolveCustomerPayload); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err := a.service.ResolveCustomer(ctx, resolveCustomerPayload.AwsMpSubscriptionDocID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (a MarketplaceAWS) ValidateEntitlement(ctx *gin.Context) error {
	var entitlementValidationPayload struct {
		AwsMpSubscriptionDocID string `json:"awsMpSubscriptionDocId"`
	}

	if err := ctx.ShouldBindJSON(&entitlementValidationPayload); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err := a.service.ValidateEntitlement(ctx, entitlementValidationPayload.AwsMpSubscriptionDocID)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

package service

import (
	"context"
	"github.com/doitintl/hello/scheduled-tasks/aws-marketplace/http"
	"github.com/doitintl/hello/scheduled-tasks/aws-marketplace/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	httpClient "github.com/doitintl/http"
)

type AWSMarketplaceService struct {
	loggerProvider logger.Provider
	client         httpClient.IClient
	conn           *connection.Connection
}

type IntegrationSvcPayload struct {
	AwsMpSubscriptionDocID string `json:"awsMpSubscriptionDocId,omitempty"`
}

type AWSMarketplaceMeteringMessageData struct {
	PayerID   string  `json:"PayerID" validate:"required"`
	Charge    float32 `json:"Charge" validate:"required"`
	Timestamp string  `json:"Timestamp" validate:"required"`
}

func NewAWSMarketplaceService(log logger.Provider, conn *connection.Connection) (iface.MarketplaceServiceIface, error) {
	client, err := http.NewIntegrationServiceClient()
	if err != nil {
		return nil, err
	}

	return AWSMarketplaceService{
		log,
		client,
		conn,
	}, nil
}

func (a AWSMarketplaceService) ResolveCustomer(ctx context.Context, awsSubscriptionID string) error {
	res := struct {
		Healthy bool   `json:"healthy,omitempty"`
		Error   string `json:"error_message,omitempty"`
	}{}

	_, err := a.client.Post(ctx, &httpClient.Request{
		URL: "/api/v1/internal/resolve-customer",
		Payload: IntegrationSvcPayload{
			AwsMpSubscriptionDocID: awsSubscriptionID,
		},
		ResponseType: &res,
	})

	if err != nil {
		return err
	}

	return nil
}

func (a AWSMarketplaceService) ValidateEntitlement(ctx context.Context, awsSubscriptionID string) error {
	res := struct {
		Healthy bool   `json:"healthy,omitempty"`
		Error   string `json:"error_message,omitempty"`
	}{}

	_, err := a.client.Post(ctx, &httpClient.Request{
		URL: "/api/v1/internal/entitlement-validation",
		Payload: IntegrationSvcPayload{
			AwsMpSubscriptionDocID: awsSubscriptionID,
		},
		ResponseType: &res,
	})

	if err != nil {
		return err
	}

	return nil
}

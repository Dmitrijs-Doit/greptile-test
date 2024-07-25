package service

import (
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type StripeWebhookService struct {
	loggerProvider logger.Provider
	*connection.Connection
	stripeClient     *Client
	integrationDocID string
}

func NewStripeWebhookService(loggerProvider logger.Provider, conn *connection.Connection, stripeClient *Client) *StripeWebhookService {
	integrationDocID := stripeClient.integrationDocID

	return &StripeWebhookService{
		loggerProvider,
		conn,
		stripeClient,
		integrationDocID,
	}
}

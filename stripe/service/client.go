package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v74/client"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

type Client struct {
	*client.API
	accountID        domain.StripeAccountID
	webhookSignKey   string
	integrationDocID string
}

type stripeSecret struct {
	APIKey         string `json:"api_key"`
	WebhookSignKey string `json:"webhook_sign_key"`
}

func NewStripeClient(account domain.StripeAccountID) (*Client, error) {
	ctx := context.Background()

	stripeSecret, err := getStripeSecret(ctx, account)
	if err != nil {
		return nil, err
	}

	// Init stripe client
	var stripeClient client.API

	stripeClient.Init(stripeSecret.APIKey, nil)

	return &Client{
		&stripeClient,
		account,
		stripeSecret.WebhookSignKey,
		getIntegrationDocID(),
	}, nil
}

func getStripeSecret(ctx context.Context, account domain.StripeAccountID) (stripeSecret, error) {
	// Stripe have a signing key for each webhook; CMP currently uses only one webhook.
	// If creating new a new webhook, this secret and code should probably be refactored.
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretStripeAccounts)
	if err != nil {
		return stripeSecret{}, err
	}

	stripeAccountSecrets := make(map[domain.StripeAccountID]stripeSecret)

	if err := json.Unmarshal(data, &stripeAccountSecrets); err != nil {
		return stripeSecret{}, err
	}

	if secret, ok := stripeAccountSecrets[account]; ok {
		return secret, nil
	}

	return stripeSecret{}, fmt.Errorf("invalid stripe account: %s", account)
}

func getIntegrationDocID() string {
	if !common.Production {
		return "stripe-test"
	}

	return "stripe"
}

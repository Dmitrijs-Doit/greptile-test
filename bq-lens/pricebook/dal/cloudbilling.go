package dal

import (
	"context"
	"fmt"

	cb "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

//go:generate mockery --name CloudBilling --output ./mocks --case=underscore
type CloudBilling interface {
	NewService(ctx context.Context) (*cb.APIService, error)
	GetServiceSKUs(ctx context.Context, serviceName string) (*cb.ListSkusResponse, error)
}

type CloudBillingAPI struct{}

func NewCloudBillingAPI() *CloudBillingAPI {
	return &CloudBillingAPI{}
}

func (s *CloudBillingAPI) NewService(ctx context.Context) (*cb.APIService, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
	if err != nil {
		return nil, err
	}

	creds := option.WithCredentialsJSON(secret)

	return cb.NewService(ctx, creds)
}

func (s *CloudBillingAPI) GetServiceSKUs(ctx context.Context, serviceName string) (*cb.ListSkusResponse, error) {
	service, err := s.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudbilling service: %w", err)
	}

	return service.Services.Skus.List(serviceName).Do()
}

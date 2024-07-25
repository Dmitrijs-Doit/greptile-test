package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/azure"
	"github.com/doitintl/hello/scheduled-tasks/azure/dal"
	azureErrors "github.com/doitintl/hello/scheduled-tasks/azure/errors"
	azurePkg "github.com/doitintl/hello/scheduled-tasks/azure/iface"
	customer "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/go-playground/validator/v10"
)

type Service interface {
	StoreBillingDataConnection(ctx context.Context, customerID string, data azurePkg.Payload) error
	GetStorageAccountNameForOnboarding(ctx context.Context, customerID string) (string, error)
}

func NewService(ctx context.Context, fs *firestore.Client) (Service, error) {
	appConfig, err := getAppConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &service{
		azure.NewService(appConfig),
		dal.NewFirestoreDAL(func(ctx context.Context) *firestore.Client { return fs }),
		customer.NewCustomersFirestoreWithClient(func(ctx context.Context) *firestore.Client { return fs }),
		&saasconsole.SlackUtilsFunctions{},
	}, nil
}

type service struct {
	azureDAL     azure.Service
	firestoreDAL dal.FirestoreDAL
	customerDAL  customer.Customers
	slackUtils   saasconsole.SlackUtils
}

func getAppConfig(ctx context.Context) (azure.ClientConfig, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.AzureLighthouseAccess)
	if err != nil {
		return azure.ClientConfig{}, err
	}

	var config azure.ClientConfig
	if err := json.Unmarshal(secret, &config); err != nil {
		return azure.ClientConfig{}, err
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return azure.ClientConfig{}, fmt.Errorf("invalid app config: %w", err)
	}

	return config, nil
}

func (s *service) StoreBillingDataConnection(ctx context.Context, customerID string, data azurePkg.Payload) error {
	configs, err := s.firestoreDAL.GetCustomerBillingDataConfigs(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to get customer billing data configs: %w", err)
	}

	for _, config := range configs {
		if config.Container == data.Container &&
			config.ResourceGroup == data.ResourceGroup &&
			config.SubscriptionID == data.SubscriptionID &&
			config.CustomerID == customerID &&
			config.Directory == data.Directory {
			return &azureErrors.InvalidRequestError{Message: "Connection with these details already exists"}
		}
	}

	detectedDirectory, err := s.azureDAL.VerifyBillingDataConnection(ctx, data.SubscriptionID, data.ResourceGroup, data.Account, data.Container, data.Directory)
	if err != nil {
		var reqErr *azure.InvalidRequestError

		if errors.As(err, &reqErr) {
			return &azureErrors.InvalidRequestError{Message: reqErr.Message}
		} else {
			return fmt.Errorf("failed to verify billing data connection: %w", err)
		}
	}

	if err := s.firestoreDAL.CreateCustomerBillingDataConfig(ctx, dal.BillingDataConfig{
		CustomerID:     customerID,
		Container:      data.Container,
		Account:        data.Account,
		ResourceGroup:  data.ResourceGroup,
		SubscriptionID: data.SubscriptionID,
		Directory:      detectedDirectory,
		CreatedAt:      time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to create customer billing data config: %w", err)
	}

	err = s.customerDAL.UpdateCustomerFieldValueDeep(ctx,
		customerID, []string{"enabledSaaSConsole", "AZURE"}, true)
	if err != nil {
		return fmt.Errorf("failed to update customer field value: %w", err)
	}

	slackErr := s.slackUtils.PublishOnboardSuccessSlackNotification(ctx, pkg.AZURE, s.customerDAL, customerID, data.Account)
	if slackErr != nil {
		loggerProvider := logger.FromContext(ctx)
		loggerProvider.Error(slackErr)
	}

	return nil
}

func (s *service) GetStorageAccountNameForOnboarding(ctx context.Context, customerID string) (string, error) {
	configs, err := s.firestoreDAL.GetCustomerBillingDataConfigs(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("failed to get customer billing data configs: %w", err)
	}

	var lowerCaseCustomerID = strings.ToLower(customerID)

	if len(configs) == 0 {
		return lowerCaseCustomerID, nil
	}

	const maxConnections = 99

	if (len(configs)) >= maxConnections {
		return "", errors.New("customer has reached the maximum number of connections")
	}

	return fmt.Sprintf("%s%d", lowerCaseCustomerID, len(configs)+1), nil
}

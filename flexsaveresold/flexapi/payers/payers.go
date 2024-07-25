package payers

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/shared"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/http"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	UnsubscribeCustomerPayerAccount(ctx context.Context, payerAccountID string) error
	UpdatePayerConfigsForCustomer(ctx context.Context, configs []types.PayerConfig) ([]types.PayerConfig, error)
	UpdateStatusWithRequired(ctx context.Context, accountID string, serviceType utils.FlexsaveType, serviceStatus string) error
	CreatePayerConfigForCustomer(ctx context.Context, payload types.PayerConfigCreatePayload) error
	GetPayerConfigsForCustomer(ctx context.Context, customerID string) ([]*types.PayerConfig, error)
	GetAWSStandaloneCustomerIDs(ctx context.Context) ([]string, error)
	GetPayerConfig(ctx context.Context, accountID string) (*types.PayerConfig, error)
}

type service struct {
	flexAPIClient http.IClient
}

func NewService() (Service, error) {
	ctx := context.Background()
	baseURL := shared.GetFlexAPIURL()

	tokenSource, err := shared.GetTokenSource(ctx)
	if err != nil {
		return nil, err
	}

	client, err := http.NewClient(ctx, &http.Config{
		BaseURL:     baseURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		return nil, err
	}

	return &service{
		client,
	}, nil
}

func (s *service) GetPayerConfigsForCustomer(ctx context.Context, customerID string) ([]*types.PayerConfig, error) {
	var ids []*types.PayerConfig

	if _, err := s.flexAPIClient.Get(ctx, &http.Request{
		URL:          fmt.Sprintf("/customers/%s/payers", customerID),
		ResponseType: &ids,
	}); err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *service) CreatePayerConfigForCustomer(ctx context.Context, payload types.PayerConfigCreatePayload) error {
	if _, err := s.flexAPIClient.Post(ctx, &http.Request{
		URL:     "/payers",
		Payload: &payload,
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) UpdatePayerConfigsForCustomer(ctx context.Context, configs []types.PayerConfig) ([]types.PayerConfig, error) {
	var updatedConfigs []types.PayerConfig

	changedBy, ok := ctx.Value(common.CtxKeys.Email).(string)
	if !ok {
		changedBy = ""
	}

	reason, ok := ctx.Value(utils.StatusChangeReasonContextKey).(string)
	if !ok {
		reason = ""
	}

	var payload types.PayerConfigUpdatePayload
	payload.PayerConfigs = configs
	payload.ChangedBy = changedBy
	payload.Reason = reason

	if _, err := s.flexAPIClient.Put(ctx, &http.Request{
		URL:          "/payers",
		Payload:      &payload,
		ResponseType: &updatedConfigs,
	}); err != nil {
		return updatedConfigs, err
	}

	return updatedConfigs, nil
}

func (s *service) UpdateStatusWithRequired(ctx context.Context, accountID string, serviceType utils.FlexsaveType, serviceStatus string) error {
	data, err := s.GetPayerConfig(ctx, accountID)
	if err != nil {
		return err
	}

	withDefaults := types.PayerConfig{
		CustomerID:    data.CustomerID,
		AccountID:     data.AccountID,
		PrimaryDomain: data.PrimaryDomain,
		Name:          data.Name,
		Type:          data.Type,
		FriendlyName:  data.FriendlyName,
	}

	// update relevant fields for each service
	// compute status field is required for endpoint validation
	switch serviceType {
	case utils.ComputeFlexsaveType:
		withDefaults.Status = serviceStatus

	case utils.RDSFlexsaveType:
		withDefaults.Status = data.Status
		withDefaults.RDSStatus = serviceStatus

	case utils.SageMakerFlexsaveType:
		withDefaults.Status = data.Status
		withDefaults.SageMakerStatus = serviceStatus
	}

	_, err = s.UpdatePayerConfigsForCustomer(ctx, []types.PayerConfig{withDefaults})

	return err
}

func (s *service) UnsubscribeCustomerPayerAccount(ctx context.Context, payerAccountID string) error {
	if _, err := s.flexAPIClient.Put(ctx, &http.Request{
		URL: fmt.Sprintf("/payers/%s/unsubscribe", payerAccountID),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) GetAWSStandaloneCustomerIDs(ctx context.Context) ([]string, error) {
	var ids []string

	if _, err := s.flexAPIClient.Get(ctx, &http.Request{
		URL:          "/payers/customer/standalone/ids",
		ResponseType: &ids,
	}); err != nil {
		return nil, err
	}

	return ids, nil
}

func (s *service) GetPayerConfig(ctx context.Context, accountID string) (*types.PayerConfig, error) {
	var payer types.PayerConfig

	if _, err := s.flexAPIClient.Get(ctx, &http.Request{
		URL:          fmt.Sprintf("/payers/%s", accountID),
		ResponseType: &payer,
	}); err != nil {
		return nil, err
	}

	return &payer, nil
}

package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

const (
	bigqueryScope = "https://www.googleapis.com/auth/bigquery"
)

type ExternalBigQueryClient interface {
	GetCustomerBQClient(ctx context.Context, billingAccountID string) (*bigquery.Client, error)
	GetCustomerBQClientWithParams(ctx context.Context, serviceAccountEmail, projectID string) (*bigquery.Client, error)
}
type ExternalBigQueryClientImpl struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata *dal.Metadata
}

func NewExternalBigQueryClient(log logger.Provider, conn *connection.Connection) *ExternalBigQueryClientImpl {
	return &ExternalBigQueryClientImpl{
		log,
		conn,
		dal.NewMetadata(log, conn),
	}
}

func (s *ExternalBigQueryClientImpl) GetCustomerBQClient(ctx context.Context, billingAccountID string) (*bigquery.Client, error) {
	etm, err := s.metadata.GetExternalTaskMetadata(ctx, billingAccountID)
	if err != nil {
		return nil, err
	}

	return s.GetCustomerBQClientWithParams(ctx, etm.ServiceAccountEmail, etm.BQTable.ProjectID)
}

func (s *ExternalBigQueryClientImpl) GetCustomerBQClientWithParams(ctx context.Context, serviceAccountEmail, projectID string) (*bigquery.Client, error) {
	logger := s.loggerProvider(ctx)

	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: serviceAccountEmail,
		Scopes:          []string{"https://www.googleapis.com/auth/bigquery", "https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		return nil, err
	}

	customerBQ, err := bigquery.NewClient(ctx, projectID, option.WithTokenSource(ts))
	if err != nil {
		err = fmt.Errorf("unable to create bq NewClient for SA %s in project %s. Caused by %s", serviceAccountEmail, projectID, err)
		logger.Error(err)

		return nil, err
	}

	return customerBQ, nil
}

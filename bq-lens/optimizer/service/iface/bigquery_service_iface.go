package iface

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

//go:generate mockery --name Bigquery --output ../mocks --case=underscore
type Bigquery interface {
	GetCustomerDiscounts(ctx context.Context, bq *bigquery.Client) (map[string]float64, error)
	GetDatasetLocationAndProjectID(ctx context.Context, bq *bigquery.Client, datasetID string) (string, string, error)
	GetTableDiscoveryMetadata(ctx context.Context, bq *bigquery.Client) (*bigquery.TableMetadata, error)
	GetAggregatedJobStatistics(ctx context.Context, bq *bigquery.Client, projectID, location string) ([]bqmodels.AggregatedJobStatistic, error)
	GetMinAndMaxDates(ctx context.Context, bq *bigquery.Client, projectID string, location string) (*bqmodels.CheckCompleteDaysResult, error)
	GenerateStorageRecommendation(ctx context.Context, customerID string, bq *bigquery.Client, discount float64, replacements domain.Replacements, now time.Time, hasTableDiscovery bool) (domain.PeriodTotalPrice, dal.RecommendationSummary, error)
	GetBillingProjectsWithEditions(ctx context.Context, bq *bigquery.Client) (map[string][]domain.BillingProjectWithReservation, error)
	GetBillingProjectsWithEditionsSingleCustomer(ctx context.Context, bq *bigquery.Client, customerID string) ([]domain.BillingProjectWithReservation, error)
}

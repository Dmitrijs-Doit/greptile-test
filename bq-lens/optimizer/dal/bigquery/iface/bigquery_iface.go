//go:generate mockery --name Bigquery --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

type Bigquery interface {
	RunQuery(
		ctx context.Context,
		bq *bigquery.Client,
		query string,
	) (iface.RowIterator, error)

	GetDatasetLocationAndProjectID(
		ctx context.Context,
		bq *bigquery.Client,
		datasetID string,
	) (string, string, error)

	GetTableDiscoveryMetadata(
		ctx context.Context,
		bq *bigquery.Client,
	) (*bigquery.TableMetadata, error)

	RunCheckCompleteDaysQuery(
		ctx context.Context,
		query string,
		bq *bigquery.Client,
	) ([]bqmodels.CheckCompleteDaysResult, error)

	RunAggregatedJobStatisticsQuery(
		ctx context.Context,
		bq *bigquery.Client,
		projectID string,
		location string,
	) ([]bqmodels.AggregatedJobStatistic, error)

	RunTableStorageTBQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.TableStorageTBResult, error)

	RunTableStoragePriceQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.TableStoragePriceResult, error)

	RunDatasetStorageTBQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.DatasetStorageTBResult, error)

	RunDatasetStoragePriceQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.DatasetStoragePriceResult, error)

	RunProjectStorageTBQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.ProjectStorageTBResult, error)

	RunProjectStoragePriceQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.ProjectStoragePriceResult, error)

	RunCostFromTableTypesQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.CostFromTableTypesResult, error)

	RunStorageRecommendationsQuery(
		ctx context.Context,
		bq *bigquery.Client,
		replacements domain.Replacements,
		now time.Time,
	) ([]bqmodels.StorageRecommendationsResult, error)

	RunDiscountsAllCustomersQuery(
		ctx context.Context,
		query string,
		bq *bigquery.Client,
	) ([]bqmodels.DiscountsAllCustomersResult, error)

	RunScheduledQueriesMovementQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) ([]bqmodels.ScheduledQueriesMovementResult, error)

	RunTotalScanPricePerPeriod(
		ctx context.Context,
		bq *bigquery.Client,
		replacements domain.Replacements,
		now time.Time,
	) ([]bqmodels.ScanPricePerPeriod, error)

	RunBillingProjectSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunBillingProjectResult, error)

	RunFlatRateUserSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunUserSlotsResult, error)

	RunFlatRateSlotsExplorerQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.FlatRateSlotsExplorerResult, error)

	RunStandardUserSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunStandardUserSlotsResult, error)

	RunEnterpriseUserSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunEnterpriseUserSlotsResult, error)

	RunEnterprisePlusUserSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunEnterprisePlusUserSlotsResult, error)

	RunOnDemandSlotsExplorerQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.OnDemandSlotsExplorerResult, error)

	RunBillingProjectsWithEditionsQuery(
		ctx context.Context,
		query string,
		bq *bigquery.Client,
	) ([]bqmodels.BillingProjectsWithReservationsResult, error)

	RunStandardSlotsExplorerQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.StandardSlotsExplorerResult, error)

	RunEnterpriseSlotsExplorerQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.EnterpriseSlotsExplorerResult, error)

	RunEnterprisePlusSlotsExplorerQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.EnterprisePlusSlotsExplorerResult, error)

	RunStandardScheduledQueriesMovementQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.StandardScheduledQueriesMovementResult, error)

	RunEnterpriseScheduledQueriesMovementQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.EnterpriseScheduledQueriesMovementResult, error)

	RunEnterprisePlusScheduledQueriesMovementQuery(
		ctx context.Context,
		query string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		_ bqmodels.TimeRange,
	) ([]bqmodels.EnterprisePlusScheduledQueriesMovementResult, error)

	RunStandardBillingProjectSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunStandardBillingProjectResult, error)

	RunEnterpriseBillingProjectSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunEnterpriseBillingProjectResult, error)

	RunEnterprisePlusBillingProjectSlots(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunEnterprisePlusBillingProjectResult, error)

	RunOnDemandBillingProjectQuery(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunODBillingProjectResult, error)

	RunOnDemandUserQuery(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunODUserResult, error)

	RunOnDemandProjectQuery(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunODProjectResult, error)

	RunOnDemandDatasetQuery(
		ctx context.Context,
		_ string,
		replacements domain.Replacements,
		bq *bigquery.Client,
		timeRange bqmodels.TimeRange,
	) (*bqmodels.RunODDatasetResult, error)
}

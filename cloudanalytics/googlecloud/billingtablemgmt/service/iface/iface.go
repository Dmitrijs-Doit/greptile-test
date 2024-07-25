//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
)

type BillingTableManagementService interface {
	StandaloneBillingUpdateEvents(ctx context.Context) error
	InitRawTable(ctx context.Context) error
	InitRawResourceTable(ctx context.Context) error
	UpdateDiscounts(ctx context.Context) error
	ScheduledBillingAccountsTableUpdate(ctx context.Context) error
	UpdateRawTableLastPartition(ctx context.Context) error
	UpdateRawResourceTableLastPartition(ctx context.Context) error
	UpdateBillingAccountsTable(ctx context.Context, input domain.UpdateBillingAccountsTableInput) error
	UpdateBillingAccountTable(
		ctx context.Context,
		uri string,
		billingAccountID string,
		allPartitions bool,
		refreshMetadata bool,
		assetType string,
		fromDate string,
		numPartitions int,
	) error
	UpdateIAMResources(ctx context.Context) error
	UpdateCSPBillingAccounts(
		ctx context.Context,
		params domain.UpdateCspTaskParams,
		numPartitions int,
		allPartitions bool,
		fromDate string,
	) error
	AppendToTempCSPBillingAccountTable(
		ctx context.Context,
		billingAccountID string,
		updateAll bool,
		allPartitions bool,
		numPartitions int,
		fromDate string,
	) error
	UpdateCSPTableAndDeleteTemp(
		ctx context.Context,
		billingAccountID string,
		allPartitions bool,
		fromDate string,
		numPartitions int,
	) error
	JoinCSPTempTable(
		ctx context.Context,
		billingAccountID string,
		idx int,
		allPartitions bool,
		fromDate string,
		numPartitions int,
	) error
	UpdateCSPAggregatedTable(
		ctx context.Context,
		billingAccountID string,
		allPartitions bool,
	) error
	UpdateAggregatedTable(
		ctx context.Context,
		billingAccountID string,
		interval string,
		fromDate string,
		numPartitions int,
		allPartitions bool,
	) error
	UpdateAllAggregatedTables(
		ctx context.Context,
		billingAccountID string,
		fromDate string,
		numPartitions int,
		allPartitions bool,
	) []error
}

package tablemanagement

import (
	"context"
)

type IBillingTableManagementService interface {
	UpdateAggregatedTable(
		ctx context.Context,
		suffix,
		interval string,
		allPartitions bool,
	) error
	UpdateAllAggregatedTablesAllCustomers(
		ctx context.Context,
		allPartitions bool,
	) []error
	UpdateAllAggregatedTables(
		ctx context.Context,
		suffix string,
		allPartitions bool,
	) []error
	UpdateCSPAccounts(
		ctx context.Context,
		allPartitions bool,
		fromDate string,
		numPartitions int,
	) error
	UpdateCSPAggregatedTable(
		ctx context.Context,
		allPartitions bool,
		fromDate string,
		numPartitions int,
	) error
	UpdateDiscounts(
		ctx context.Context,
	) error
}

//go:generate mockery --output=../mocks --all
package tablemanagement

import (
	"context"
)

type BillingTableManagementService interface {
	UpdateAggregatedTable(ctx context.Context, suffix, interval string, allPartitions bool) error
	UpdateAllAggregatedTablesAllCustomers(ctx context.Context, allPartitions bool) []error
	UpdateAllAggregatedTables(ctx context.Context, suffix string, allPartitions bool) []error

	UpdateCSPTable(ctx context.Context, startDate, endDate string) error
	UpdateCSPAggregatedTable(ctx context.Context, allPartitions bool, startDate string, numPartitions int) error
}

//go:generate mockery --output=../mocks --all
package iface

import "context"

type ICostAllocationService interface {
	UpdateActiveCustomers(ctx context.Context) error
	UpdateMissingClusters(ctx context.Context) error
	InitStandaloneAccount(ctx context.Context, billingAccountID string) error

	ScheduleInitStandaloneAccounts(ctx context.Context, billingAccountIDs []string) error
}

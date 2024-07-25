//go:generate mockery --output=../mocks --all
package iface

import (
	"context"
)

type MarketplaceServiceIface interface {
	ResolveCustomer(ctx context.Context, awsSubscriptionID string) error
	ValidateEntitlement(ctx context.Context, awsSubscriptionID string) error
}

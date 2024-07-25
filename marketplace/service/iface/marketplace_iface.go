//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

type MarketplaceIface interface {
	ApproveAccount(ctx context.Context, accountID, email string) error
	RejectAccount(ctx context.Context, accountID, email string) error
	GetAccount(ctx context.Context, accountID string) (*domain.AccountFirestore, error)
	ApproveEntitlement(
		ctx context.Context,
		entitlementID string,
		email string,
		doitEmployee bool,
		approveFlexsaveProduct bool,
	) error
	RejectEntitlement(ctx context.Context, entitlementID, email string) error
	HandleCancelledEntitlement(ctx context.Context, entitlementID string) error
	PopulateBillingAccounts(
		ctx context.Context,
		populateBillingAccounts domain.PopulateBillingAccounts,
	) (domain.PopulateBillingAccountsResult, error)
	Subscribe(ctx context.Context, subscribePayload domain.SubscribePayload) error
	StandaloneApprove(ctx context.Context, customerID, billingAccountID string) error
}

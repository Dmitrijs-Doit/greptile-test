//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"google.golang.org/api/cloudcommerceprocurement/v1"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

type ProcurementDAL interface {
	ApproveAccount(ctx context.Context, accountID, reason string) error
	RejectAccount(ctx context.Context, accountID, reason string) error
	GetEntitlement(ctx context.Context, entitlementID string) (*cloudcommerceprocurement.Entitlement, error)
	ApproveEntitlement(ctx context.Context, entitlementID string) error
	RejectEntitlement(ctx context.Context, entitlementID, reason string) error
	ListEntitlements(ctx context.Context, filters ...dal.Filter) ([]*cloudcommerceprocurement.Entitlement, error)
	PublishAccountApprovalRequestEvent(ctx context.Context, payload domain.SubscribePayload) error
}

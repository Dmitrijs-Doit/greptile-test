//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

type IEntitlementFirestoreDAL interface {
	GetEntitlement(ctx context.Context, entitlementID string) (*domain.EntitlementFirestore, error)
}

//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
)

type AttributionGroupTierService interface {
	CheckAccessToExternalAttributionGroup(
		ctx context.Context,
		customerID string,
		attributionIDs []string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToCustomAttributionGroup(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToAttributionGroupID(
		ctx context.Context,
		customerID string,
		attributionGroupID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToPresetAttributionGroup(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
}

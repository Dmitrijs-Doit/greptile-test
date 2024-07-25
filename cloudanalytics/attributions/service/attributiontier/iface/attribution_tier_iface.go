//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
)

type AttributionTierService interface {
	CheckAccessToCustomAttribution(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToAttributionIDs(
		ctx context.Context,
		customerID string,
		attributionIDs []string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToAttributionID(
		ctx context.Context,
		customerID string,
		attributionID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToPresetAttribution(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
	CheckAccessToQueryRequest(
		ctx context.Context,
		customerID string,
		qr *cloudanalytics.QueryRequest,
	) (*domainTier.AccessDeniedError, error)
}

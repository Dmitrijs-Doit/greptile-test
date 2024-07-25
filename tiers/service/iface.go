package service

import (
	"context"

	"github.com/doitintl/firestore/pkg"
)

//go:generate mockery --name TierServiceIface --output ./mocks
type TierServiceIface interface {
	SendTrialNotifications(ctx context.Context, dryRun bool) error

	UpdateTier(ctx context.Context, ID string, upd *TierUpdateRequest) error
	CustomerCanAccessFeature(ctx context.Context, ID string, key pkg.TiersFeatureKey) (bool, error)
}

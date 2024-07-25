package service

import (
	"context"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/tiers/service"
)

type TierUpdateRequest struct {
	Entitlements []string `json:"entitlements"`
}

func (s *TiersService) UpdateTier(ctx context.Context, ID string, upd *TierUpdateRequest) error {
	return s.tiersSvc.UpdateTier(ctx, ID, &service.TierUpdate{
		Entitlements: upd.Entitlements,
	})
}

type CustomerFeatureAccessRequest struct {
	Key string `json:"key"`
}

func (s *TiersService) CustomerCanAccessFeature(ctx context.Context, ID string, key pkg.TiersFeatureKey) (bool, error) {
	return s.tiersSvc.CustomerCanAccessFeature(ctx, ID, key)
}

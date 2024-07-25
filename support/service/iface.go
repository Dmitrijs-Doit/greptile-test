package service

import (
	"context"

	"github.com/doitintl/firestore/pkg"
)

//go:generate mockery --output=./mocks --all
type SupportServiceInterface interface {
	ListPlatforms(ctx context.Context) (*PlatformsAPI, error)
	ListProducts(ctx context.Context, incomingPlatform string) (*ProductsAPI, error)
	ApplyNewSupportTier(ctx context.Context, customerID string, newTier pkg.TierNameType) error
	ApplyOneTimeSupport(ctx context.Context, customerID string, oneTime pkg.OneTimeProductType, email string) error
}

//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/support/domain"
)

type Support interface {
	ListPlatforms(ctx context.Context, isProductOnlySupported bool) ([]domain.Platform, error)
	ListProducts(ctx context.Context, platforms []string) ([]domain.Product, error)
}

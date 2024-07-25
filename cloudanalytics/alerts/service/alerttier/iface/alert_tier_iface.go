//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	domainTier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tier/domain"
)

type AlertTierService interface {
	CheckAccessToAlerts(
		ctx context.Context,
		customerID string,
	) (*domainTier.AccessDeniedError, error)
}

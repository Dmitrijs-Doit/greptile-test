package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/domain"
)

type BillingUpdate interface {
	ListBillingUpdateEvents(ctx context.Context) ([]*domain.BillingEvent, error)
	UpdateTimeCompleted(ctx context.Context, id string) error
}
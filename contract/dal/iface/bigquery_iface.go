package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
)

//go:generate mockery --name BigQuery --output ../mocks --case=underscore
type BigQuery interface {
	GetBillingAccountsSKU(ctx context.Context, startDate string, endDate string) ([]domain.SKUBillingRecord, error)
}

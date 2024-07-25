//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	domainDiscounts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/domain"
)

type IDiscountsService interface {
	UpdateDiscounts(ctx context.Context, data *domainDiscounts.DiscountsTableUpdateData) error
}

//go:generate mockery --name OptimizerService --output ../mocks --outpkg mocks --case=underscore
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
)

type OptimizerService interface {
	SingleCustomerOptimizer(ctx context.Context, customerID string, payload domain.Payload) error
	Schedule(ctx context.Context) (taskErrors []error, _ error)
}

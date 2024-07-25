//go:generate mockery --output=./mocks --all
package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config"
)

type Configs interface {
	GetExtendedMetrics(ctx context.Context) ([]config.ExtendedMetric, error)
}

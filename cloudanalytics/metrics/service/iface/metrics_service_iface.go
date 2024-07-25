//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/service"
)

type IMetricsService interface {
	ToInternal(
		ctx context.Context,
		customerID string,
		externalMetric *metrics.ExternalMetric,
	) (*metrics.InternalMetricParameters, []errormsg.ErrorMsg, error)
	ToExternal(params *metrics.InternalMetricParameters) (*metrics.ExternalMetric, []errormsg.ErrorMsg, error)
	DeleteMany(ctx context.Context, req service.DeleteMetricsRequest) error
}

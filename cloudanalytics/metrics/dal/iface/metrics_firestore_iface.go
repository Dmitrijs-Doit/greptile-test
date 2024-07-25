//go:generate mockery --output=../mocks --all
package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
)

type Metrics interface {
	GetRef(ctx context.Context, calculatedMetricID string) *firestore.DocumentRef
	Exists(ctx context.Context, calculatedMetricID string) (bool, error)
	GetCustomMetric(ctx context.Context, calculatedMetricID string) (*metrics.CalculatedMetric, error)
	GetMetricsUsingAttr(ctx context.Context, attrRef *firestore.DocumentRef) ([]*metrics.CalculatedMetric, error)
	DeleteMany(ctx context.Context, IDs []string) error
}

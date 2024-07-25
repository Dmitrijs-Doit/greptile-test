package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDAL "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsDALIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
)

const (
	MetricsCollection = "cloudAnalytics/metrics/cloudAnalyticsMetrics"
)

// MetricsFirestore is used to interact with cloud analytics metrics stored on Firestore.
type MetricsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	labelsDal          labelsDALIface.Labels
}

// NewMetricsFirestore returns a new MetricsFirestore instance with given project id.
func NewMetricsFirestore(ctx context.Context, projectID string) (*MetricsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	metricsFirestore := NewMetricsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return metricsFirestore, nil
}

// NewMetricsFirestoreWithClient returns a new MetricsFirestore using given client.
func NewMetricsFirestoreWithClient(fun connection.FirestoreFromContextFun) *MetricsFirestore {
	return &MetricsFirestore{
		fun,
		doitFirestore.DocumentHandler{},
		labelsDAL.NewLabelsFirestoreWithClient(fun),
	}
}

func (d *MetricsFirestore) Exists(ctx context.Context, calculatedMetricID string) (bool, error) {
	docRef := d.GetRef(ctx, calculatedMetricID)

	docSnap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil && status.Code(err) != codes.NotFound {
		return false, err
	}

	return docSnap.Exists(), nil
}

func (d *MetricsFirestore) GetRef(ctx context.Context, calculatedMetricID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection("cloudAnalytics").
		Doc("metrics").
		Collection("cloudAnalyticsMetrics").
		Doc(calculatedMetricID)
}

func (d *MetricsFirestore) GetMetricsUsingAttr(ctx context.Context, attrRef *firestore.DocumentRef) ([]*metrics.CalculatedMetric, error) {
	docs, err := d.firestoreClientFun(ctx).Collection(MetricsCollection).
		Where("variables", common.ArrayContainsAny, []map[string]interface{}{
			{
				"attribution": attrRef,
				"metric":      report.MetricCost,
			},
			{
				"attribution": attrRef,
				"metric":      report.MetricUsage,
			},
			{
				"attribution": attrRef,
				"metric":      report.MetricSavings,
			},
		}).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var calculatedMetrics []*metrics.CalculatedMetric

	for _, doc := range docs {
		var calculatedMetric *metrics.CalculatedMetric
		if err := doc.DataTo(&calculatedMetric); err != nil {
			return nil, err
		}

		calculatedMetric.ID = doc.Ref.ID

		calculatedMetrics = append(calculatedMetrics, calculatedMetric)
	}

	return calculatedMetrics, nil
}

// GetCustomMetric returns a custom metric by ID.
func (d *MetricsFirestore) GetCustomMetric(ctx context.Context, calculatedMetricID string) (*metrics.CalculatedMetric, error) {
	docRef := d.GetRef(ctx, calculatedMetricID)

	metricSnap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		return nil, err
	}

	var metric metrics.CalculatedMetric
	if err := metricSnap.DataTo(&metric); err != nil {
		return nil, err
	}

	return &metric, nil
}

func (d *MetricsFirestore) DeleteMany(ctx context.Context, IDs []string) error {
	metricsRefs := make([]*firestore.DocumentRef, len(IDs))

	for i, id := range IDs {
		mRef := d.GetRef(ctx, id)

		metricsRefs[i] = mRef
	}

	return d.labelsDal.DeleteManyObjectsWithLabels(ctx, metricsRefs)
}

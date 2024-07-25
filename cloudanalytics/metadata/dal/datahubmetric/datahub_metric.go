package datahubmetric

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
)

const (
	cloudAnalyticsCollection = "cloudAnalytics"
	datahubMetricCollection  = "datahubMetrics"

	metricsDoc = "metrics"
)

type DataHubMetricFirestore struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewDataHubMetricFirestore(ctx context.Context, projectID string) (*DataHubMetricFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewDataHubMetricFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		}), nil
}

func NewDataHubMetricFirestoreWithClient(fun firestoreIface.FirestoreFromContextFun) *DataHubMetricFirestore {
	return &DataHubMetricFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *DataHubMetricFirestore) GetRef(
	ctx context.Context,
	customerID string,
) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection(cloudAnalyticsCollection).
		Doc(metricsDoc).
		Collection(datahubMetricCollection).
		Doc(customerID)
}

func (d *DataHubMetricFirestore) GetMergeableDocument(
	tx *firestore.Transaction,
	docRef *firestore.DocumentRef,
) (*domain.DataHubMetrics, error) {
	docSnap, err := tx.Get(docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return &domain.DataHubMetrics{}, nil
		}

		return nil, err
	}

	var datahubMetric domain.DataHubMetrics

	if err := docSnap.DataTo(&datahubMetric); err != nil {
		return nil, err
	}

	return &datahubMetric, nil
}

func (d *DataHubMetricFirestore) Get(
	ctx context.Context,
	customerID string,
) (*domain.DataHubMetrics, error) {
	docSnap, err := d.GetRef(ctx, customerID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var metadata domain.DataHubMetrics

	if err := docSnap.DataTo(&metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// Delete deletes the customer datahub metrics document
func (d *DataHubMetricFirestore) Delete(
	ctx context.Context,
	customerID string,
) error {
	docRef := d.GetRef(ctx, customerID)
	_, err := d.documentsHandler.Delete(ctx, docRef)

	return err
}

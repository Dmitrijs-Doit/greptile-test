package dal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	cloudAnalyticsCollection        = "cloudAnalytics"
	configDoc                       = "configs"
	cloudAnalyticsConfigsCollection = "cloudAnalyticsConfigs"
	extendedMetricsDoc              = "extended-metrics"
)

// CreditsFirestore is used to interact with Credits stored on Firestore.
type ConfigsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewCreditsFirestoreWithClient returns a new CreditsFirestore using given client.
func NewConfigsFirestoreWithClient(fun connection.FirestoreFromContextFun) *ConfigsFirestore {
	return &ConfigsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (s *ConfigsFirestore) GetExtendedMetrics(ctx context.Context) ([]config.ExtendedMetric, error) {
	fs := s.firestoreClientFun(ctx)

	docRef := fs.Collection(cloudAnalyticsCollection).Doc(configDoc).Collection(cloudAnalyticsConfigsCollection).Doc(extendedMetricsDoc)

	docSnap, err := s.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var extendedMetrics config.ExtendedMetrics
	if err := docSnap.DataTo(&extendedMetrics); err != nil {
		return nil, err
	}

	return extendedMetrics.Metrics, nil
}

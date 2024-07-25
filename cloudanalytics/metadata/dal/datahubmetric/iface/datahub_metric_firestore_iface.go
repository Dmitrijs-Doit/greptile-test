//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	domain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/datahub"
)

type DataHubMetricFirestore interface {
	GetRef(ctx context.Context, customerID string) *firestore.DocumentRef
	GetMergeableDocument(
		tx *firestore.Transaction,
		docRef *firestore.DocumentRef,
	) (*domain.DataHubMetrics, error)
	Get(
		ctx context.Context,
		customerID string,
	) (*domain.DataHubMetrics, error)
	Delete(
		ctx context.Context,
		customerID string,
	) error
}

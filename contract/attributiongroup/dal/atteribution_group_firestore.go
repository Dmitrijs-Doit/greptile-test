package dal

import (
	"context"
	doitFirestore "github.com/doitintl/firestore"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	attributionCollection    = "cloudAnalyticsAttributionGroups"
	cloudAnalyticsCollection = "cloudAnalytics"
)

// AttributionGroupFirestore is used to interact with contracts stored on Firestore.
type AttributionGroupFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewAttributionGroupFirestore returns a new ContractsFirestore instance with given project id.
func NewAttributionGroupFirestore(ctx context.Context, projectID string) (*AttributionGroupFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewContractsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewContractsFirestoreWithClient returns a new ContractsFirestore using given client.
func NewContractsFirestoreWithClient(fun connection.FirestoreFromContextFun) *AttributionGroupFirestore {
	return &AttributionGroupFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *AttributionGroupFirestore) GetRampPlanEligibleSpendAttributionGroup(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	return d.firestoreClientFun(ctx).Collection(cloudAnalyticsCollection).
		Doc("attribution-groups").
		Collection(attributionCollection).
		Where("name", "==", "Ramp plan eligible spend").
		Documents(ctx).GetAll()
}

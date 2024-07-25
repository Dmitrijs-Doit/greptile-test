//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
)

type Labels interface {
	GetRef(ctx context.Context, labelID string) *firestore.DocumentRef
	Get(ctx context.Context, labelID string) (*labels.Label, error)
	GetLabels(ctx context.Context, labelIDs []string) ([]*labels.Label, error)
	Create(ctx context.Context, label *labels.Label) (*labels.Label, error)
	Update(ctx context.Context, labelID string, updates []firestore.Update) (*labels.Label, error)
	GetObjectLabels(ctx context.Context, obj *firestore.DocumentRef) ([]*firestore.DocumentRef, error)
	DeleteObjectWithLabels(ctx context.Context, deletedObjRef *firestore.DocumentRef) error
	DeleteManyObjectsWithLabels(ctx context.Context, deletedObjRefs []*firestore.DocumentRef) error
}

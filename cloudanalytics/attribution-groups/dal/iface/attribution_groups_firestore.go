//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

type AttributionGroups interface {
	GetRef(ctx context.Context, attributionGroupID string) *firestore.DocumentRef
	Get(ctx context.Context, id string) (*attributiongroups.AttributionGroup, error)
	GetAll(
		ctx context.Context,
		attributionGroupsRefs []*firestore.DocumentRef,
	) ([]*attributiongroups.AttributionGroup, error)
	GetByName(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		name string,
	) (*attributiongroups.AttributionGroup, error)
	Share(
		ctx context.Context,
		attributionGroupID string,
		collaborators []collab.Collaborator,
		public *collab.PublicAccess,
	) error
	Create(ctx context.Context, attributionGroup *attributiongroups.AttributionGroup) (string, error)
	Update(
		ctx context.Context,
		id string,
		attributionGroup *attributiongroups.AttributionGroup,
	) error
	Delete(ctx context.Context, id string) error
	List(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		email string,
	) ([]attributiongroups.AttributionGroup, error)
	GetByCustomer(
		ctx context.Context,
		customerRef *firestore.DocumentRef,
		attrRef *firestore.DocumentRef,
	) ([]*attributiongroups.AttributionGroup, error)
	GetByType(ctx context.Context, customerRef *firestore.DocumentRef, attrGroupType attribution.ObjectType) ([]*attributiongroups.AttributionGroup, error)
}

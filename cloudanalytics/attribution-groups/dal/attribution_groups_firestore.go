package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDal "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsDalIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
)

const (
	AttributionGroupsCollection = "cloudAnalytics/attribution-groups/cloudAnalyticsAttributionGroups"
	collaboratorsField          = "collaborators"
	publicField                 = "public"
)

// AttributionGroupsFirestore is used to interact with cloud analytics attributionGroups stored on Firestore.
type AttributionGroupsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	batchProvider      iface.BatchProvider
	labelsDal          labelsDalIface.Labels
}

// NewAttributionGroupsFirestore returns a new AttributionGroupsFirestore instance with given project id.
func NewAttributionGroupsFirestore(ctx context.Context, projectID string) (*AttributionGroupsFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	attributionGroupsFirestore := NewAttributionGroupsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return attributionGroupsFirestore, nil
}

// NewAttributionGroupsFirestoreWithClient returns a new AttributionGroupsFirestore using given client.
func NewAttributionGroupsFirestoreWithClient(fun connection.FirestoreFromContextFun) *AttributionGroupsFirestore {
	return &AttributionGroupsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
		batchProvider:      doitFirestore.NewBatchProvider(fun, 0),
		labelsDal:          labelsDal.NewLabelsFirestoreWithClient(fun),
	}
}

func (d *AttributionGroupsFirestore) getCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(AttributionGroupsCollection)
}

func (d *AttributionGroupsFirestore) GetByCustomer(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	attrRef *firestore.DocumentRef,
) ([]*attributiongroups.AttributionGroup, error) {
	allDocs, err := d.getCollection(ctx).
		Where("customer", "==", customerRef).
		Where("attributions", common.ArrayContains, attrRef).Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	var attributionGroups []*attributiongroups.AttributionGroup

	for _, doc := range allDocs {
		var attributionGroup *attributiongroups.AttributionGroup
		if err := doc.DataTo(&attributionGroup); err != nil {
			return nil, err
		}

		attributionGroup.ID = doc.Ref.ID

		attributionGroups = append(attributionGroups, attributionGroup)
	}

	return attributionGroups, nil

}

func (d *AttributionGroupsFirestore) GetRef(ctx context.Context, attributionGroupID string) *firestore.DocumentRef {
	return d.getCollection(ctx).Doc(attributionGroupID)
}

// Get returns a cloud analytics attribution group data.
func (d *AttributionGroupsFirestore) Get(ctx context.Context, attributionGroupID string) (*attributiongroups.AttributionGroup, error) {
	if attributionGroupID == "" {
		return nil, attributiongroups.ErrNoAttributionGroupID
	}

	docRef := d.GetRef(ctx, attributionGroupID)

	docSnap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, attributiongroups.ErrNotFound
		}

		return nil, err
	}

	var attributionGroup attributiongroups.AttributionGroup

	if err := docSnap.DataTo(&attributionGroup); err != nil {
		return nil, err
	}

	attributionGroup.ID = attributionGroupID

	return &attributionGroup, nil
}

func (d *AttributionGroupsFirestore) GetAll(
	ctx context.Context,
	attributionGroupsRefs []*firestore.DocumentRef,
) ([]*attributiongroups.AttributionGroup, error) {
	if len(attributionGroupsRefs) == 0 {
		return nil, domainAttributions.ErrEmptyAttributionRefsList
	}

	query := d.firestoreClientFun(ctx).Collection(AttributionGroupsCollection).Query.
		Where(firestore.DocumentID, "in", attributionGroupsRefs)

	allDocs, err := d.documentsHandler.GetAll(query.Documents(ctx))
	if err != nil {
		return nil, err
	}

	var attributionGroups []*attributiongroups.AttributionGroup

	for _, attr := range allDocs {
		var attributionGroup *attributiongroups.AttributionGroup
		if err := attr.DataTo(&attributionGroup); err != nil {
			return nil, err
		}

		attributionGroup.ID = attr.ID()

		attributionGroups = append(attributionGroups, attributionGroup)
	}

	return attributionGroups, nil
}

// GetByName returns a cloud analytics attribution group data by name.
func (d *AttributionGroupsFirestore) GetByName(ctx context.Context, customerRef *firestore.DocumentRef, name string) (*attributiongroups.AttributionGroup, error) {
	query := d.getCollection(ctx).Where("name", "==", name)

	if customerRef != nil {
		query = query.Where("type", "==", "custom").Where("customerRef", "==", customerRef)
	} else {
		query = query.Where("type", "==", "preset")
	}

	iter := query.Limit(1).Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, attributiongroups.ErrNotFound
	}

	docSnap := docSnaps[0]

	var attributionGroup attributiongroups.AttributionGroup

	if err := docSnap.DataTo(&attributionGroup); err != nil {
		return nil, err
	}

	attributionGroup.ID = docSnap.ID()

	return &attributionGroup, nil
}

// Share the attributionGroup with the given collaborators, and set the attributionGroup public access.
func (d *AttributionGroupsFirestore) Share(ctx context.Context, attributionGroupID string, collaborators []collab.Collaborator, public *collab.PublicAccess) error {
	attributionGroupRef := d.GetRef(ctx, attributionGroupID)

	if _, err := d.documentsHandler.Update(ctx, attributionGroupRef, []firestore.Update{
		{
			FieldPath: []string{collaboratorsField},
			Value:     collaborators,
		}, {
			FieldPath: []string{publicField},
			Value:     public,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (d *AttributionGroupsFirestore) Create(ctx context.Context, attributionGroup *attributiongroups.AttributionGroup) (string, error) {
	if attributionGroup == nil {
		return "", attributiongroups.ErrInvalidAttributionGroup
	}

	docRef := d.firestoreClientFun(ctx).Collection(AttributionGroupsCollection).NewDoc()

	if _, err := d.documentsHandler.Set(ctx, docRef, attributionGroup); err != nil {
		return "", err
	}

	return docRef.ID, nil
}

func (d *AttributionGroupsFirestore) Update(ctx context.Context, id string, attributionGroup *attributiongroups.AttributionGroup) error {
	if attributionGroup == nil {
		return attributiongroups.ErrInvalidAttributionGroup
	}

	if id == "" {
		return attributiongroups.ErrNoAttributionGroupID
	}

	docRef := d.GetRef(ctx, id)

	if _, err := d.documentsHandler.Update(ctx, docRef, []firestore.Update{
		{
			FieldPath: []string{"name"},
			Value:     attributionGroup.Name,
		},
		{
			FieldPath: []string{"description"},
			Value:     attributionGroup.Description,
		},
		{
			FieldPath: []string{"attributions"},
			Value:     attributionGroup.Attributions,
		},
		{
			FieldPath: []string{"timeModified"},
			Value:     firestore.ServerTimestamp,
		},
		{
			FieldPath: []string{"nullFallback"},
			Value:     attributionGroup.NullFallback,
		},
	}); err != nil {
		return err
	}

	return nil
}

func (d *AttributionGroupsFirestore) Delete(ctx context.Context, id string) error {
	if id == "" {
		return attributiongroups.ErrNoAttributionGroupID
	}

	docRef := d.GetRef(ctx, id)

	return d.labelsDal.DeleteObjectWithLabels(ctx, docRef)
}

func (d *AttributionGroupsFirestore) List(ctx context.Context, customerRef *firestore.DocumentRef, email string) ([]attributiongroups.AttributionGroup, error) {
	q := d.firestoreClientFun(ctx).
		Collection(AttributionGroupsCollection).
		Query.Where("customer", "==", nil).
		Where("type", "==", domainAttributions.ObjectTypePreset)

	docSnaps, err := d.documentsHandler.GetAll(q.Documents(ctx))
	if err != nil {
		return nil, err
	}

	q = d.firestoreClientFun(ctx).
		Collection(AttributionGroupsCollection).
		Query.Where("customer", "==", customerRef).
		Where("type", "in", []domainAttributions.ObjectType{domainAttributions.ObjectTypeManaged, domainAttributions.ObjectTypeCustom})

	docSnapsWCustomer, err := d.documentsHandler.GetAll(q.Documents(ctx))
	if err != nil {
		return nil, err
	}

	docSnaps = append(docSnaps, docSnapsWCustomer...)

	var sortableAg []attributiongroups.AttributionGroup

	for _, doc := range docSnaps {
		var a attributiongroups.AttributionGroup
		if err := doc.DataTo(&a); err != nil {
			return nil, err
		}

		a.ID = doc.ID()
		if a.CanView(email) {
			sortableAg = append(sortableAg, a)
		}
	}

	return sortableAg, nil
}

// GetByType returns a cloud analytics attribution group data by type.
func (d *AttributionGroupsFirestore) GetByType(ctx context.Context, customerRef *firestore.DocumentRef, attrGroupType domainAttributions.ObjectType) ([]*attributiongroups.AttributionGroup, error) {
	query := d.getCollection(ctx).Where("type", "==", attrGroupType).Where("customer", "==", customerRef)

	docSnaps, err := d.documentsHandler.GetAll(query.Documents(ctx))
	if err != nil {
		return nil, err
	}

	var attributionGroups []*attributiongroups.AttributionGroup

	for _, attr := range docSnaps {
		var attributionGroup *attributiongroups.AttributionGroup
		if err := attr.DataTo(&attributionGroup); err != nil {
			return nil, err
		}

		attributionGroup.ID = attr.ID()

		attributionGroups = append(attributionGroups, attributionGroup)
	}

	return attributionGroups, nil
}

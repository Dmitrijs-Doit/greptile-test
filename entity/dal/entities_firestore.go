package dal

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	stripeDomain "github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

const (
	entitiesCollection = "entities"
)

// EntitiesFirestore is used to interact with entities stored on Firestore.
type EntitiesFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewEntitiesFirestore returns a new EntitiesFirestore instance with given project id.
func NewEntitiesFirestore(ctx context.Context, projectID string) (*EntitiesFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewEntitiesFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewEntitiesFirestoreWithClient returns a new EntitiesFirestore using given client.
func NewEntitiesFirestoreWithClient(fun connection.FirestoreFromContextFun) *EntitiesFirestore {
	return &EntitiesFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *EntitiesFirestore) GetRef(ctx context.Context, entityID string) *firestore.DocumentRef {
	return d.GetEntitiesCollectionRef(ctx).Doc(entityID)
}

// GetEntity returns entity's data.
func (d *EntitiesFirestore) GetEntity(ctx context.Context, entityID string) (*common.Entity, error) {
	if entityID == "" {
		return nil, errors.New("invalid entity id")
	}

	doc := d.GetRef(ctx, entityID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var entity common.Entity

	err = snap.DataTo(&entity)
	if err != nil {
		return nil, err
	}

	entity.Snapshot = snap.Snapshot()

	return &entity, nil
}

func (d *EntitiesFirestore) GetCustomerEntities(ctx context.Context, customerRef *firestore.DocumentRef) ([]*common.Entity, error) {
	if customerRef == nil {
		return nil, ErrInvalidCustomerRef
	}

	entitiesDocs := d.GetEntitiesCollectionRef(ctx).Where("customer", "==", customerRef).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(entitiesDocs)
	if err != nil {
		return nil, err
	}

	entities := make([]*common.Entity, len(snaps))

	for i, snap := range snaps {
		if err := snap.DataTo(&entities[i]); err != nil {
			return nil, err
		}

		entities[i].Snapshot = snap.Snapshot()
	}

	return entities, nil
}

func (d *EntitiesFirestore) GetEntitiesCollectionRef(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(entitiesCollection)
}

func (d *EntitiesFirestore) GetEntities(ctx context.Context) ([]*common.Entity, error) {
	entitiesDocs := d.GetEntitiesCollectionRef(ctx).Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(entitiesDocs)
	if err != nil {
		return nil, err
	}

	entities := make([]*common.Entity, len(snaps))

	for i, snap := range snaps {
		if err := snap.DataTo(&entities[i]); err != nil {
			return nil, err
		}

		entities[i].Snapshot = snap.Snapshot()
	}

	return entities, nil
}

// ListActiveEntitiesForPayments returns entities with active status and payment types for a stripe account.
func (d *EntitiesFirestore) ListActiveEntitiesForPayments(
	ctx context.Context,
	stripeAccount stripeDomain.StripeAccountID,
	paymentTypes []common.EntityPaymentType,
) ([]*common.Entity, error) {
	if stripeAccount == "" {
		return nil, ErrInvalidEmptyStripeAccountID
	}

	if len(paymentTypes) == 0 {
		return nil, ErrInvalidEmptyPaymentTypes
	}

	entitiesDocs := d.GetEntitiesCollectionRef(ctx).
		Where("active", "==", true).
		Where("payment.accountId", "==", stripeAccount).
		Where("payment.type", "in", paymentTypes).
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(entitiesDocs)
	if err != nil {
		return nil, err
	}

	entities := make([]*common.Entity, len(snaps))

	for i, snap := range snaps {
		if err := snap.DataTo(&entities[i]); err != nil {
			return nil, err
		}

		entities[i].Snapshot = snap.Snapshot()
	}

	return entities, nil
}

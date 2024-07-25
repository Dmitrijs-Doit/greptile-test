package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared/domain"
)

// BillingUpdateFirestore is used to interact with SaaS Console update information stored on Firestore.
type BillingUpdateFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewBillingUpdateFirestore returns a new BillingUpdateFirestore instance with given project id.
func NewBillingUpdateFirestore(ctx context.Context, projectID string) (*BillingUpdateFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewBillingUpdateFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewBillingUpdateFirestoreWithClient returns a new BillingUpdateFirestore using given client.
func NewBillingUpdateFirestoreWithClient(fun connection.FirestoreFromContextFun) *BillingUpdateFirestore {
	return &BillingUpdateFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *BillingUpdateFirestore) GetRef(ctx context.Context, id string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(domain.BillingUpdateCollection).Doc(id)
}

func (d *BillingUpdateFirestore) BillingUpdateCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(domain.BillingUpdateCollection)
}

func (d *BillingUpdateFirestore) CreateBillingUpdateEvent(ctx context.Context, bue *domain.BillingEvent) error {
	_, err := d.GetRef(ctx, createBillingUpdateDocKey(bue)).Create(ctx, bue)
	if err != nil && status.Code(err) != codes.AlreadyExists {
		return err
	}

	return nil
}

func createBillingUpdateDocKey(bue *domain.BillingEvent) string {
	return fmt.Sprintf("%s-%d-%d", bue.BillingAccountID, bue.EventRange.StartTime.Unix(), bue.EventRange.EndTime.Unix())
}

func (d *BillingUpdateFirestore) getBillingUpdateFromIterator(iter *firestore.DocumentIterator) ([]*domain.BillingEvent, error) {
	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	billingUpdates := make([]*domain.BillingEvent, len(snaps))

	for i, snap := range snaps {
		var event domain.BillingEvent
		if err := snap.DataTo(&event); err != nil {
			return nil, err
		}

		event.Snapshot = snap.Snapshot()

		billingUpdates[i] = &event
	}

	return billingUpdates, nil
}

// ListBillingUpdateEvents returns a list of gcp BillingUpdate events.
func (d *BillingUpdateFirestore) ListBillingUpdateEvents(ctx context.Context) ([]*domain.BillingEvent, error) {
	iter := d.BillingUpdateCollection(ctx).Where("timeCompleted", "==", nil).Documents(ctx)

	return d.getBillingUpdateFromIterator(iter)
}

func (d *BillingUpdateFirestore) UpdateTimeCompleted(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("invalid document id")
	}

	doc := d.GetRef(ctx, id)

	if _, err := doc.Update(ctx, []firestore.Update{
		{Path: "timeCompleted", Value: time.Now()},
	}); err != nil {
		return err
	}

	return nil
}

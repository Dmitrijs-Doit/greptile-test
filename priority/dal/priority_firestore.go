package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

const (
	appCollection    = "app"
	avalaraStatusDoc = "avalara"

	avalaraMinPingInterval = 10 * time.Minute
	avalaraMaxPingInterval = 60 * time.Minute
)

type PriorityFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewPriorityFirestore(ctx context.Context, projectID string) (*PriorityFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewPriorityFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewPriorityFirestoreWithClient(fun connection.FirestoreFromContextFun) *PriorityFirestore {
	return &PriorityFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *PriorityFirestore) getAvalaraStatusRef(ctx context.Context) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(appCollection).Doc(avalaraStatusDoc)
}

func (d *PriorityFirestore) HandleAvalaraStatus(ctx context.Context) (bool, bool, error) {
	statusRef := d.getAvalaraStatusRef(ctx)

	var shouldPingAvalara, healthy bool

	if err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docSnap, err := tx.Get(statusRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				shouldPingAvalara = true
				return nil
			} else {
				return err
			}
		}

		var avalaraStatus priorityDomain.AvalaraStatus
		if err := docSnap.DataTo(&avalaraStatus); err != nil {
			return err
		}

		healthy = avalaraStatus.Healthy

		timeSinceLastPing := time.Since(avalaraStatus.Timestamp)
		if avalaraStatus.Locked {
			shouldPingAvalara = timeSinceLastPing > avalaraMaxPingInterval
			return nil
		} else if timeSinceLastPing < avalaraMinPingInterval {
			return nil
		}

		shouldPingAvalara = true

		return tx.Update(statusRef, []firestore.Update{
			{
				Path:  "locked",
				Value: true,
			},
		})
	}, firestore.MaxAttempts(10)); err != nil {
		return false, false, err
	}

	return shouldPingAvalara, healthy, nil
}

func (d *PriorityFirestore) SetAvalaraHealthyStatus(ctx context.Context, healthy bool) error {
	_, err := d.getAvalaraStatusRef(ctx).Set(ctx, priorityDomain.AvalaraStatus{
		Healthy:   healthy,
		Locked:    false,
		Timestamp: time.Now(),
	})

	return err
}

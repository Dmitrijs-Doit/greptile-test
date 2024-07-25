package dal

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

const webhookSubscriptionCollection = "integrations/zapier/webhooks"

//go:generate mockery --name WebhookSubscriptionDAL --output=./mocks
type WebhookSubscriptionDAL interface {
	Create(ctx context.Context, subscription *domain.WebhookSubscription) (string, error)
	Delete(ctx context.Context, subscriptionID string) error
	GetForDispatch(
		ctx context.Context,
		customer *firestore.DocumentRef,
		itemID string,
		event domain.EventType,
	) ([]*domain.WebhookSubscription, error)
}

// WebhookSubscriptionFirestore is used to interact with webhook subscriptions stored on Firestore.
type WebhookSubscriptionFirestore struct {
	firestoreClientFn connection.FirestoreFromContextFun
	dh                firestoreIface.DocumentsHandler
	l                 logger.Provider
}

// NewWebhookSubscriptionsFirestore returns a new WebhooksSubscriptionFirestore instance with given project id.
func NewWebhookSubscriptionsFirestore(ctx context.Context, projectID string) (*WebhookSubscriptionFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewWebhookSubscriptionsFirestoreWithClient(
		func(ctx context.Context) logger.ILogger {
			return logger.FromContext(ctx)
		},
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewWebhookSubscriptionsFirestoreWithClient returns a new WebhookSubscriptionFirestore using given client.
func NewWebhookSubscriptionsFirestoreWithClient(log logger.Provider, fun connection.FirestoreFromContextFun) *WebhookSubscriptionFirestore {
	return &WebhookSubscriptionFirestore{
		firestoreClientFn: fun,
		dh:                doitFirestore.DocumentHandler{},
		l:                 log,
	}
}

// Create creates a webhook subscription in firestore
func (ws *WebhookSubscriptionFirestore) Create(
	ctx context.Context,
	subscription *domain.WebhookSubscription,
) (string, error) {
	if subscription == nil {
		return "", errors.New("subscription cannot be nil")
	}

	ref, _, err := ws.
		firestoreClientFn(ctx).
		Collection(webhookSubscriptionCollection).
		Add(ctx, subscription)
	if err != nil {
		return "", err
	}

	return ref.ID, nil
}

// Delete deletes a webhook subscription from firestore
func (ws *WebhookSubscriptionFirestore) Delete(
	ctx context.Context,
	subscriptionID string,
) error {
	if subscriptionID == "" {
		return errors.New("invalid subscription ID")
	}

	doc := ws.
		firestoreClientFn(ctx).
		Collection(webhookSubscriptionCollection).
		Doc(subscriptionID)
	_, err := doc.Delete(ctx)

	return err
}

func (ws *WebhookSubscriptionFirestore) GetForDispatch(
	ctx context.Context,
	customer *firestore.DocumentRef,
	itemID string,
	event domain.EventType,
) ([]*domain.WebhookSubscription, error) {
	docs, err := ws.firestoreClientFn(ctx).Collection(webhookSubscriptionCollection).Query.
		Where("customer", "==", customer).
		Where("itemId", "==", itemID).
		Where("eventType", "==", event).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	subs := make([]*domain.WebhookSubscription, 0, len(docs))

	for _, doc := range docs {
		var s domain.WebhookSubscription
		if err := doc.DataTo(&s); err != nil {
			ws.l(ctx).Warningf("unable to convert to webhook subscription: %s", err)
			continue
		}

		s.ID = doc.Ref.ID
		subs = append(subs, &s)
	}

	return subs, nil
}

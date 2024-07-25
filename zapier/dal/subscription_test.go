package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

func TestNewWebhookSubscriptionFirestore(t *testing.T) {
	_, err := NewWebhookSubscriptionsFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewWebhookSubscriptionsFirestoreWithClient(nil, nil)
	assert.NotNil(t, d)
}

func TestWebhookSubscription_Create(t *testing.T) {
	ctx := context.Background()

	customer := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/ImoC9XkrutBysJvyqlBm",
		ID:   "test-customer-id",
	}

	type args struct {
		ctx          context.Context
		subscription *domain.WebhookSubscription
	}

	subscription := domain.WebhookSubscription{
		Customer:  customer,
		UserEmail: "test@test.com",
		EventType: domain.AlertConditionSatisfied,
		ItemID:    "test-entity-id",
		TargetURL: "https://test.com/test",
	}

	tt := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "success creating webhook subscription",
			args: args{
				ctx:          ctx,
				subscription: &subscription,
			},
		},
		{
			name: "failure creating webhook subscription - invalid subscription",
			args: args{
				ctx:          ctx,
				subscription: nil,
			},
			expectedErr: errors.New("subscription cannot be nil"),
		},
	}

	ws, err := NewWebhookSubscriptionsFirestore(context.Background(), common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			_, err := ws.Create(test.args.ctx, test.args.subscription)

			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestWebhookSubscription_Delete(t *testing.T) {
	ctx := context.Background()

	ws, err := NewWebhookSubscriptionsFirestore(context.Background(), common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	customer := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/ImoC9XkrutBysJvyqlBm",
		ID:   "test-customer-id",
	}

	subscription := domain.WebhookSubscription{
		Customer:  customer,
		UserEmail: "test@test.com",
		EventType: domain.AlertConditionSatisfied,
		ItemID:    "test-entity-id",
		TargetURL: "https://test.com/test",
	}

	subscriptionID, _ := ws.Create(ctx, &subscription)

	type args struct {
		ctx            context.Context
		subscriptionID string
	}

	tt := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "success deleting webhook subscription",
			args: args{
				ctx:            ctx,
				subscriptionID: subscriptionID,
			},
		},
		{
			name: "failure deleting webhook subscription - invalid subscription ID",
			args: args{
				ctx:            ctx,
				subscriptionID: "",
			},
			expectedErr: errors.New("invalid subscription ID"),
		},
		{
			name: "delete subscription that doesnt exist",
			args: args{
				ctx:            ctx,
				subscriptionID: "not-a-real-id",
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			err := ws.Delete(test.args.ctx, test.args.subscriptionID)

			assert.Equal(t, test.expectedErr, err)
		})
	}
}

func TestWebhookSubscription_GetForDispatch(t *testing.T) {
	ctx := context.Background()

	ws, err := NewWebhookSubscriptionsFirestore(context.Background(), common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	var (
		itemID = "test-item-id"
		event  = domain.AlertConditionSatisfied
	)

	customer := &firestore.DocumentRef{
		Path: "projects/doitintl-cmp-dev/databases/(default)/documents/customers/ImoC9XkrutBysJvyqlBm",
		ID:   "test-customer-id",
	}

	subscription := domain.WebhookSubscription{
		Customer:  customer,
		UserEmail: "test@test.com",
		EventType: event,
		ItemID:    itemID,
		TargetURL: "https://test.com/test",
	}

	_, err = ws.Create(ctx, &subscription)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	type args struct {
		ctx      context.Context
		customer *firestore.DocumentRef
		itemID   string
		event    domain.EventType
	}

	tt := []struct {
		name        string
		args        args
		expectedErr error
	}{
		{
			name: "success finding webhook subscriptions",
			args: args{
				ctx:      ctx,
				customer: customer,
				itemID:   itemID,
				event:    event,
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			subs, err := ws.GetForDispatch(test.args.ctx, test.args.customer, test.args.itemID, test.args.event)

			assert.Equal(t, test.expectedErr, err)
			assert.Len(t, subs, 1)
			assert.Equal(t, subscription.Customer, customer)
			assert.Equal(t, subscription.EventType, event)
			assert.Equal(t, subscription.ItemID, itemID)
		})
	}
}

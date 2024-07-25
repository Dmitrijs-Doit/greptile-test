package dal

import (
	"context"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/tests"
)

func TestNewAccountUpdateDAL(t *testing.T) {
	_, err := NewBillingUpdateFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	a := NewBillingUpdateFirestoreWithClient(nil)
	assert.NotNil(t, a)
}

func NewAccountUpdateFirestoreWithClientMock(ctx context.Context) *BillingUpdateFirestore {
	logging, err := logger.NewLogging(ctx)
	if err != nil {
		log.Printf("main: could not initialize logging. error %s", err)
		return nil
	}

	// Initialize db connections clients
	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		log.Printf("main: could not initialize db connections. error %s", err)
		return nil
	}

	return NewBillingUpdateFirestoreWithClient(conn.Firestore)
}

func TestBillingUpdateFirestore_ListBillingUpdateEvents(t *testing.T) {
	ctx := context.Background()
	s := NewAccountUpdateFirestoreWithClientMock(ctx)

	if err := tests.LoadTestData("GCPBillingCopyData"); err != nil {
		t.Error(err)
	}

	events, err := s.ListBillingUpdateEvents(ctx)
	if err != nil {
		t.Error(err)
	}

	expectedEvent := events[0]
	// filters out timeCompleted that is not null
	assert.Equal(t, len(events), 1)
	assert.Equal(t, expectedEvent.ID(), "0UL9qjiByFFmQwSif88P")
	assert.Equal(t, expectedEvent.BillingAccountID, "123ABC-123ABC-123ABC")
	assert.Equal(t, string(expectedEvent.EventType), "onboarding")
}

func TestBillingUpdateFirestore_UpdateTimeCompleted(t *testing.T) {
	ctx := context.Background()
	d := NewAccountUpdateFirestoreWithClientMock(ctx)

	const testID = "0UL9qjiByFFmQwSif88P"

	if err := tests.LoadTestData("GCPBillingCopyData"); err != nil {
		t.Error(err)
	}

	type args struct {
		ctx context.Context
		id  string
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: ctx,
				id:  testID,
			},
		},
		{
			name: "invalid id",
			args: args{
				ctx: ctx,
				id:  "",
			},
			wantErr: true,
		},
		{
			name: "id that does not exist",
			args: args{
				ctx: ctx,
				id:  "abcd",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.UpdateTimeCompleted(tt.args.ctx, tt.args.id); err != nil {
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					t.Errorf("UpdateTimeCompleted error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			if tt.name == "success" {
				snap, err := d.GetRef(ctx, testID).Get(ctx)
				if err != nil {
					t.Error(err)
				}

				var event domain.BillingEvent
				if err := snap.DataTo(&event); err != nil {
					t.Error(err)
				}

				assert.NotNil(t, event.TimeCompleted)
			}
		})
	}
}

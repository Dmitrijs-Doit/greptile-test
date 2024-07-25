package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	userMocks "github.com/doitintl/hello/scheduled-tasks/algolia/dal/mocks"
	alertMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	budgetMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/zapier/dal/mocks"
)

func TestNewWebhookSubscriptionService(t *testing.T) {
	ctx := context.Background()

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	s := NewWebhookSubscriptionService(func(ctx context.Context) logger.ILogger {
		return &loggerMocks.ILogger{}
	}, conn)

	assert.NotNil(t, s)
}

func TestWebhookSubscriptionService_Create(t *testing.T) {
	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		dal                mocks.WebhookSubscriptionDAL
		userDAL            userMocks.UserDAL
		alertsDAL          alertMocks.Alerts
		budgetsDAL         budgetMocks.Budgets
	}

	type args struct {
		ctx                  context.Context
		createWebhookRequest *CreateWebhookRequest
	}

	ctx := context.Background()

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	var (
		customerID       = "fake-customer-ID"
		userID           = "fake-user-ID"
		userEmail        = "fake@test.com"
		targetURL        = "https://test.com/test"
		eventType        = "alertConditionSatisfied"
		invalidEventType = "invalid_event"
		itemID           = "fake-customer-ID"
	)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully created webhook subscription",
			args: args{
				ctx: ctx,
				createWebhookRequest: &CreateWebhookRequest{
					CustomerID: customerID,
					UserID:     userID,
					UserEmail:  userEmail,
					TargetURL:  targetURL,
					EventType:  eventType,
					ItemID:     itemID,
				},
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.userDAL.On("GetUser", ctx, userID).
					Return(&common.User{
						ID:          userID,
						Permissions: []string{string(common.PermissionCloudAnalytics)},
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{ID: customerID},
						},
					}, nil)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.AnythingOfType("*common.User")).Return(true)
				f.dal.On("Create", ctx, mock.AnythingOfType("*domain.WebhookSubscription")).Return("possible-id", nil)

				view := collab.PublicAccessView
				f.alertsDAL.On("GetAlert", ctx, itemID).Return(&domain.Alert{
					Access: collab.Access{
						Public: &view,
					},
				}, nil)
			},
		},
		{
			name: "failed to create webhook subscription - invalid event type",
			args: args{
				ctx: ctx,
				createWebhookRequest: &CreateWebhookRequest{
					CustomerID: customerID,
					UserID:     userID,
					UserEmail:  userEmail,
					TargetURL:  targetURL,
					EventType:  invalidEventType,
					ItemID:     itemID,
				},
			},
			on: func(f *fields) {
				f.userDAL.On("GetUser", ctx, userID).
					Return(&common.User{
						ID:          userID,
						Permissions: []string{string(common.PermissionCloudAnalytics)},
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{ID: customerID},
						},
					}, nil)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.AnythingOfType("*common.User")).Return(true)
				f.budgetsDAL.On("GetBudget", ctx, itemID).Return(nil, errors.New(""))
			},
			expectedErr: errors.New("invalid entity"),
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProviderMock: loggerMocks.ILogger{},
				dal:                mocks.WebhookSubscriptionDAL{},
				userDAL:            userMocks.UserDAL{},
				alertsDAL:          alertMocks.Alerts{},
				budgetsDAL:         budgetMocks.Budgets{},
			}

			s := &WebhookService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &tt.fields.loggerProviderMock
				},
				conn:      conn,
				dal:       &tt.fields.dal,
				userDAL:   &tt.fields.userDAL,
				validator: NewEventValidator(&tt.fields.alertsDAL, &tt.fields.budgetsDAL),
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, err := s.CreateSubscription(tt.args.ctx, tt.args.createWebhookRequest)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

func TestWebhookSubscriptionService_Delete(t *testing.T) {
	type fields struct {
		loggerProviderMock loggerMocks.ILogger
		dal                mocks.WebhookSubscriptionDAL
		userDAL            userMocks.UserDAL
	}

	type args struct {
		ctx                  context.Context
		deleteWebhookRequest *DeleteWebhookRequest
	}

	ctx := context.Background()

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	var (
		customerID     = "fake-customer-ID"
		userID         = "fake-user-ID"
		userEmail      = "fake@test.com"
		subscriptionID = "subscription-id"
	)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully deleted webhook subscription",
			args: args{
				ctx: ctx,
				deleteWebhookRequest: &DeleteWebhookRequest{
					CustomerID:     customerID,
					UserID:         userID,
					UserEmail:      userEmail,
					SubscriptionID: subscriptionID,
				},
			},
			expectedErr: nil,
			on: func(f *fields) {
				f.userDAL.On("GetUser", ctx, userID).
					Return(&common.User{
						ID:          userID,
						Permissions: []string{string(common.PermissionCloudAnalytics)},
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{ID: customerID},
						},
					}, nil)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.AnythingOfType("*common.User")).Return(true)
				f.dal.On("Delete", ctx, mock.AnythingOfType("string")).Return(nil)
			},
		},
		{
			name: "failed to delete webhook subscription - user not found",
			args: args{
				ctx: ctx,
				deleteWebhookRequest: &DeleteWebhookRequest{
					CustomerID:     customerID,
					UserID:         userID,
					UserEmail:      userEmail,
					SubscriptionID: subscriptionID,
				},
			},
			on: func(f *fields) {
				f.userDAL.On("GetUser", ctx, userID).Return(nil, errors.New("user not found"))
			},
			expectedErr: errors.New("user not found"),
		},
		{
			name: "failed to delete webhook subscription - user does not belong",
			args: args{
				ctx: ctx,
				deleteWebhookRequest: &DeleteWebhookRequest{
					CustomerID:     customerID,
					UserID:         userID,
					UserEmail:      userEmail,
					SubscriptionID: subscriptionID,
				},
			},
			on: func(f *fields) {
				f.userDAL.On("GetUser", ctx, userID).
					Return(&common.User{
						ID:          userID,
						Permissions: []string{string(common.PermissionCloudAnalytics)},
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{ID: "invalid-customer-id"},
						},
					}, nil)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.AnythingOfType("*common.User")).Return(true)
			},
			expectedErr: errors.New("user does not belong to this organization"),
		},
		{
			name: "failed to delete webhook subscription - user not authorized",
			args: args{
				ctx: ctx,
				deleteWebhookRequest: &DeleteWebhookRequest{
					CustomerID:     customerID,
					UserID:         userID,
					UserEmail:      userEmail,
					SubscriptionID: subscriptionID,
				},
			},
			on: func(f *fields) {
				f.userDAL.On("GetUser", ctx, userID).
					Return(&common.User{
						ID:          userID,
						Permissions: []string{},
						Customer: common.UserCustomer{
							Ref: &firestore.DocumentRef{ID: customerID},
						},
					}, nil)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.AnythingOfType("*common.User")).Return(false)
			},
			expectedErr: errors.New("user is not authorized"),
		},
	}

	for i := 0; i < len(tests); i++ {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProviderMock: loggerMocks.ILogger{},
				dal:                mocks.WebhookSubscriptionDAL{},
				userDAL:            userMocks.UserDAL{},
			}

			s := &WebhookService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &tt.fields.loggerProviderMock
				},
				conn:    conn,
				dal:     &tt.fields.dal,
				userDAL: &tt.fields.userDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.DeleteSubscription(tt.args.ctx, tt.args.deleteWebhookRequest)

			assert.Equal(t, tt.expectedErr, err)
		})
	}
}

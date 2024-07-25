package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	"github.com/doitintl/http"
	"github.com/doitintl/http/mocks"
	doitPubsubIface "github.com/doitintl/pubsub/iface"
	doitPubsubMocks "github.com/doitintl/pubsub/mocks"
)

func TestNewProcurementDAL(t *testing.T) {
	type args struct {
		procurementClient http.IClient
		topicHandler      doitPubsubIface.TopicHandler
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				procurementClient: mocks.NewIClient(t),
				topicHandler:      &doitPubsubMocks.TopicHandler{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProcurementDAL(tt.args.procurementClient, tt.args.topicHandler)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMarketplaceDAL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProcurementDAL_ApproveAccount(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
	}

	type args struct {
		ctx       context.Context
		accountID string
		reason    string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "Success",
			args: args{
				ctx:       context.Background(),
				accountID: "1",
				reason:    "approve reason",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, nil).Once()
			},
		},
		{
			name: "Error on Post",
			args: args{
				ctx:       context.Background(),
				accountID: "1",
				reason:    "approve reason",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, errors.New("error")).Once()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
			}
			s := &ProcurementDAL{
				procurementClient: fields.procurementClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.ApproveAccount(tt.args.ctx, tt.args.accountID, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.ApproveAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcurementDAL_RejectAccount(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
	}

	type args struct {
		ctx       context.Context
		accountID string
		reason    string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "Success",
			args: args{
				ctx:       context.Background(),
				accountID: "1",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, nil).Once()
			},
		},
		{
			name: "Error on Post",
			args: args{
				ctx:       context.Background(),
				accountID: "1",
				reason:    "reason",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, errors.New("error")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
			}
			s := &ProcurementDAL{
				procurementClient: fields.procurementClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.RejectAccount(tt.args.ctx, tt.args.accountID, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.RejectAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcurementDAL_GetEntitlement(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
	}

	type args struct {
		ctx           context.Context
		entitlementID string
	}

	entitlementID := "12345"

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "get entitlement",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
			},
			on: func(f *fields) {
				f.procurementClient.On(
					"Get",
					testutils.ContextBackgroundMock,
					matchByURL(fmt.Sprintf("/v1/entitlements/%s", entitlementID)),
				).
					Return(nil, nil).Once()
			},
		},
		{
			name: "error on get entitlement",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
			},
			wantErr: true,
			on: func(f *fields) {
				f.procurementClient.On(
					"Get",
					testutils.ContextBackgroundMock,
					matchByURL(fmt.Sprintf("/v1/entitlements/%s", entitlementID)),
				).
					Return(nil, errors.New("some error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
			}
			s := &ProcurementDAL{
				procurementClient: fields.procurementClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if _, err := s.GetEntitlement(tt.args.ctx, tt.args.entitlementID); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.GetEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcurementDAL_ApproveEntitlement(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
	}

	type args struct {
		ctx           context.Context
		entitlementID string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "Success",
			args: args{
				ctx:           context.Background(),
				entitlementID: "1",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, nil).Once()
			},
		},
		{
			name: "Error on Post",
			args: args{
				ctx:           context.Background(),
				entitlementID: "1",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, errors.New("error")).Once()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
			}
			s := &ProcurementDAL{
				procurementClient: fields.procurementClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.ApproveEntitlement(tt.args.ctx, tt.args.entitlementID); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.ApproveEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcurementDAL_RejectEntitlement(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
	}

	type args struct {
		ctx           context.Context
		entitlementID string
		reason        string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "Success",
			args: args{
				ctx:           context.Background(),
				entitlementID: "1",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, nil).Once()
			},
		},
		{
			name: "Error on Post",
			args: args{
				ctx:           context.Background(),
				entitlementID: "1",
				reason:        "reason",
			},
			on: func(f *fields) {
				f.procurementClient.On("Post", mock.Anything, mock.Anything).Return(nil, errors.New("error")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
			}
			s := &ProcurementDAL{
				procurementClient: fields.procurementClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.RejectEntitlement(tt.args.ctx, tt.args.entitlementID, tt.args.reason); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.RejectEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcurementDAL_ListEntitlements(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
	}

	type args struct {
		ctx           context.Context
		entitlementID string
		filters       []Filter
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "get entitlements without filters",
			args: args{
				ctx:           context.Background(),
				entitlementID: "someEntitlement",
				filters:       nil,
			},
			on: func(f *fields) {
				f.procurementClient.On("Get", testutils.ContextBackgroundMock, matchByURL("/v1/entitlements")).Return(nil, nil).Once()
			},
		},
		{
			name: "get entitlements with filters",
			args: args{
				ctx:           context.Background(),
				entitlementID: "someEntitlement",
				filters: []Filter{
					{
						Key:   EntitlementFilterKeyAccount,
						Value: "111",
					},
					{
						Key:   EntitlementFilterKeyCustomerBillingAccount,
						Value: "12345",
					},
				},
			},
			on: func(f *fields) {
				f.procurementClient.On("Get", testutils.ContextBackgroundMock, matchByURL("/v1/entitlements?account=111&customer_billing_account=12345")).Return(nil, nil).Once()
			},
		},
		{
			name: "error on get",
			args: args{
				ctx:           context.Background(),
				entitlementID: "someEntitlement",
				filters:       nil,
			},
			on: func(f *fields) {
				f.procurementClient.On("Get", testutils.ContextBackgroundMock, matchByURL("/v1/entitlements")).Return(nil, errors.New("error")).Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
			}
			s := &ProcurementDAL{
				procurementClient: fields.procurementClient,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if _, err := s.ListEntitlements(tt.args.ctx, tt.args.filters...); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.ListEntitlements() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcurementDAL_PublishAccountApprovalRequestEvent(t *testing.T) {
	type fields struct {
		procurementClient *mocks.IClient
		topicHandler      *doitPubsubMocks.TopicHandler
	}

	type args struct {
		ctx     context.Context
		payload domain.SubscribePayload
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "publish account approval request",
			args: args{
				ctx: context.Background(),
				payload: domain.SubscribePayload{
					ProcurementAccountID: "11111",
					CustomerID:           "22222",
					Email:                "test@test.com",
					UID:                  "33333",
				},
			},
			on: func(f *fields) {
				f.topicHandler.On(
					"Publish",
					testutils.ContextBackgroundMock,
					mock.AnythingOfType("*pubsub.Message"),
				).
					Return(nil).Once()
			},
		},
		{
			name: "returns error if publishing fails",
			args: args{
				ctx:     context.Background(),
				payload: domain.SubscribePayload{},
			},
			wantErr: true,
			on: func(f *fields) {
				f.topicHandler.On(
					"Publish",
					testutils.ContextBackgroundMock,
					mock.AnythingOfType("*pubsub.Message"),
				).
					Return(errors.New("some pubsub error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				procurementClient: &mocks.IClient{},
				topicHandler:      &doitPubsubMocks.TopicHandler{},
			}

			s, err := NewProcurementDAL(
				fields.procurementClient,
				fields.topicHandler,
			)
			if err != nil {
				t.Error(err)
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.PublishAccountApprovalRequestEvent(
				tt.args.ctx,
				tt.args.payload,
			); (err != nil) != tt.wantErr {
				t.Errorf("ProcurementDAL.PublishAccountApprovalRequestEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func matchByURL(url string) interface{} {
	return mock.MatchedBy(func(i interface{}) bool {
		httpRequest := i.(*http.Request)
		return httpRequest.URL == url
	})
}

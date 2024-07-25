package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/service/mocks"
	doitPubsubIface "github.com/doitintl/pubsub/iface"
	doitPubsubMocks "github.com/doitintl/pubsub/mocks"
)

func mockTopicHandlerProvider(isProduction bool) (doitPubsubIface.TopicHandler, error) {
	return &doitPubsubMocks.TopicHandler{}, nil
}

func getConnection() (*connection.Connection, error) {
	ctx := context.Background()

	log, err := logger.NewLogging(ctx)
	if err != nil {
		return nil, err
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func TestNewMarketplace(t *testing.T) {
	type args struct {
		log                  logger.Provider
		conn                 *connection.Connection
		topicHandlerProvider TopicHandlerProviderIface
	}

	conn, err := getConnection()
	if err != nil {
		t.Error(err)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				log:                  logger.FromContext,
				conn:                 conn,
				topicHandlerProvider: mockTopicHandlerProvider,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMarketplace(tt.args.log, tt.args.conn, tt.args.topicHandlerProvider)
			if got == nil {
				t.Errorf("NewMarketplace() error = %v, wantErr %v", got, tt.wantErr)
			}
		})
	}
}

const (
	// mock account id uuid4
	accountID     = "aaaaaaaa-b7b8-b9ba-bbbd-bebfc0c1c2c3"
	entitlementID = "eeeeeeee-b7b8-b9ba-bbbd-bebfc0c1c2c3"
	email         = "user@example.com"
)

func TestMarketplaceGCP_ApproveAccount(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = []gin.Param{
		{Key: "accountID", Value: accountID},
	}

	ctx.Set("email", email)

	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.On("ApproveAccount", mock.AnythingOfType("*gin.Context"), accountID, email).Return(nil).Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Error on ApproveAccount",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.On("ApproveAccount", mock.AnythingOfType("*gin.Context"), accountID, email).Return(errors.New("error")).Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{service: mocks.MarketplaceIface{}}
			h := &MarketplaceGCP{
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			respond := h.ApproveAccount(tt.args.ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("MarketplaceGCP.ApproveAccount() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMarketplaceGCP_RejectAccount(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = []gin.Param{
		{Key: "accountID", Value: accountID},
	}

	ctx.Set("email", email)

	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.On("RejectAccount", mock.Anything, accountID, email).Return(nil).Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Error on RejectAccount",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.On("RejectAccount", mock.Anything, accountID, email).Return(errors.New("error")).Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{service: mocks.MarketplaceIface{}}

			h := &MarketplaceGCP{
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			respond := h.RejectAccount(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("MarketplaceGCP.RejectAccount() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMarketplaceGCP_ApproveEntitlement(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = []gin.Param{
		{Key: "entitlementID", Value: entitlementID},
	}

	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		ctx          *gin.Context
		email        string
		doitEmployee bool
	}

	email := "user@doit.com"

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx:          ctx,
				doitEmployee: true,
				email:        email,
			},
			on: func(f *fields) {
				f.service.On(
					"ApproveEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
					email,
					true,
					true,
				).Return(nil).Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "return error code when customer is not eligible for flexsave",
			args: args{
				ctx:          ctx,
				doitEmployee: true,
				email:        email,
			},
			on: func(f *fields) {
				f.service.On(
					"ApproveEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
					email,
					true,
					true,
				).Return(service.ErrCustomerIsNotEligibleFlexsave).Once()
			},
			wantErr: false,
		},
		{
			name: "Error on ApproveEntitlement",
			args: args{
				ctx:          ctx,
				doitEmployee: true,
				email:        email,
			},
			on: func(f *fields) {
				f.service.On(
					"ApproveEntitlement",
					mock.AnythingOfType("*gin.Context"),
					entitlementID,
					email,
					true,
					true,
				).Return(errors.New("error")).Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{service: mocks.MarketplaceIface{}}
			h := &MarketplaceGCP{
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			ctx.Set("doitEmployee", tt.args.doitEmployee)
			ctx.Set("email", tt.args.email)

			respond := h.ApproveEntitlement(tt.args.ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("MarketplaceGCP.ApproveEntitlement() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMarketplaceGCP_RejectEntitlement(t *testing.T) {
	recorder := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(recorder)

	ctx.Params = []gin.Param{
		{Key: "entitlementID", Value: entitlementID}}

	ctx.Set("email", email)

	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.On("RejectEntitlement", mock.Anything, entitlementID, email).Return(nil).Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Error on RejectEntitlement",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.On("RejectEntitlement", mock.Anything, entitlementID, email).Return(errors.New("error")).Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{service: mocks.MarketplaceIface{}}

			h := &MarketplaceGCP{
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			respond := h.RejectEntitlement(tt.args.ctx)

			if (respond != nil) != tt.wantErr {
				t.Errorf("MarketplaceGCP.RejectEntitlement() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMarketplaceGCP_PopulateBillingAccounts(t *testing.T) {
	bodyReader := strings.NewReader("[]")
	request := httptest.NewRequest(http.MethodPost, "/someRequest", bodyReader)
	recorder := httptest.NewRecorder()

	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	type fields struct {
		service mocks.MarketplaceIface
	}

	type args struct {
		ctx *gin.Context
	}

	tests := []struct {
		name         string
		args         args
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "Happy path on PopulateBillingAccounts",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.
					On("PopulateBillingAccounts",
						mock.AnythingOfType("*gin.Context"),
						mock.AnythingOfType("domain.PopulateBillingAccounts"),
					).
					Return(nil, nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "Error on PopulateBillingAccounts",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.service.
					On("PopulateBillingAccounts",
						mock.AnythingOfType("*gin.Context"),
						mock.AnythingOfType("domain.PopulateBillingAccounts"),
					).
					Return(nil, errors.New("error")).
					Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{service: mocks.MarketplaceIface{}}
			h := &MarketplaceGCP{
				service: &fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			respond := h.PopulateBillingAccounts(tt.args.ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("MarketplaceGCP.PopulateBillingAccounts() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

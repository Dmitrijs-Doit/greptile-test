package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	sharedFirestoreMocks "github.com/doitintl/firestore/mocks"
	firestorePkg "github.com/doitintl/firestore/pkg"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	flexsaveResoldMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	slackMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/service/slack/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	userMocks "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
	userDomain "github.com/doitintl/hello/scheduled-tasks/user/domain"
)

const (
	entitlementID = "bbbbbbbb-b7b8-b9ba-bbbc-bdbebfc0c1c2"
	email         = "user@doit.com"
)

const (
	anomalyDetectionDevProductID = "doit-cloud-cost-anomaly-detection-development.endpoints.doit-intl-public.cloud.goog"
	flexSaveDevProductID         = "doit-flexsave-development.endpoints.doit-intl-public.cloud.goog"
	doitConsoleDevProductID      = "doit-console-development.endpoints.doit-intl-public.cloud.goog"
)

func TestMarketplaceService_ApproveEntitlement(t *testing.T) {
	type fields struct {
		loggerProvider        logger.Provider
		entitlementDAL        *dalMocks.IEntitlementFirestoreDAL
		procurementDAL        *dalMocks.ProcurementDAL
		accountDAL            *dalMocks.IAccountFirestoreDAL
		customerDAL           *customerMocks.Customers
		integrationDAL        *sharedFirestoreMocks.Integrations
		flexsaveResoldService *flexsaveResoldMocks.FlexsaveGCPServiceInterface
		userDAL               *userMocks.IUserFirestoreDAL
	}

	type args struct {
		ctx                    context.Context
		entitlementID          string
		email                  string
		doitEmployee           bool
		approveFlexsaveProduct bool
	}

	accountID := "aaaaaaaa-b7b8-b9ba-bbbc-bdbebfc0c1c2"
	billingAccountID := "AAAAAA-ABCDEF-123456"

	anomalyDevEntitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: anomalyDetectionDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateActivationRequested,
		},
	}

	flexsaveDevEntitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: flexSaveDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateActivationRequested,
		},
	}

	doitConsoleDevEntitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: doitConsoleDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateActivationRequested,
		},
	}

	anomalyDevActiveEntitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: anomalyDetectionDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateActive,
		},
	}

	anomalyDevCanceledEntitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: anomalyDetectionDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateCancelled,
		},
	}

	timeToday, err := time.Parse(time.RFC3339, "2022-01-02T15:04:05+07:00")
	if err != nil {
		t.Error(err)
	}

	customerID := "22222"

	account := &domain.AccountFirestore{
		BillingAccountType: assets.AssetGoogleCloud,
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		BillingAccountID: billingAccountID,
		User: &domain.UserDetails{
			Email: "firestoreUserEmail@doit.com",
		},
		ProcurementAccount: &domain.ProcurementAccountFirestore{
			State: domain.AccountStateActive,
			Approvals: []*domain.ApprovalFirestore{
				{
					Name:       domain.ApprovalNameSignup,
					State:      domain.ApprovalStateApproved,
					UpdateTime: &timeToday,
				},
			},
		},
	}

	accountNotApproved := &domain.AccountFirestore{
		BillingAccountType: assets.AssetGoogleCloud,
		Customer: &firestore.DocumentRef{
			ID: "22222",
		},
		BillingAccountID: billingAccountID,
		User: &domain.UserDetails{
			Email: "firestoreUserEmail@doit.com",
		},
		ProcurementAccount: &domain.ProcurementAccountFirestore{
			State: domain.AccountStateActive,
			Approvals: []*domain.ApprovalFirestore{
				{
					Name:       domain.ApprovalNameSignup,
					State:      domain.ApprovalStatePending,
					UpdateTime: &timeToday,
				},
			},
		},
	}

	standaloneAccount := &domain.AccountFirestore{
		BillingAccountType: assets.AssetStandaloneGoogleCloud,
		Customer: &firestore.DocumentRef{
			ID: "22222",
		},
		BillingAccountID: billingAccountID,
		User: &domain.UserDetails{
			Email: "firestoreUserEmail@doit.com",
		},
		ProcurementAccount: &domain.ProcurementAccountFirestore{
			State: domain.AccountStateActive,
			Approvals: []*domain.ApprovalFirestore{
				{
					Name:       domain.ApprovalNameSignup,
					State:      domain.ApprovalStateApproved,
					UpdateTime: &timeToday,
				},
			},
		},
	}

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: customerRef,
		},
		EarlyAccessFeatures: []string{"FlexSave GCP"},
	}

	user := &userDomain.User{
		ID: "userID44444",
	}

	flexsaveConfiguration := &firestorePkg.FlexsaveConfiguration{
		GCP: firestorePkg.FlexsaveSavings{
			Enabled: false,
		},
	}

	tests := []struct {
		name        string
		args        args
		fields      fields
		wantErr     bool
		on          func(*fields)
		expectedErr error
	}{
		{
			name: "approve entitlement when request comes from user and product is cost-anomaly",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "user@doit.com",
				doitEmployee:           true,
				approveFlexsaveProduct: false,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", mock.Anything, entitlementID).
					Return(&anomalyDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.procurementDAL.
					On("ApproveEntitlement", mock.Anything, entitlementID).
					Return(nil).
					Once()
			},
		},
		{
			name: "approve entitlement when request comes from user and product is doit-console",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "user@doit.com",
				doitEmployee:           true,
				approveFlexsaveProduct: false,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", mock.Anything, entitlementID).
					Return(&doitConsoleDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.procurementDAL.
					On("ApproveEntitlement", mock.Anything, entitlementID).
					Return(nil).
					Once()
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						customerID,
					).
					Return(customer, nil).
					Once()
				f.accountDAL.
					On(
						"UpdateCustomerWithDoitConsoleStatus",
						testutils.ContextBackgroundMock,
						customerRef,
						true,
					).
					Return(nil).
					Once()
			},
		},
		{
			name: "approve entitlement and do not call flexsaveAPI when user is standalone and product is flexsave",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "",
				doitEmployee:           false,
				approveFlexsaveProduct: true,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", testutils.ContextBackgroundMock, entitlementID).
					Return(&anomalyDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(standaloneAccount, nil).
					Once()
				f.procurementDAL.
					On("ApproveEntitlement", testutils.ContextBackgroundMock, entitlementID).
					Return(nil).
					Once()
			},
		},
		{
			name: "do not approve entitlement when account is not approved",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "user@doit.com",
				doitEmployee:           true,
				approveFlexsaveProduct: true,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", mock.Anything, entitlementID).
					Return(&anomalyDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(accountNotApproved, nil).
					Once()
			},
			wantErr:     true,
			expectedErr: ErrProcurementAccountIsNotApproved,
		},
		{
			name: "do not approve entitlement when it's already active",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "user@doit.com",
				doitEmployee:           true,
				approveFlexsaveProduct: true,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", mock.Anything, entitlementID).
					Return(&anomalyDevActiveEntitlement, nil).
					Once()
			},
		},
		{
			name: "return error when entitlement is not active and is not pending",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "user@doit.com",
				doitEmployee:           true,
				approveFlexsaveProduct: true,
			},
			wantErr:     true,
			expectedErr: ErrEntitlementStatusIsNotPending,
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", mock.Anything, entitlementID).
					Return(&anomalyDevCanceledEntitlement, nil).
					Once()
			},
		},
		{
			name: "approve entitlement and enable flexsave when request comes from user and product is flexsave",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "user@doit.com",
				doitEmployee:           true,
				approveFlexsaveProduct: true,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On(
						"GetEntitlement",
						testutils.ContextBackgroundMock,
						entitlementID,
					).
					Return(&flexsaveDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(customer, nil)
				f.integrationDAL.
					On(
						"GetFlexsaveConfigurationCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(flexsaveConfiguration, nil)
				f.procurementDAL.
					On(
						"ApproveEntitlement",
						testutils.ContextBackgroundMock,
						entitlementID,
					).
					Return(nil).
					Once()
				f.flexsaveResoldService.
					On(
						"EnableFlexsaveGCP",
						testutils.ContextBackgroundMock,
						"22222",
						"",
						true,
						"user@doit.com",
					).
					Return(nil)
			},
		},
		{
			name: "approve entitlement and enable flexsave when request comes from event and product is flexsave",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "",
				doitEmployee:           false,
				approveFlexsaveProduct: true,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On(
						"GetEntitlement",
						testutils.ContextBackgroundMock,
						entitlementID,
					).
					Return(&flexsaveDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(customer, nil)
				f.userDAL.
					On(
						"GetUserByEmail",
						testutils.ContextBackgroundMock,
						"firestoreUserEmail@doit.com",
						account.Customer.ID,
					).
					Return(user, nil)
				f.integrationDAL.
					On(
						"GetFlexsaveConfigurationCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(flexsaveConfiguration, nil)
				f.procurementDAL.
					On(
						"ApproveEntitlement",
						testutils.ContextBackgroundMock,
						entitlementID,
					).
					Return(nil).
					Once()
				f.flexsaveResoldService.
					On(
						"EnableFlexsaveGCP",
						testutils.ContextBackgroundMock,
						"22222",
						"userID44444",
						false,
						"firestoreUserEmail@doit.com",
					).
					Return(nil)
			},
		},
		{
			name: "approve entitlement and dot not enable flexsave" +
				"when request comes from event and product is flexsave and approveFlexsave is disabled",
			args: args{
				ctx:                    context.Background(),
				entitlementID:          entitlementID,
				email:                  "",
				doitEmployee:           false,
				approveFlexsaveProduct: false,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On(
						"GetEntitlement",
						testutils.ContextBackgroundMock,
						entitlementID,
					).
					Return(&flexsaveDevEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(customer, nil)
				f.userDAL.
					On(
						"GetUserByEmail",
						testutils.ContextBackgroundMock,
						"firestoreUserEmail@doit.com",
						account.Customer.ID,
					).
					Return(user, nil)
				f.integrationDAL.
					On(
						"GetFlexsaveConfigurationCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(flexsaveConfiguration, nil)
				f.procurementDAL.
					On(
						"ApproveEntitlement",
						testutils.ContextBackgroundMock,
						entitlementID,
					).
					Return(nil).
					Once().
					Return(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &logger.Logger{}
				},
				procurementDAL:        &dalMocks.ProcurementDAL{},
				entitlementDAL:        &dalMocks.IEntitlementFirestoreDAL{},
				accountDAL:            &dalMocks.IAccountFirestoreDAL{},
				customerDAL:           &customerMocks.Customers{},
				integrationDAL:        &sharedFirestoreMocks.Integrations{},
				flexsaveResoldService: &flexsaveResoldMocks.FlexsaveGCPServiceInterface{},
				userDAL:               &userMocks.IUserFirestoreDAL{},
			}

			s := &MarketplaceService{
				loggerProvider:        tt.fields.loggerProvider,
				procurementDAL:        tt.fields.procurementDAL,
				entitlementDAL:        tt.fields.entitlementDAL,
				accountDAL:            tt.fields.accountDAL,
				customerDAL:           tt.fields.customerDAL,
				integrationDAL:        tt.fields.integrationDAL,
				flexsaveResoldService: tt.fields.flexsaveResoldService,
				userDAL:               tt.fields.userDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.ApproveEntitlement(
				tt.args.ctx,
				tt.args.entitlementID,
				tt.args.email,
				tt.args.doitEmployee,
				tt.args.approveFlexsaveProduct,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.ApproveEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("MarketplaceService.ApproveEntitlement() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestMarketplaceService_RejectEntitlement(t *testing.T) {
	type fields struct {
		procurementDAL *dalMocks.ProcurementDAL
	}

	type args struct {
		ctx           context.Context
		entitlementID string
		email         string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "happy path",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
				email:         email,
			},
			on: func(f *fields) {
				f.procurementDAL.
					On("RejectEntitlement", mock.Anything, entitlementID, mock.AnythingOfType("string")).
					Return(nil).
					Once()
			},
		},
		{
			name: "error on RejectEntitlement",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
				email:         email,
			},
			on: func(f *fields) {
				f.procurementDAL.
					On("RejectEntitlement", mock.Anything, entitlementID, mock.AnythingOfType("string")).
					Return(errors.New("error on RejectEntitlement")).
					Once()
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
			}

			s := &MarketplaceService{procurementDAL: tt.fields.procurementDAL}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.RejectEntitlement(tt.args.ctx, tt.args.entitlementID, tt.args.email); (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.RejectEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMarketplaceService_isFlexsaveEnabled(t *testing.T) {
	type fields struct {
		integrationDAL *sharedFirestoreMocks.Integrations
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	customerID := "22222"

	flexsaveConfigurationGCPEnabled := &firestorePkg.FlexsaveConfiguration{
		GCP: firestorePkg.FlexsaveSavings{
			Enabled: true,
		},
	}

	flexsaveConfigurationGCPDisabled := &firestorePkg.FlexsaveConfiguration{
		GCP: firestorePkg.FlexsaveSavings{
			Enabled: false,
		},
	}

	tests := []struct {
		name           string
		args           args
		fields         fields
		wantErr        bool
		expectedResult bool
		on             func(*fields)
	}{
		{
			name: "returns true when flexsave GCP is already enabled",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
			},
			wantErr:        false,
			expectedResult: true,
			on: func(f *fields) {
				f.integrationDAL.
					On(
						"GetFlexsaveConfigurationCustomer",
						testutils.ContextBackgroundMock,
						customerID,
					).
					Return(flexsaveConfigurationGCPEnabled, nil)

			},
		},
		{
			name: "returns false when flexsave GCP is disabled",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
			},
			wantErr:        false,
			expectedResult: false,
			on: func(f *fields) {
				f.integrationDAL.
					On(
						"GetFlexsaveConfigurationCustomer",
						testutils.ContextBackgroundMock,
						"22222",
					).
					Return(flexsaveConfigurationGCPDisabled, nil)

			},
		},
		{
			name: "returns error when getFlexsave returns error",
			args: args{
				ctx:        context.Background(),
				customerID: customerID,
			},
			wantErr:        true,
			expectedResult: false,
			on: func(f *fields) {
				f.integrationDAL.
					On(
						"GetFlexsaveConfigurationCustomer",
						testutils.ContextBackgroundMock,
						customerID,
					).
					Return(nil, errors.New("some error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				integrationDAL: &sharedFirestoreMocks.Integrations{},
			}

			s := &MarketplaceService{
				integrationDAL: tt.fields.integrationDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res, err := s.isFlexsaveEnabled(
				tt.args.ctx,
				tt.args.customerID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.isFlexsaveEnabled() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expectedResult, res)
		})
	}
}

func TestMarketplaceService_isFlexsaveGcpDisabled(t *testing.T) {
	type args struct {
		featureFlags []string
	}

	tests := []struct {
		name           string
		args           args
		expectedResult bool
	}{
		{
			name: "returns true when fsgcp disabled flag is present",
			args: args{
				featureFlags: []string{"some other flag", "FSGCP Disabled"},
			},
			expectedResult: true,
		},
		{
			name: "returns true when fsgcp marketplace disabled flag is present",
			args: args{
				featureFlags: []string{"some other flag", "FSGCP Marketplace disabled"},
			},
			expectedResult: false,
		},
		{
			name: "returns false when fsgcp disabled flag is not present",
			args: args{
				featureFlags: []string{"some other flag", "flexsave AWS"},
			},
			expectedResult: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := isFlexsaveGcpDisabled(tt.args.featureFlags)

			assert.Equal(t, tt.expectedResult, res)
		})
	}
}

func TestMarketplaceService_CancelEntitlement(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		entitlementDAL *dalMocks.IEntitlementFirestoreDAL
		accountDAL     *dalMocks.IAccountFirestoreDAL
		customerDAL    *customerMocks.Customers
		slackService   *slackMocks.ISlackService
	}

	type args struct {
		ctx           context.Context
		entitlementID string
	}

	entitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: flexSaveDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateCancelled,
		},
	}

	doitConsoleEntitlement := domain.EntitlementFirestore{
		ProcurementEntitlement: &domain.ProcurementEntitlementFirestore{
			Product: doitConsoleDevProductID,
			Name:    fmt.Sprintf("somename/%s", entitlementID),
			Account: fmt.Sprintf("someaccountname/%s", accountID),
			State:   domain.EntitlementStateCancelled,
		},
	}

	timeToday, err := time.Parse(time.RFC3339, "2022-01-02T15:04:05+07:00")
	if err != nil {
		t.Error(err)
	}

	customerID := "22222"

	account := &domain.AccountFirestore{
		BillingAccountType: assets.AssetGoogleCloud,
		Customer: &firestore.DocumentRef{
			ID: customerID,
		},
		BillingAccountID: billingAccountID,
		User: &domain.UserDetails{
			Email: "firestoreUserEmail@doit.com",
		},
		ProcurementAccount: &domain.ProcurementAccountFirestore{
			State: domain.AccountStateActive,
			Approvals: []*domain.ApprovalFirestore{
				{
					Name:       domain.ApprovalNameSignup,
					State:      domain.ApprovalStateApproved,
					UpdateTime: &timeToday,
				},
			},
		},
	}

	primaryDomain := "www.testcustomer.com"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: customerRef,
		},
		PrimaryDomain: primaryDomain,
	}

	errorSendingMessage := errors.New("error sending slack message")

	tests := []struct {
		name        string
		args        args
		fields      fields
		wantErr     bool
		on          func(*fields)
		expectedErr error
	}{
		{
			name: "cancel entitlement when account is valid and message is sent",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", testutils.ContextBackgroundMock, entitlementID).
					Return(&entitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.customerDAL.
					On("GetCustomer", testutils.ContextBackgroundMock, customerID).
					Return(customer, nil).
					Once()
				f.slackService.
					On("PublishEntitlementCancelledMessage", testutils.ContextBackgroundMock, primaryDomain, billingAccountID).
					Return(nil).
					Once()
			},
		},
		{
			name: "cancel entitlement when product is doitConsole and when account is valid and message is sent",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
			},
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", testutils.ContextBackgroundMock, entitlementID).
					Return(&doitConsoleEntitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.customerDAL.
					On("GetCustomer", testutils.ContextBackgroundMock, customerID).
					Return(customer, nil).
					Once()
				f.accountDAL.
					On(
						"UpdateCustomerWithDoitConsoleStatus",
						testutils.ContextBackgroundMock,
						customerRef,
						false,
					).
					Return(nil).
					Once()
				f.slackService.
					On("PublishEntitlementCancelledMessage", testutils.ContextBackgroundMock, primaryDomain, billingAccountID).
					Return(nil).
					Once()
			},
		},
		{
			name: "cancel entitlement fail when entitlement does not exist",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
			},
			wantErr:     true,
			expectedErr: ErrEntitlementNotFound,
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", testutils.ContextBackgroundMock, entitlementID).
					Return(nil, ErrEntitlementNotFound).
					Once()
			},
		},
		{
			name: "cancel entitlement fails when account is valid and message was not sent",
			args: args{
				ctx:           context.Background(),
				entitlementID: entitlementID,
			},
			wantErr:     true,
			expectedErr: errorSendingMessage,
			on: func(f *fields) {
				f.entitlementDAL.
					On("GetEntitlement", testutils.ContextBackgroundMock, entitlementID).
					Return(&entitlement, nil).
					Once()
				f.accountDAL.
					On(
						"GetAccount",
						testutils.ContextBackgroundMock,
						accountID,
					).
					Return(account, nil).
					Once()
				f.customerDAL.
					On("GetCustomer", testutils.ContextBackgroundMock, customerID).
					Return(customer, nil).
					Once()
				f.slackService.
					On("PublishEntitlementCancelledMessage", testutils.ContextBackgroundMock, primaryDomain, billingAccountID).
					Return(errorSendingMessage).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &logger.Logger{}
				},
				entitlementDAL: &dalMocks.IEntitlementFirestoreDAL{},
				accountDAL:     &dalMocks.IAccountFirestoreDAL{},
				customerDAL:    &customerMocks.Customers{},
				slackService:   &slackMocks.ISlackService{},
			}

			s := &MarketplaceService{
				loggerProvider: tt.fields.loggerProvider,
				entitlementDAL: tt.fields.entitlementDAL,
				accountDAL:     tt.fields.accountDAL,
				customerDAL:    tt.fields.customerDAL,
				slackService:   tt.fields.slackService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.HandleCancelledEntitlement(
				tt.args.ctx,
				tt.args.entitlementID,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.HandleCancelledEntitlement() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("MarketplaceService.HandleCancelledEntitlement() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

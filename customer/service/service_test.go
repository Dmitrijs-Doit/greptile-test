package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/pkg"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	pkgAssets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	analyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	cloudanalyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDalMocks "github.com/doitintl/hello/scheduled-tasks/contract/dal/mocks"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	entitiesDALMocks "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing"
	invoicesDalMocks "github.com/doitintl/hello/scheduled-tasks/invoicing/dal/invoices/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	marketplaceGCPDalMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/mocks"
	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	userMocks "github.com/doitintl/hello/scheduled-tasks/user/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/user/domain"
)

func newCustomerServiceTest(t *testing.T) (*Service, *userMocks.IUserFirestoreDAL) {
	userFirestoreDALMock := new(userMocks.IUserFirestoreDAL)
	customerDALMock := new(customerMocks.Customers)
	entitiesDALMock := new(entitiesDALMocks.Entites)
	assetDALMock := new(assetDalMocks.Assets)
	contractDALMock := contractDalMocks.NewContractFirestore(t)
	invoiceDALMock := new(invoicesDalMocks.InvoicesDAL)
	marketplaceGCPDALMock := new(marketplaceGCPDalMocks.IAccountFirestoreDAL)
	conn := new(connection.Connection)
	cloudAnalytics := new(analyticsMocks.CloudAnalytics)

	service := &Service{
		logger.FromContext,
		conn,
		cloudAnalytics,
		userFirestoreDALMock,
		customerDALMock,
		entitiesDALMock,
		assetDALMock,
		contractDALMock,
		invoiceDALMock,
		marketplaceGCPDALMock,
	}

	var (
		user1 pkg.User
		user2 pkg.User
	)

	user1.ID = "123"
	user2.ID = "234"
	user1.Notifications = []int64{1, 2, 3}
	user2.Notifications = []int64{2, 3}

	userFirestoreDALMock.On("GetCustomerUsersWithNotifications", mock.Anything, mock.Anything, mock.Anything).
		Return([]*pkg.User{&user1, &user2}, nil)

	userFirestoreDALMock.On("ClearUserNotifications", mock.Anything, mock.Anything).Return(nil)
	userFirestoreDALMock.On("RestoreUserNotifications", mock.Anything, mock.Anything).Return(nil)

	return service, userFirestoreDALMock
}

func TestCustomerService_UserNotifications(t *testing.T) {
	t.Run("ClearUserNotifications", func(t *testing.T) {
		service, mock := newCustomerServiceTest(t)
		err := service.ClearCustomerUsersNotifications(context.Background(), "123")

		mock.AssertNumberOfCalls(t, "ClearUserNotifications", 2)
		assert.NoError(t, err)
	})

	t.Run("RestoreUserNotifications", func(t *testing.T) {
		service, mock := newCustomerServiceTest(t)
		err := service.RestoreCustomerUsersNotifications(context.Background(), "123")

		mock.AssertNumberOfCalls(t, "RestoreUserNotifications", 2)
		assert.NoError(t, err)
	})
}

func TestCustomerService_DeleteCustomer(t *testing.T) {
	type fields struct {
		loggerProvider    logger.Provider
		userDAL           *userMocks.IUserFirestoreDAL
		customerDAL       *customerMocks.Customers
		entitiesDALMock   *entitiesDALMocks.Entites
		assetDAL          *assetDalMocks.Assets
		contractDAL       *contractDalMocks.ContractFirestore
		invoiceDAL        *invoicesDalMocks.InvoicesDAL
		marketplaceGCPDAL *marketplaceGCPDalMocks.IAccountFirestoreDAL
	}

	type args struct {
		ctx        context.Context
		customerID string
		execute    bool
	}

	ctx := context.Background()

	itemLimit := 1

	customerID := "111"

	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	customer := common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: customerRef,
		},
	}

	var noEntities []*common.Entity

	entities := []*common.Entity{{}}

	var noContracts []common.Contract

	contracts := []common.Contract{{}}

	var noAssets []*pkgAssets.BaseAsset

	assets := []*pkgAssets.BaseAsset{{}}

	var noInvoices []*invoicing.Invoice

	invoices := []*invoicing.Invoice{{}}

	var noUsers []*domain.User

	users := []*domain.User{{}}

	gcpMarketplaceAccount := &gcpTableMgmtDomain.AccountFirestore{}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "no errors and no deletion, when customer doesn't have any items and execute=false",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(noContracts, nil).
					Once()
				f.assetDAL.On("ListBaseAssetsForCustomer", ctx, customerRef, itemLimit).
					Return(noAssets, nil).
					Once()
				f.invoiceDAL.On("ListInvoices", ctx, customerRef, itemLimit).
					Return(noInvoices, nil).
					Once()
				f.userDAL.On("ListUsers", ctx, customerRef, itemLimit).
					Return(noUsers, nil).
					Once()
				f.marketplaceGCPDAL.On("GetAccountByCustomer", ctx, customerID).
					Return(nil, nil).
					Once()
			},
		},
		{
			name: "no errors and deletion, when customer doesn't have any items and execute=true",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    true,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(noContracts, nil).
					Once()
				f.assetDAL.On("ListBaseAssetsForCustomer", ctx, customerRef, itemLimit).
					Return(noAssets, nil).
					Once()
				f.invoiceDAL.On("ListInvoices", ctx, customerRef, itemLimit).
					Return(noInvoices, nil).
					Once()
				f.userDAL.On("ListUsers", ctx, customerRef, itemLimit).
					Return(noUsers, nil).
					Once()
				f.marketplaceGCPDAL.On("GetAccountByCustomer", ctx, customerID).
					Return(nil, nil).
					Once()
				f.customerDAL.On("DeleteCustomer", ctx, customerID).
					Return(nil).
					Once()
			},
		},
		{
			name: "error and no deletion, when customer has gcp marketplace account",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr:     true,
			expectedErr: ErrCustomerHasGCPMarketplaceAccounts,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(noContracts, nil).
					Once()
				f.assetDAL.On("ListBaseAssetsForCustomer", ctx, customerRef, itemLimit).
					Return(noAssets, nil).
					Once()
				f.invoiceDAL.On("ListInvoices", ctx, customerRef, itemLimit).
					Return(noInvoices, nil).
					Once()
				f.userDAL.On("ListUsers", ctx, customerRef, itemLimit).
					Return(noUsers, nil).
					Once()
				f.marketplaceGCPDAL.On("GetAccountByCustomer", ctx, customerID).
					Return(gcpMarketplaceAccount, nil).
					Once()
			},
		},
		{
			name: "error and no deletion, when customer has users",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr:     true,
			expectedErr: ErrCustomerHasUsers,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(noContracts, nil).
					Once()
				f.assetDAL.On("ListBaseAssetsForCustomer", ctx, customerRef, itemLimit).
					Return(noAssets, nil).
					Once()
				f.invoiceDAL.On("ListInvoices", ctx, customerRef, itemLimit).
					Return(noInvoices, nil).
					Once()
				f.userDAL.On("ListUsers", ctx, customerRef, itemLimit).
					Return(users, nil).
					Once()
			},
		},
		{
			name: "error and no deletion, when customer has invoices",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr:     true,
			expectedErr: ErrCustomerHasInvoices,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(noContracts, nil).
					Once()
				f.assetDAL.On("ListBaseAssetsForCustomer", ctx, customerRef, itemLimit).
					Return(noAssets, nil).
					Once()
				f.invoiceDAL.On("ListInvoices", ctx, customerRef, itemLimit).
					Return(invoices, nil).
					Once()
			},
		},
		{
			name: "error and no deletion, when customer has invoices",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr:     true,
			expectedErr: ErrCustomerHasAssets,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(noContracts, nil).
					Once()
				f.assetDAL.On("ListBaseAssetsForCustomer", ctx, customerRef, itemLimit).
					Return(assets, nil).
					Once()
			},
		},
		{
			name: "error and no deletion, when customer has contracts",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr:     true,
			expectedErr: ErrCustomerHasContracts,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(noEntities, nil).
					Once()
				f.contractDAL.On("ListContracts", ctx, customerRef, itemLimit).
					Return(contracts, nil).
					Once()
			},
		},
		{
			name: "error and no deletion, when customer has entities",
			args: args{
				ctx:        ctx,
				customerID: customerID,
				execute:    false,
			},
			wantErr:     true,
			expectedErr: ErrCustomerHasBillingProfiles,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer", ctx, customerID).
					Return(&customer, nil).
					Once()
				f.entitiesDALMock.On("GetCustomerEntities", ctx, customerRef).
					Return(entities, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			tt.fields = fields{
				loggerProvider:    logger.FromContext,
				userDAL:           &userMocks.IUserFirestoreDAL{},
				customerDAL:       &customerMocks.Customers{},
				entitiesDALMock:   &entitiesDALMocks.Entites{},
				assetDAL:          &assetDalMocks.Assets{},
				contractDAL:       contractDalMocks.NewContractFirestore(t),
				invoiceDAL:        &invoicesDalMocks.InvoicesDAL{},
				marketplaceGCPDAL: &marketplaceGCPDalMocks.IAccountFirestoreDAL{},
			}

			customerService := Service{
				loggerProvider:    tt.fields.loggerProvider,
				userDAL:           tt.fields.userDAL,
				customerDAL:       tt.fields.customerDAL,
				entitiesDAL:       tt.fields.entitiesDALMock,
				assetDAL:          tt.fields.assetDAL,
				contractDAL:       tt.fields.contractDAL,
				invoiceDAL:        tt.fields.invoiceDAL,
				marketplaceGCPDAL: tt.fields.marketplaceGCPDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := customerService.Delete(
				ctx,
				tt.args.customerID,
				tt.args.execute,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("CustomerService.Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("CustomerService.Delete() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestCustomerService_getSegmentQueryRequestCols(t *testing.T) {
	got := getSegmentQueryRequestCols()
	if got[0].Field != "T.usage_date_time" {
		t.Errorf("getSegmentQueryRequestCols() = %s; want T.usage_date_time", got[0].Field)
	}

	if got[0].Position != "col" {
		t.Errorf("getSegmentQueryRequestCols() = %s; want col", got[0].Position)
	}
}

func TestCustomerService_getSegmentQueryRequest(t *testing.T) {
	type fields struct {
		cloudAnalytics *cloudanalyticsMocks.CloudAnalytics
	}

	type args struct {
		today time.Time
	}

	customerID := "test-testCustomer-id"
	tests := []struct {
		name             string
		wantTimeSettings cloudanalytics.QueryRequestTimeSettings
		wantErr          bool
		fields           fields
		on               func(*fields)
		args             args
	}{
		{
			name:    "test when fromDate is end of month",
			wantErr: false,
			args: args{
				today: time.Date(2022, 01, 31, 0, 0, 0, 0, time.UTC),
			},
			wantTimeSettings: cloudanalytics.QueryRequestTimeSettings{
				Interval: "month",
				From:     &[]time.Time{time.Date(2022, 01, 31, 0, 0, 0, 0, time.UTC)}[0],
				To:       &[]time.Time{time.Date(2021, 11, 02, 0, 0, 0, 0, time.UTC)}[0],
			},
			on: func(f *fields) {
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
			},
		},
		{
			name:    "test when fromDate is beginning of month",
			wantErr: false,
			args: args{
				today: time.Date(2022, 01, 01, 0, 0, 0, 0, time.UTC),
			},
			wantTimeSettings: cloudanalytics.QueryRequestTimeSettings{
				Interval: "month",
				From:     &[]time.Time{time.Date(2022, 01, 01, 0, 0, 0, 0, time.UTC)}[0],
				To:       &[]time.Time{time.Date(2021, 10, 03, 0, 0, 0, 0, time.UTC)}[0],
			},
			on: func(f *fields) {
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
			},
		},
		{
			name:    "test when fromDate is middle of month",
			wantErr: false,
			args: args{
				today: time.Date(2022, 06, 17, 0, 0, 0, 0, time.UTC),
			},
			wantTimeSettings: cloudanalytics.QueryRequestTimeSettings{
				Interval: "month",
				From:     &[]time.Time{time.Date(2022, 06, 17, 0, 0, 0, 0, time.UTC)}[0],
				To:       &[]time.Time{time.Date(2022, 03, 19, 0, 0, 0, 0, time.UTC)}[0],
			},
			on: func(f *fields) {
				f.cloudAnalytics.On("GetAccounts", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]string{"account1"}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				cloudAnalytics: &cloudanalyticsMocks.CloudAnalytics{},
			}
			ctx := context.Background()

			customerService := &Service{
				cloudAnalytics: tt.fields.cloudAnalytics,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := customerService.getSegmentQueryRequest(ctx, customerID, tt.args.today)
			if (err != nil) != tt.wantErr {
				t.Errorf("AnalyticsAlertsService.getReportRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				assert.EqualValues(t, tt.wantTimeSettings, *got.TimeSettings)
			}
		})
	}
}

func TestCustomerService_getSegmentValue(t *testing.T) {
	t.Run("getSegmentValue below 10,000", func(t *testing.T) {
		segmentVal := getSegmentValue(99)
		assert.Equal(t, segmentVal, common.Invest)
	})

	t.Run("getSegmentValue between 10,000 and 100,000", func(t *testing.T) {
		segmentVal := getSegmentValue(15000)
		assert.Equal(t, segmentVal, common.Incubate)
	})

	t.Run("getSegmentValue above 100,000", func(t *testing.T) {
		segmentVal := getSegmentValue(100001)
		assert.Equal(t, segmentVal, common.Accelerate)
	})
}

func TestCustomerService_getCustomerSegmentValue(t *testing.T) {
	t.Run("getCustomerSegmentValue with value", func(t *testing.T) {
		bqVal := [][]bigquery.Value{
			{"a", "b", 2.4},
			{"a", "b", 1.0},
			{"a", "b", 5.0},
		}

		segmentVal, err := getCustomerSegmentValue(bqVal, 2)
		if err != nil {
			t.Errorf("getCustomerSegmentValue error = %v, wantErr %v", err, false)
			return
		}

		assert.True(t, math.Abs(segmentVal-2.8) < 0.01)
	})

	t.Run("getCustomerSegmentValue with error", func(t *testing.T) {
		bqVal := [][]bigquery.Value{
			{"a", "b", "1"},
			{"a", "b", 1.0},
			{"a", "b", 5.0},
		}

		segmentVal, err := getCustomerSegmentValue(bqVal, 2)
		if err == nil {
			t.Errorf("getCustomerSegmentValue error = %v, wantErr %v", err, true)
		}

		assert.Equal(t, segmentVal, 0.0)
	})
}

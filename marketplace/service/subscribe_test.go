package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/cloudcommerceprocurement/v1"

	fsDalMocks "github.com/doitintl/firestore/mocks"
	assetsMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestMarketplaceService_Subscribe(t *testing.T) {
	type fields struct {
		loggerProvider  logger.Provider
		accountDAL      *dalMocks.IAccountFirestoreDAL
		assetDAL        *assetsMocks.Assets
		customerDAL     *customerMocks.Customers
		procurementDAL  *dalMocks.ProcurementDAL
		customerTypeDal *fsDalMocks.CustomerTypeIface
	}

	type args struct {
		ctx              context.Context
		subscribePayload domain.SubscribePayload
	}

	const (
		resoldCustomerID     = "resoldCustomerID"
		standaloneCustomerID = "standaloneCustomerID"
		hybridCustomerID     = "hybridCustomerID"
		procurementAccountID = "procurementAccountID"
		gcpBillingAccountID  = "gcpBillingAccountID"
	)

	resoldCustomerRef := &firestore.DocumentRef{
		ID: resoldCustomerID,
	}

	standaloneCustomerRef := &firestore.DocumentRef{
		ID: standaloneCustomerID,
	}

	hybridCustomerRef := &firestore.DocumentRef{
		ID: hybridCustomerID,
	}

	resoldCustomer := &common.Customer{
		ID: resoldCustomerID,
		Snapshot: &firestore.DocumentSnapshot{
			Ref: resoldCustomerRef,
		},
	}

	standaloneCustomer := &common.Customer{
		ID: standaloneCustomerID,
		Snapshot: &firestore.DocumentSnapshot{
			Ref: standaloneCustomerRef,
		},
	}

	hybridCustomer := &common.Customer{
		ID: hybridCustomerID,
		Snapshot: &firestore.DocumentSnapshot{
			Ref: hybridCustomerRef,
		},
		EarlyAccessFeatures: []string{"Flexsave GCP Standalone"},
	}

	resoldProductFlexsaveSubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           resoldCustomerID,
		Email:                "resold@test.com",
		UID:                  "11111",
		Product:              "doit-flexsave-development",
		UserID:               "someFirestoreUserId44444",
	}

	standaloneProductFlexsaveSubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           standaloneCustomerID,
		Email:                "standalone@test.com",
		UID:                  "22222",
		Product:              "doit-flexsave-development",
		UserID:               "someFirestoreUserId44444",
	}

	resoldProductAnomalySubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           resoldCustomerID,
		Email:                "resold@test.com",
		UID:                  "11111",
		Product:              "doit-cloud-cost-anomaly-detection-development",
		UserID:               "someFirestoreUserId44444",
	}

	resoldProductConsoleSubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           resoldCustomerID,
		Email:                "resold@test.com",
		UID:                  "11111",
		Product:              "doit-console-development",
		UserID:               "someFirestoreUserId44444",
	}

	standaloneProductAnomalySubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           standaloneCustomerID,
		Email:                "standalone@test.com",
		UID:                  "22222",
		Product:              "doit-cloud-cost-anomaly-detection-development",
		UserID:               "someFirestoreUserId44444",
	}

	standaloneProductConsoleSubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           standaloneCustomerID,
		Email:                "standalone@test.com",
		UID:                  "22222",
		Product:              "doit-console-development",
		UserID:               "someFirestoreUserId44444",
	}

	hybridSubscribePayload := domain.SubscribePayload{
		ProcurementAccountID: procurementAccountID,
		CustomerID:           hybridCustomerID,
		Email:                "hybrid@test.com",
		UID:                  "33333",
		Product:              "doit-cloud-cost-anomaly-detection-development",
		UserID:               "someFirestoreUserId44444",
	}

	resoldProcAccount := &domain.AccountFirestore{
		Customer: &firestore.DocumentRef{
			ID: resoldCustomerID,
		},
	}

	standaloneProcAccount := &domain.AccountFirestore{
		Customer: &firestore.DocumentRef{
			ID: standaloneCustomerID,
		},
	}

	gcpAssetsFilters := []string{assetsPkg.AssetGoogleCloud, assetsPkg.AssetStandaloneGoogleCloud}

	gcpAssets := []*assetsPkg.GCPAsset{
		{
			BaseAsset: assetsPkg.BaseAsset{
				AssetType: assetsPkg.AssetStandaloneGoogleCloud,
			},
			Properties: &assetsPkg.GCPProperties{
				BillingAccountID: gcpBillingAccountID,
			},
		},
	}

	gcpAssetsResold := []*assetsPkg.GCPAsset{
		{
			BaseAsset: assetsPkg.BaseAsset{
				AssetType: assetsPkg.AssetGoogleCloud,
			},
			Properties: &assetsPkg.GCPProperties{
				BillingAccountID: gcpBillingAccountID,
			},
		},
	}

	tests := []struct {
		name        string
		args        args
		fields      fields
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "reject subscribe resold customer when product is flexsave",
			args: args{
				ctx:              context.Background(),
				subscribePayload: resoldProductFlexsaveSubscribePayload,
			},
			wantErr:     true,
			expectedErr: ErrFlexsaveProductIsDisabled,
		},
		{
			name: "reject subscribe standalone customer when product is flexsave",
			args: args{
				ctx:              context.Background(),
				subscribePayload: standaloneProductFlexsaveSubscribePayload,
			},
			wantErr:     true,
			expectedErr: ErrFlexsaveProductIsDisabled,
		},
		{
			name: "successfully subscribe resold customer when product is anomaly",
			args: args{
				ctx:              context.Background(),
				subscribePayload: resoldProductAnomalySubscribePayload,
			},
			on: func(f *fields) {
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						resoldCustomerID,
					).
					Return(resoldCustomer, nil)
				f.accountDAL.
					On(
						"UpdateAccountWithCustomerDetails",
						testutils.ContextBackgroundMock,
						resoldCustomerRef,
						resoldProductAnomalySubscribePayload,
					).
					Return(true, nil).
					Once()
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, procurementAccountID).
					Return(resoldProcAccount, nil).
					Once()
				f.customerDAL.
					On(
						"GetRef",
						testutils.ContextBackgroundMock,
						resoldCustomerID,
					).
					Return(resoldCustomerRef)
				f.assetDAL.
					On("GetCustomerGCPAssetsWithTypes",
						testutils.ContextBackgroundMock,
						resoldCustomerRef,
						gcpAssetsFilters,
					).
					Return(gcpAssets, nil).
					Once()
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: gcpBillingAccountID,
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: procurementAccountID,
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()
				f.accountDAL.
					On(
						"UpdateGcpBillingAccountDetails",
						testutils.ContextBackgroundMock,
						procurementAccountID,
						dal.BillingAccountDetails{
							BillingAccountID:   gcpBillingAccountID,
							BillingAccountType: assetsPkg.AssetStandaloneGoogleCloud,
						},
					).
					Return(nil).
					Once()
				f.procurementDAL.
					On("PublishAccountApprovalRequestEvent",
						testutils.ContextBackgroundMock,
						resoldProductAnomalySubscribePayload,
					).
					Return(nil).
					Once()
				f.customerTypeDal.On(
					"IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock,
					resoldCustomerID).
					Return(true, nil)
			},
		},
		{
			name: "successfully subscribe resold customer and publish account activation when product is console",
			args: args{
				ctx:              context.Background(),
				subscribePayload: resoldProductConsoleSubscribePayload,
			},
			on: func(f *fields) {
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						resoldCustomerID,
					).
					Return(resoldCustomer, nil)
				f.accountDAL.
					On(
						"UpdateAccountWithCustomerDetails",
						testutils.ContextBackgroundMock,
						resoldCustomerRef,
						resoldProductConsoleSubscribePayload,
					).
					Return(true, nil).
					Once()
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, procurementAccountID).
					Return(resoldProcAccount, nil).
					Once()
				f.customerDAL.
					On(
						"GetRef",
						testutils.ContextBackgroundMock,
						resoldCustomerID,
					).
					Return(resoldCustomerRef)
				f.assetDAL.
					On("GetCustomerGCPAssetsWithTypes",
						testutils.ContextBackgroundMock,
						resoldCustomerRef,
						gcpAssetsFilters,
					).
					Return(gcpAssetsResold, nil).
					Once()
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: gcpBillingAccountID,
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: procurementAccountID,
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()
				f.accountDAL.
					On(
						"UpdateGcpBillingAccountDetails",
						testutils.ContextBackgroundMock,
						procurementAccountID,
						dal.BillingAccountDetails{
							BillingAccountID:   gcpBillingAccountID,
							BillingAccountType: assetsPkg.AssetGoogleCloud,
						},
					).
					Return(nil).
					Once()
				f.procurementDAL.
					On("PublishAccountApprovalRequestEvent",
						testutils.ContextBackgroundMock,
						resoldProductConsoleSubscribePayload,
					).
					Return(nil).
					Once()
				f.customerTypeDal.On(
					"IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock,
					resoldCustomerID).
					Return(true, nil)
			},
		},
		{
			name: "successfully subscribe standalone customer and publish account activation when product is console",
			args: args{
				ctx:              context.Background(),
				subscribePayload: standaloneProductConsoleSubscribePayload,
			},
			on: func(f *fields) {
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						standaloneCustomerID,
					).
					Return(standaloneCustomer, nil)
				f.accountDAL.
					On(
						"UpdateAccountWithCustomerDetails",
						testutils.ContextBackgroundMock,
						standaloneCustomerRef,
						standaloneProductConsoleSubscribePayload,
					).
					Return(true, nil).
					Once()
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, procurementAccountID).
					Return(standaloneProcAccount, nil).
					Once()
				f.customerDAL.
					On(
						"GetRef",
						testutils.ContextBackgroundMock,
						standaloneCustomerID,
					).
					Return(standaloneCustomerRef)
				f.assetDAL.
					On("GetCustomerGCPAssetsWithTypes",
						testutils.ContextBackgroundMock,
						standaloneCustomerRef,
						gcpAssetsFilters,
					).
					Return(gcpAssets, nil).
					Once()
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: gcpBillingAccountID,
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: procurementAccountID,
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()
				f.accountDAL.
					On(
						"UpdateGcpBillingAccountDetails",
						testutils.ContextBackgroundMock,
						procurementAccountID,
						dal.BillingAccountDetails{
							BillingAccountID:   gcpBillingAccountID,
							BillingAccountType: assetsPkg.AssetStandaloneGoogleCloud,
						},
					).
					Return(nil).
					Once()
				f.procurementDAL.
					On("PublishAccountApprovalRequestEvent",
						testutils.ContextBackgroundMock,
						standaloneProductConsoleSubscribePayload,
					).
					Return(nil).
					Once()
				f.customerTypeDal.On(
					"IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock,
					standaloneCustomerID).
					Return(false, nil)
			},
		},
		{
			name: "error on subscribing standalone customer when product is anomaly",
			args: args{
				ctx:              context.Background(),
				subscribePayload: standaloneProductAnomalySubscribePayload,
			},
			wantErr:     true,
			expectedErr: ErrCustomerIsNotEligibleCostAnomaly,
			on: func(f *fields) {
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						standaloneCustomerID,
					).
					Return(standaloneCustomer, nil)
				f.accountDAL.
					On(
						"UpdateAccountWithCustomerDetails",
						testutils.ContextBackgroundMock,
						standaloneCustomerRef,
						standaloneProductAnomalySubscribePayload,
					).
					Return(true, nil).
					Once()
				f.customerTypeDal.On(
					"IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock,
					standaloneCustomerID).
					Return(false, nil)
			},
		},
		{
			name: "do not subscribe hybrid customer to costAnomaly",
			args: args{
				ctx:              context.Background(),
				subscribePayload: hybridSubscribePayload,
			},
			wantErr:     true,
			expectedErr: ErrCustomerIsNotEligibleCostAnomaly,
			on: func(f *fields) {
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						hybridCustomerID,
					).
					Return(hybridCustomer, nil)
				f.accountDAL.
					On(
						"UpdateAccountWithCustomerDetails",
						testutils.ContextBackgroundMock,
						hybridCustomerRef,
						hybridSubscribePayload,
					).
					Return(true, nil).
					Once()
				f.customerTypeDal.On(
					"IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock,
					hybridCustomerID).
					Return(false, nil)
			},
		},
		{
			name: "do not subscribe if customer is already subscribed when product is anomaly",
			args: args{
				ctx:              context.Background(),
				subscribePayload: resoldProductAnomalySubscribePayload,
			},
			wantErr: true,
			on: func(f *fields) {
				f.customerDAL.
					On(
						"GetCustomer",
						testutils.ContextBackgroundMock,
						resoldCustomerID,
					).
					Return(resoldCustomer, nil)
				f.accountDAL.
					On(
						"UpdateAccountWithCustomerDetails",
						testutils.ContextBackgroundMock,
						resoldCustomerRef,
						resoldProductAnomalySubscribePayload,
					).
					Return(false, nil).
					Once()
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, procurementAccountID).
					Return(resoldProcAccount, nil).
					Once()
				f.customerDAL.
					On(
						"GetRef",
						testutils.ContextBackgroundMock,
						resoldCustomerID,
					).
					Return(resoldCustomerRef)
				f.assetDAL.
					On("GetCustomerGCPAssetsWithTypes",
						testutils.ContextBackgroundMock,
						resoldCustomerRef,
						gcpAssetsFilters,
					).
					Return(gcpAssets, nil).
					Once()
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: gcpBillingAccountID,
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: procurementAccountID,
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()
				f.accountDAL.
					On("UpdateGcpBillingAccountDetails",
						testutils.ContextBackgroundMock,
						procurementAccountID,
						dal.BillingAccountDetails{
							BillingAccountID:   gcpBillingAccountID,
							BillingAccountType: assetsPkg.AssetStandaloneGoogleCloud,
						},
					).
					Return(nil).
					Once()
				f.customerTypeDal.On(
					"IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock,
					resoldCustomerID).
					Return(true, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &logger.Logger{}
				},
				customerDAL:     &customerMocks.Customers{},
				accountDAL:      &dalMocks.IAccountFirestoreDAL{},
				assetDAL:        &assetsMocks.Assets{},
				customerTypeDal: &fsDalMocks.CustomerTypeIface{},
			}

			s := &MarketplaceService{
				loggerProvider:  tt.fields.loggerProvider,
				accountDAL:      tt.fields.accountDAL,
				assetDAL:        tt.fields.assetDAL,
				customerDAL:     tt.fields.customerDAL,
				procurementDAL:  tt.fields.procurementDAL,
				customerTypeDal: tt.fields.customerTypeDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			err := s.Subscribe(
				tt.args.ctx,
				tt.args.subscribePayload,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.Subscribe() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("MarketplaceService.Subscribe() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

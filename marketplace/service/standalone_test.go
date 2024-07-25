package service

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/cloudcommerceprocurement/v1"

	fsDalMocks "github.com/doitintl/firestore/mocks"
	assetsMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestMarketplaceService_StandaloneApprove(t *testing.T) {
	type fields struct {
		loggerProvider  logger.Provider
		accountDAL      *dalMocks.IAccountFirestoreDAL
		assetDAL        *assetsMocks.Assets
		customerDAL     *customerMocks.Customers
		procurementDAL  *dalMocks.ProcurementDAL
		customerTypeDal *fsDalMocks.CustomerTypeIface
	}

	const (
		resoldCustomerID     = "resoldCustomerID"
		standaloneCustomerID = "standaloneCustomerID"
		hybridCustomerID     = "hybridCustomerID"
		procurementAccountID = "procurementAccountID"
		gcpBillingAccountID  = "gcpBillingAccountID"
	)

	var (
		procurementAccountName = fmt.Sprintf("providers/doit-intl-public/accounts/%s", procurementAccountID)
		assetTypes             = []string{assetsPkg.AssetGoogleCloud, assetsPkg.AssetStandaloneGoogleCloud}
		billingAccountDetails  = dal.BillingAccountDetails{
			BillingAccountID:   gcpBillingAccountID,
			BillingAccountType: assetsPkg.AssetStandaloneGoogleCloud,
		}
	)

	gcpAssets := []*pkg.GCPAsset{
		{
			BaseAsset: assetsPkg.BaseAsset{
				AssetType: assetsPkg.AssetStandaloneGoogleCloud,
			},
			Properties: &pkg.GCPProperties{
				BillingAccountID: gcpBillingAccountID,
			},
		},
	}

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
		EarlyAccessFeatures: []string{},
	}

	standaloneCustomer := &common.Customer{
		ID: standaloneCustomerID,
		Snapshot: &firestore.DocumentSnapshot{
			Ref: standaloneCustomerRef,
		},
		EarlyAccessFeatures: []string{},
	}

	hybridCustomer := &common.Customer{
		ID: hybridCustomerID,
		Snapshot: &firestore.DocumentSnapshot{
			Ref: hybridCustomerRef,
		},
		EarlyAccessFeatures: []string{"Flexsave GCP Standalone"},
	}

	standaloneProcAccount := &domain.AccountFirestore{
		ProcurementAccount: &domain.ProcurementAccountFirestore{Name: procurementAccountName},
		Customer: &firestore.DocumentRef{
			ID: standaloneCustomerID,
		},
	}

	hybridProcAccount := &domain.AccountFirestore{
		ProcurementAccount: &domain.ProcurementAccountFirestore{Name: procurementAccountName},
		Customer: &firestore.DocumentRef{
			ID: hybridCustomerID,
		},
	}

	filterBillingAccount := dal.Filter{
		Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
		Value: gcpBillingAccountID,
	}
	filterProcAccount := dal.Filter{
		Key:   dal.EntitlementFilterKeyAccount,
		Value: procurementAccountID,
	}

	type args struct {
		ctx              context.Context
		customerID       string
		billingAccountID string
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully approve a standalone customer procurement account",
			args: args{
				ctx:              context.Background(),
				customerID:       standaloneCustomerID,
				billingAccountID: gcpBillingAccountID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneCustomer, nil).Once()
				f.accountDAL.On("GetAccountByCustomer",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneProcAccount, nil).Once()
				f.accountDAL.On("GetAccount",
					testutils.ContextBackgroundMock, procurementAccountID,
				).Return(standaloneProcAccount, nil).Once()
				f.customerDAL.On("GetRef",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneCustomerRef)
				f.assetDAL.On("GetCustomerGCPAssetsWithTypes",
					testutils.ContextBackgroundMock, standaloneCustomerRef, assetTypes,
				).Return(gcpAssets, nil).Once()
				f.procurementDAL.On("ListEntitlements",
					testutils.ContextBackgroundMock, filterBillingAccount, filterProcAccount,
				).Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).Once()
				f.accountDAL.On("UpdateGcpBillingAccountDetails",
					testutils.ContextBackgroundMock, procurementAccountID, billingAccountDetails,
				).Return(nil).Once()
				f.procurementDAL.On("ApproveAccount",
					testutils.ContextBackgroundMock, procurementAccountID, "Approved by Flexsave standalone",
				).Return(nil).Once()
				f.customerTypeDal.On("IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(false, nil).Once()
			},
		},
		{
			name: "successfully approve a hybrid customer procurement account",
			args: args{
				ctx:              context.Background(),
				customerID:       hybridCustomerID,
				billingAccountID: gcpBillingAccountID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer",
					testutils.ContextBackgroundMock, hybridCustomerID,
				).Return(hybridCustomer, nil).Once()
				f.accountDAL.On("GetAccountByCustomer",
					testutils.ContextBackgroundMock, hybridCustomerID,
				).Return(hybridProcAccount, nil).Once()
				f.accountDAL.On("GetAccount",
					testutils.ContextBackgroundMock, procurementAccountID,
				).Return(hybridProcAccount, nil).Once()
				f.customerDAL.On("GetRef",
					testutils.ContextBackgroundMock, hybridCustomerID,
				).Return(hybridCustomerRef)
				f.assetDAL.On("GetCustomerGCPAssetsWithTypes",
					testutils.ContextBackgroundMock, hybridCustomerRef, assetTypes,
				).Return(gcpAssets, nil).Once()
				f.procurementDAL.On("ListEntitlements",
					testutils.ContextBackgroundMock, filterBillingAccount, filterProcAccount,
				).Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).Once()
				f.accountDAL.On("UpdateGcpBillingAccountDetails",
					testutils.ContextBackgroundMock, procurementAccountID, billingAccountDetails,
				).Return(nil).Once()
				f.procurementDAL.On("ApproveAccount",
					testutils.ContextBackgroundMock, procurementAccountID, "Approved by Flexsave standalone",
				).Return(nil).Once()
				f.customerTypeDal.On("IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock, hybridCustomerID,
				).Return(false, nil).Once()
			},
		},
		{
			name: "fail on non-standalone customer type",
			args: args{
				ctx:              context.Background(),
				customerID:       resoldCustomerID,
				billingAccountID: gcpBillingAccountID,
			},
			wantErr:     true,
			expectedErr: ErrCustomerNotStandalone,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer",
					testutils.ContextBackgroundMock, resoldCustomerID,
				).Return(resoldCustomer, nil).Once()
				f.customerTypeDal.On("IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock, resoldCustomerID,
				).Return(true, nil).Once()
			},
		},
		{
			name: "fail on mismatch gcp billing account id",
			args: args{
				ctx:              context.Background(),
				customerID:       standaloneCustomerID,
				billingAccountID: "otherGcpBillingAccountID",
			},
			wantErr:     true,
			expectedErr: ErrBillingAccountMismatch,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneCustomer, nil).Once()
				f.accountDAL.On("GetAccountByCustomer",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneProcAccount, nil).Once()
				f.accountDAL.On("GetAccount",
					testutils.ContextBackgroundMock, procurementAccountID,
				).Return(standaloneProcAccount, nil).Once()
				f.customerDAL.On("GetRef",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneCustomerRef)
				f.assetDAL.On("GetCustomerGCPAssetsWithTypes",
					testutils.ContextBackgroundMock, standaloneCustomerRef, assetTypes,
				).Return(gcpAssets, nil).Once()
				f.procurementDAL.On("ListEntitlements",
					testutils.ContextBackgroundMock, filterBillingAccount, filterProcAccount,
				).Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).Once()
				f.accountDAL.On("UpdateGcpBillingAccountDetails",
					testutils.ContextBackgroundMock, procurementAccountID, billingAccountDetails,
				).Return(nil).Once()
				f.customerTypeDal.On("IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(false, nil).Once()
			},
		},
		{
			name: "fail on populate billing account id",
			args: args{
				ctx:              context.Background(),
				customerID:       standaloneCustomerID,
				billingAccountID: "otherGcpBillingAccountID",
			},
			wantErr:     true,
			expectedErr: ErrBillingAccountNotFound,
			on: func(f *fields) {
				f.customerDAL.On("GetCustomer",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneCustomer, nil).Once()
				f.accountDAL.On("GetAccountByCustomer",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneProcAccount, nil).Once()
				f.accountDAL.On("GetAccount",
					testutils.ContextBackgroundMock, procurementAccountID,
				).Return(standaloneProcAccount, nil).Once()
				f.customerDAL.On("GetRef",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(standaloneCustomerRef)
				f.assetDAL.On("GetCustomerGCPAssetsWithTypes",
					testutils.ContextBackgroundMock, standaloneCustomerRef, assetTypes,
				).Return(gcpAssets, nil).Once()
				f.procurementDAL.On("ListEntitlements",
					testutils.ContextBackgroundMock, filterBillingAccount, filterProcAccount,
				).Return([]*cloudcommerceprocurement.Entitlement{}, nil).Once()
				f.customerTypeDal.On("IsProcurementOnlyCustomerType",
					testutils.ContextBackgroundMock, standaloneCustomerID,
				).Return(false, nil).Once()
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

			err := s.StandaloneApprove(tt.args.ctx, tt.args.customerID, tt.args.billingAccountID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.StandaloneApprove() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.expectedErr != nil && err != tt.expectedErr {
				t.Errorf("MarketplaceService.StandaloneApprove() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

package service

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/cloudcommerceprocurement/v1"

	assetsMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/dal"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/marketplace/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestMarketplaceService_PopulateBillingAccounts(t *testing.T) {
	type fields struct {
		accountDAL     *dalMocks.IAccountFirestoreDAL
		customerDAL    *customerMocks.Customers
		assetDAL       *assetsMocks.Assets
		procurementDAL *dalMocks.ProcurementDAL
	}

	type args struct {
		ctx                     context.Context
		populateBillingAccounts domain.PopulateBillingAccounts
	}

	populateBillingAccounts := domain.PopulateBillingAccounts{
		{
			ProcurementAccountID: "11111",
		},
		{
			ProcurementAccountID: "22222",
		},
	}

	account1 := &domain.AccountFirestore{
		Customer: &firestore.DocumentRef{
			ID: "someAccountID1",
		},
	}

	account2 := &domain.AccountFirestore{
		Customer: &firestore.DocumentRef{
			ID: "someAccountID2",
		},
	}

	customerRef1 := &firestore.DocumentRef{}
	customerRef2 := &firestore.DocumentRef{}

	gcpAssets1 := []*assetsPkg.GCPAsset{
		{
			BaseAsset: assetsPkg.BaseAsset{
				AssetType: assetsPkg.AssetGoogleCloud,
			},
			Properties: &assetsPkg.GCPProperties{
				BillingAccountID: "billingAccount1",
			},
		},
	}

	gcpAssets2 := []*assetsPkg.GCPAsset{
		{
			BaseAsset: assetsPkg.BaseAsset{
				AssetType: assetsPkg.AssetGoogleCloud,
			},
			Properties: &assetsPkg.GCPProperties{
				BillingAccountID: "someOtherBillingAccount3",
			},
		},
		{
			BaseAsset: assetsPkg.BaseAsset{
				AssetType: assetsPkg.AssetStandaloneGoogleCloud,
			},
			Properties: &assetsPkg.GCPProperties{
				BillingAccountID: "billingAccount2",
			},
		},
	}

	gcpAssetsFilters := []string{assetsPkg.AssetGoogleCloud, assetsPkg.AssetStandaloneGoogleCloud}

	tests := []struct {
		name           string
		args           args
		fields         fields
		expectedResult domain.PopulateBillingAccountsResult
		wantErr        bool
		on             func(*fields)
	}{
		{
			name: "populate one account and one fail",
			args: args{
				ctx:                     context.Background(),
				populateBillingAccounts: populateBillingAccounts,
			},
			expectedResult: domain.PopulateBillingAccountsResult{
				{
					ProcurementAccountID: "11111",
					BillingAccountID:     "",
					Error:                "billing account not found in marketplace",
				},
				{
					ProcurementAccountID: "22222",
					BillingAccountID:     "billingAccount2",
					Error:                "",
				},
			},
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, "11111").
					Return(account1, nil).
					Once()
				f.accountDAL.
					On("GetAccount", testutils.ContextBackgroundMock, "22222").
					Return(account2, nil).
					Once()

				f.customerDAL.
					On("GetRef", testutils.ContextBackgroundMock, "someAccountID1").
					Return(customerRef1).
					Once()
				f.customerDAL.
					On("GetRef", testutils.ContextBackgroundMock, "someAccountID2").
					Return(customerRef2).
					Once()

				f.assetDAL.
					On("GetCustomerGCPAssetsWithTypes",
						testutils.ContextBackgroundMock,
						customerRef1,
						gcpAssetsFilters,
					).
					Return(gcpAssets1, nil).
					Once()
				f.assetDAL.
					On("GetCustomerGCPAssetsWithTypes",
						testutils.ContextBackgroundMock,
						customerRef2,
						gcpAssetsFilters,
					).
					Return(gcpAssets2, nil).
					Once()

				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "billingAccount1",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "11111",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{}, nil).
					Once()

				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "someOtherBillingAccount3",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "22222",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{}, nil).
					Once()

				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "billingAccount2",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "22222",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()

				f.accountDAL.
					On("UpdateGcpBillingAccountDetails",
						testutils.ContextBackgroundMock,
						"22222",
						dal.BillingAccountDetails{
							BillingAccountID:   "billingAccount2",
							BillingAccountType: assetsPkg.AssetStandaloneGoogleCloud,
						},
					).
					Return(nil).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
				accountDAL:     &dalMocks.IAccountFirestoreDAL{},
				customerDAL:    &customerMocks.Customers{},
				assetDAL:       &assetsMocks.Assets{},
			}

			s := &MarketplaceService{
				procurementDAL: tt.fields.procurementDAL,
				accountDAL:     tt.fields.accountDAL,
				customerDAL:    tt.fields.customerDAL,
				assetDAL:       tt.fields.assetDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result, err := s.PopulateBillingAccounts(tt.args.ctx, tt.args.populateBillingAccounts)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.PopulateBillingAccounts() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestMarketplaceService_entitlementExists(t *testing.T) {
	type fields struct {
		procurementDAL *dalMocks.ProcurementDAL
	}

	type args struct {
		ctx                  context.Context
		procurementAccountID string
		billingAccountID     string
	}

	tests := []struct {
		name           string
		args           args
		fields         fields
		expectedResult bool
		wantErr        bool
		on             func(*fields)
	}{
		{
			name: "entitlement exists",
			args: args{
				ctx:                  context.Background(),
				procurementAccountID: "someProcurementAccount1",
				billingAccountID:     "someBillingAccount1",
			},
			expectedResult: true,
			on: func(f *fields) {
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "someBillingAccount1",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "someProcurementAccount1",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()
			},
		},
		{
			name: "entitlement does not exist",
			args: args{
				ctx:                  context.Background(),
				procurementAccountID: "someProcurementAccount1",
				billingAccountID:     "someBillingAccount1",
			},
			expectedResult: false,
			on: func(f *fields) {
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "someBillingAccount1",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "someProcurementAccount1",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{}, nil).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
			}

			s := &MarketplaceService{
				procurementDAL: tt.fields.procurementDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result, err := s.entitlementExists(tt.args.ctx, tt.args.procurementAccountID, tt.args.billingAccountID)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.entitlementExists() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestMarketplaceService_getMarketplaceBillingAccount(t *testing.T) {
	type fields struct {
		procurementDAL *dalMocks.ProcurementDAL
	}

	type args struct {
		ctx                  context.Context
		procurementAccountID string
		billingAccountIDs    []string
	}

	tests := []struct {
		name           string
		args           args
		fields         fields
		expectedResult string
		wantErr        bool
		on             func(*fields)
	}{
		{
			name: "billing account exists",
			args: args{
				ctx:                  context.Background(),
				procurementAccountID: "someProcurementAccount1",
				billingAccountIDs:    []string{"someBillingAccount1", "someBillingAccount2"},
			},
			expectedResult: "someBillingAccount1",
			wantErr:        false,
			on: func(f *fields) {
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "someBillingAccount1",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "someProcurementAccount1",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{{}}, nil).
					Once()
			},
		},
		{
			name: "billing account does not exist",
			args: args{
				ctx:                  context.Background(),
				procurementAccountID: "someProcurementAccount1",
				billingAccountIDs:    []string{"someBillingAccount1", "someBillingAccount2"},
			},
			expectedResult: "",
			wantErr:        true,
			on: func(f *fields) {
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "someBillingAccount1",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "someProcurementAccount1",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{}, nil).
					Once()
				f.procurementDAL.
					On("ListEntitlements", testutils.ContextBackgroundMock, dal.Filter{
						Key:   dal.EntitlementFilterKeyCustomerBillingAccount,
						Value: "someBillingAccount2",
					},
						dal.Filter{
							Key:   dal.EntitlementFilterKeyAccount,
							Value: "someProcurementAccount1",
						}).
					Return([]*cloudcommerceprocurement.Entitlement{}, nil).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				procurementDAL: &dalMocks.ProcurementDAL{},
			}

			s := &MarketplaceService{
				procurementDAL: tt.fields.procurementDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result, err := s.getMarketplaceBillingAccount(tt.args.ctx, tt.args.procurementAccountID, tt.args.billingAccountIDs)

			if (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.getMarketplaceBillingAccount() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestMarketplaceService_getAllPopulateBillingAccounts(t *testing.T) {
	type fields struct {
		accountDAL *dalMocks.IAccountFirestoreDAL
	}

	type args struct {
		ctx                     context.Context
		populateBillingAccounts *domain.PopulateBillingAccounts
	}

	tests := []struct {
		name           string
		args           args
		fields         fields
		expectedResult *domain.PopulateBillingAccounts
		wantErr        bool
		on             func(*fields)
	}{
		{
			name: "returns two billing accounts",
			args: args{
				ctx:                     context.Background(),
				populateBillingAccounts: &domain.PopulateBillingAccounts{},
			},
			wantErr: false,
			expectedResult: &domain.PopulateBillingAccounts{
				{
					ProcurementAccountID: "111",
				},
				{
					ProcurementAccountID: "222",
				},
			},
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccountsIDs", testutils.ContextBackgroundMock).
					Return([]string{"111", "222"}, nil).
					Once()
			},
		},
		{
			name: "error",
			args: args{
				ctx:                     context.Background(),
				populateBillingAccounts: &domain.PopulateBillingAccounts{},
			},
			wantErr: true,
			on: func(f *fields) {
				f.accountDAL.
					On("GetAccountsIDs", testutils.ContextBackgroundMock).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				accountDAL: &dalMocks.IAccountFirestoreDAL{},
			}

			s := &MarketplaceService{
				accountDAL: tt.fields.accountDAL,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if err := s.getAllPopulateBillingAccounts(tt.args.ctx, tt.args.populateBillingAccounts); (err != nil) != tt.wantErr {
				t.Errorf("MarketplaceService.getAllPopulateBillingAccounts() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				assert.Equal(t, tt.expectedResult, tt.args.populateBillingAccounts)
			}
		})
	}
}

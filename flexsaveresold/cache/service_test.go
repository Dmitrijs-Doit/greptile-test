package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/credits"
	mocks2 "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/credits/mocks"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"

	fsdal "github.com/doitintl/firestore"
	shared_mocks "github.com/doitintl/firestore/mocks"
	fspkg "github.com/doitintl/firestore/pkg"
	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	amazonwebservices "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/mocks"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

var enabledAt = time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC)
var customerID1 = "N7d7maRbHCuoqbLVlmrS"
var customerID2 = "ImoC9XkrutBysJvyqlBm"

var existingCache = &fspkg.FlexsaveConfiguration{
	AWS: fspkg.FlexsaveSavings{
		Enabled:     true,
		TimeEnabled: &enabledAt,
		Notified:    false,
		SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
			"5_2021": {
				Savings:       0,
				OnDemandSpend: 123443.39,
			},
			"6_2021": {
				Savings:       0,
				OnDemandSpend: 875575.56,
			},
			"7_2021": {
				Savings:       0,
				OnDemandSpend: 98349.02,
			},
			"8_2021": {
				Savings:       0,
				OnDemandSpend: 23453.13,
			},
			"9_2021": {
				Savings:       0,
				OnDemandSpend: 34566.97,
			},
			"10_2021": {
				Savings:       2340,
				OnDemandSpend: 23456.9,
			},
			"11_2021": {
				Savings:       4340,
				OnDemandSpend: 89009.9,
			},
			"12_2021": {
				Savings:       123240,
				OnDemandSpend: 14698.9,
			},
			"1_2022": {
				Savings:       1230,
				OnDemandSpend: 56773.9,
			},
			"2_2022": {
				Savings:       18983.87,
				OnDemandSpend: 1343.9,
			},
			"3_2022": {
				Savings:       9038.9,
				OnDemandSpend: 15012.9,
			},
			"4_2022": {
				Savings:       154343.87,
				OnDemandSpend: 243434.9,
			},
			"5_2022": {
				Savings:       19873.87,
				OnDemandSpend: 54213.9,
			},
			"6_2022": {
				Savings:       17638.87,
				OnDemandSpend: 18100.6,
			},
		},
		SavingsSummary: &fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month: "5_2022",
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings:       7638,
				OnDemandSpend: 18100,
			},
		},
	},
}

var sharedCacheNotEnabled = fspkg.FlexsaveSavings{
	Enabled:          false,
	ReasonCantEnable: "",
	TimeEnabled:      &enabledAt,
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month: "7_2022",
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{},
	},
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{},
}

func TestNewService(t *testing.T) {
	ctx := context.Background()

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Error(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Error(err)
	}

	s := NewService(logger.FromContext, conn)
	assert.NotNil(t, s)
	assert.NotNil(t, s.DedicatedPayerService)
}

func TestCreateForSingleCustomer(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider  loggerMocks.ILogger
		shared          mocks.ServiceInterface
		dedicated       mocks.ServiceInterface
		customersDAL    customerMocks.Customers
		assetsDAL       assetDalMocks.Assets
		integrationsDAL shared_mocks.Integrations
		contractsDAL    shared_mocks.Contracts
		mpaDAL          mpaMocks.MasterPayerAccounts
		payerService    payerMocks.Service
		creditsService  mocks2.CreditServiceMock
	}

	type args struct {
		ctx        *context.Context
		customerID string
	}

	customerAttributes := pkg.CustomerInputAttributes{
		CustomerID:  customerID1,
		TimeEnabled: &enabledAt,
		IsEnabled:   true,
		AssetIDs:    []string{"bill", "bob"},
		PayerIDs:    []string{"5", "6"},
	}

	customerAttributes2 := pkg.CustomerInputAttributes{
		CustomerID:              customerID2,
		TimeEnabled:             &enabledAt,
		IsEnabled:               true,
		AssetIDs:                []string{"asso", "assini"},
		PayerIDs:                []string{"9", "10"},
		DedicatedPayerStartTime: &enabledAt,
	}

	customerAttributesNotEnabled := pkg.CustomerInputAttributes{
		CustomerID:  customerID2,
		TimeEnabled: nil,
		IsEnabled:   false,
		AssetIDs:    []string{},
		PayerIDs:    []string{},
	}

	var payerAccounts amazonwebservices.MasterPayerAccounts

	notYetOnboardedDate := time.Now().AddDate(0, 1, 0)

	payerAccounts.Accounts = map[string]*amazonwebservices.MasterPayerAccount{
		"9": {AccountNumber: "9",
			TenancyType: "dedicated",
			Features: &amazonwebservices.Features{
				BillingStartDate: &enabledAt,
			}},
		"10": {AccountNumber: "10",
			TenancyType: "dedicated",
			Features: &amazonwebservices.Features{
				BillingStartDate: &enabledAt,
			}},
		"5": {AccountNumber: "5",
			TenancyType: "dedicated"},
		"6": {AccountNumber: "6",
			TenancyType: "dedicated"},
		"7": {AccountNumber: "7",
			TenancyType: "dedicated"},
		"1": {AccountNumber: "1",
			TenancyType: "dedicated"},
		"13": {AccountNumber: "13",
			TenancyType: "dedicated",
			Status:      "retired",
		},
		"14": {AccountNumber: "14",
			TenancyType: "dedicated",
			Features: &amazonwebservices.Features{
				BillingStartDate: &notYetOnboardedDate,
			}},
	}

	var noAssetsCache fspkg.FlexsaveSavings

	errNoAssets := "no assets"
	noAssetsCache.ReasonCantEnable = errNoAssets
	noAssetsCache.Enabled = false

	var noContractCache = fspkg.FlexsaveSavings{
		ReasonCantEnable: errNoContract,
	}

	notOnboardedCacheDocument := fspkg.FlexsaveSavings{
		ReasonCantEnable: errNotOnBoarded,
	}

	nowFunc := func() time.Time {
		return time.Date(2022, 7, 5, 0, 0, 0, 0, time.UTC)
	}

	timeParams := pkg.TimeParams{Now: nowFunc(), CurrentMonth: "7_2022", ApplicableMonths: []string{"7_2022", "6_2022", "5_2022", "4_2022", "3_2022", "2_2022", "1_2022", "12_2021", "11_2021", "10_2021", "9_2021", "8_2021"}, DaysInCurrentMonth: 31, DaysInNextMonth: 31, PreviousMonth: "6_2022"}

	customerRef := &firestore.DocumentRef{ID: customerID1}
	customerRef2 := &firestore.DocumentRef{ID: customerID2}

	assetsByPayer := make(map[string][]string)
	assetsByPayer2 := make(map[string][]string)

	assetsByPayer2["333"] = append(assetsByPayer["333"], "000")

	tests := []struct {
		name    string
		args    args
		wantErr error
		want    *fspkg.FlexsaveSavings
		fields  fields
		on      func(*fields)
		assert  func(*testing.T, *fields)
		now     func() time.Time
	}{
		{
			name: "dedicated customer",
			args: args{
				ctx:        &ctx,
				customerID: customerID1,
			},
			on: func(f *fields) {
				f.customersDAL.On("GetRef", testutils.ContextBackgroundMock, customerID1).Return(customerRef)
				f.contractsDAL.On("GetActiveCustomerContractsForProductTypeAndMonth", testutils.ContextBackgroundMock, customerRef, mock.AnythingOfType("time.Time"), "amazon-web-services").Return([]*fspkg.Contract{
					{
						ID: "contract",
					},
				}, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1).Return(existingCache, nil)

				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1, mock.Anything).Return(nil).Once()

				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "5",
								},
							},
						},
					},
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "bob",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "6",
								},
							},
						},
					},
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "fran",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "13",
								},
							},
						},
					},
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "erica",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "not-in-MPA-collection",
								},
							},
						},
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)

				f.dedicated.On("GetCache", testutils.ContextBackgroundMock, customerAttributes, timeParams).
					Return(&mocks.SingleCache, nil).Once()
				f.shared.On("GetCache", testutils.ContextBackgroundMock, customerAttributes, timeParams).
					Return(nil, nil).Once()
			},
			now: nowFunc,
			assert: func(t *testing.T, f *fields) {
				f.integrationsDAL.AssertNumberOfCalls(t, "GetFlexsaveConfigurationCustomer", 1)
				f.customersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.dedicated.AssertNumberOfCalls(t, "GetCache", 1)
				f.shared.AssertNumberOfCalls(t, "GetCache", 1)
				f.assetsDAL.AssertNumberOfCalls(t, "GetCustomerAWSAssets", 1)
			},
			want: &mocks.SingleCache,
		},
		{
			name: "dedicated and shared customer",
			args: args{
				ctx:        &ctx,
				customerID: customerID2,
			},
			on: func(f *fields) {
				f.customersDAL.On("GetRef", testutils.ContextBackgroundMock, customerID2).Return(customerRef2)
				f.contractsDAL.On("GetActiveCustomerContractsForProductTypeAndMonth", testutils.ContextBackgroundMock, customerRef2, mock.AnythingOfType("time.Time"), "amazon-web-services").Return([]*fspkg.Contract{
					&fspkg.Contract{
						ID: "contract",
					},
				}, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2).Return(mocks.ExistingCache2, nil)

				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2, mock.Anything).Return(nil).Once()

				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef2.ID).Return([]*assets.AWSAsset{

					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "asso",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "9",
								},
							},
						},
					},
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "assini",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "10",
								},
							},
						},
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)

				f.dedicated.On("GetCache", testutils.ContextBackgroundMock, customerAttributes2, timeParams).
					Return(&mocks.DedicatedCache, nil).Once()
				f.shared.On("GetCache", testutils.ContextBackgroundMock, customerAttributes2, timeParams).
					Return(&mocks.EnabledCacheTwo, nil).Once()
			},
			now: nowFunc,
			assert: func(t *testing.T, f *fields) {
				f.integrationsDAL.AssertNumberOfCalls(t, "GetFlexsaveConfigurationCustomer", 1)
				f.customersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.dedicated.AssertNumberOfCalls(t, "GetCache", 1)
				f.shared.AssertNumberOfCalls(t, "GetCache", 1)
				f.assetsDAL.AssertNumberOfCalls(t, "GetCustomerAWSAssets", 1)
			},
			want: &mocks.MergedDedicatedAndSharedCache,
		},
		{
			name: "no assets",
			args: args{
				ctx:        &ctx,
				customerID: customerID2,
			},
			on: func(f *fields) {
				f.customersDAL.On("GetRef", testutils.ContextBackgroundMock, customerID2).Return(customerRef2)
				f.contractsDAL.On("GetActiveCustomerContractsForProductTypeAndMonth", testutils.ContextBackgroundMock, customerRef2, mock.AnythingOfType("time.Time"), "amazon-web-services").Return([]*fspkg.Contract{
					&fspkg.Contract{
						ID: "contract",
					},
				}, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2).Return(nil, fsdal.ErrNotFound)

				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef2.ID).Return([]*assets.AWSAsset{}, nil)

				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2, mock.Anything).Return(nil).Once()

				f.dedicated.On("GetCache", testutils.ContextBackgroundMock, customerAttributesNotEnabled, timeParams).
					Return(nil, nil).Once()
				f.shared.On("GetCache", testutils.ContextBackgroundMock, customerAttributesNotEnabled, timeParams).
					Return(nil, nil).Once()
			},
			now: nowFunc,
			assert: func(t *testing.T, f *fields) {
				f.integrationsDAL.AssertNumberOfCalls(t, "GetFlexsaveConfigurationCustomer", 1)
				f.customersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.dedicated.AssertNumberOfCalls(t, "GetCache", 1)
				f.shared.AssertNumberOfCalls(t, "GetCache", 1)
				f.assetsDAL.AssertNumberOfCalls(t, "GetCustomerAWSAssets", 1)
			},
			want: &noAssetsCache,
		},
		{
			name: "no contract",
			args: args{
				ctx:        &ctx,
				customerID: customerID2,
			},
			on: func(f *fields) {
				f.customersDAL.On("GetRef", testutils.ContextBackgroundMock, customerID2).Return(customerRef2)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2).Return(nil, fsdal.ErrNotFound)
				f.contractsDAL.On("GetActiveCustomerContractsForProductTypeAndMonth", testutils.ContextBackgroundMock, customerRef2, mock.AnythingOfType("time.Time"), "amazon-web-services").Return([]*fspkg.Contract{}, nil)
				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef2.ID).Return([]*assets.AWSAsset{}, nil)
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2, mock.Anything).Return(nil).Once()
			},
			now: nowFunc,
			assert: func(t *testing.T, f *fields) {
				f.integrationsDAL.AssertNumberOfCalls(t, "GetFlexsaveConfigurationCustomer", 1)
				f.contractsDAL.AssertNumberOfCalls(t, "GetActiveCustomerContractsForProductTypeAndMonth", 1)
				f.assetsDAL.AssertNumberOfCalls(t, "GetCustomerAWSAssets", 1)
			},
			want: &noContractCache,
		},
		{
			name: "not yet onboarded",
			args: args{
				ctx:        &ctx,
				customerID: customerID2,
			},
			on: func(f *fields) {
				f.customersDAL.On("GetRef", testutils.ContextBackgroundMock, customerID2).Return(customerRef2)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2).Return(nil, fsdal.ErrNotFound)
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID2, map[string]*fspkg.FlexsaveSavings{"AWS": &notOnboardedCacheDocument}).Return(nil).Once()

				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef2.ID).Return([]*assets.AWSAsset{
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID2}},
						Properties: &assets.AWSProperties{
							AccountID: "terry",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "14",
								},
							},
						},
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			now: nowFunc,
			assert: func(t *testing.T, f *fields) {
				f.integrationsDAL.AssertNumberOfCalls(t, "GetFlexsaveConfigurationCustomer", 1)
				f.customersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.dedicated.AssertNumberOfCalls(t, "GetCache", 1)
				f.shared.AssertNumberOfCalls(t, "GetCache", 1)
				f.assetsDAL.AssertNumberOfCalls(t, "GetCustomerAWSAssets", 1)
			},
			want: &notOnboardedCacheDocument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}

			ctx := context.Background()

			log, err := logger.NewLogging(ctx)
			if err != nil {
				t.Error(err)
			}

			conn, err := connection.NewConnection(ctx, log)
			if err != nil {
				t.Error(err)
			}

			s := &Service{
				LoggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				Connection:             conn,
				SharedPayerService:     &f.shared,
				DedicatedPayerService:  &f.dedicated,
				IntegrationsDAL:        &f.integrationsDAL,
				AssetsDAL:              &f.assetsDAL,
				CustomersDAL:           &f.customersDAL,
				ContractsDAL:           &f.contractsDAL,
				Now:                    tt.now,
				masterPayerAccountsDAL: &f.mpaDAL,
				payer:                  &f.payerService,
				creditsService:         &f.creditsService,
			}

			f.payerService.On("GetPayerConfigsForCustomer", mock.Anything, mock.Anything).Return([]*types.PayerConfig{}, nil)
			f.creditsService.On("HandleCustomerCredits", mock.Anything, mock.Anything).Return(nil)

			if tt.on != nil {
				tt.on(f)
			}

			got, err := s.RunCacheForSingleCustomer(context.Background(), tt.args.customerID)

			if err != nil {
				expectedError := tt.wantErr
				if err.Error() != expectedError.Error() {
					t.Errorf("RunCacheForSingleCustomer() error = %v, wantErr %v", err, &expectedError)
					return
				}
			}

			assert.Equalf(t, tt.want, got, "RunCacheForSingleCustomer() got = %v, want %v", got, tt.want)
		})
	}
}
func TestPrioritizeReasonCantEnable(t *testing.T) {
	assert.Equal(t, getReasonCantEnable([]string{credits.ErrCustomerHasAwsActivateCredits, noError, errLowSpend, errCHNotConfigured, errNoSpendInThirtyDays, errNoSpend, errNoAssets}), noError)
	assert.Equal(t, getReasonCantEnable([]string{credits.ErrCustomerHasAwsActivateCredits, errLowSpend, errCHNotConfigured, errNoSpendInThirtyDays, errNoSpend, errNoAssets}), credits.ErrCustomerHasAwsActivateCredits)
	assert.Equal(t, getReasonCantEnable([]string{noError, errLowSpend, errCHNotConfigured, errNoSpendInThirtyDays, errNoSpend, errNoAssets}), noError)
	assert.Equal(t, getReasonCantEnable([]string{errLowSpend, errCHNotConfigured, errNoSpendInThirtyDays, errNoSpend, errNoAssets}), errLowSpend)
	assert.Equal(t, getReasonCantEnable([]string{errCHNotConfigured, errNoSpendInThirtyDays, errNoSpend, errNoAssets}), errCHNotConfigured)
	assert.Equal(t, getReasonCantEnable([]string{errNoSpendInThirtyDays, errNoSpend, errNoAssets}), errNoSpendInThirtyDays)
	assert.Equal(t, getReasonCantEnable([]string{errCHNotConfigured, errFetchingRecommendations}), errFetchingRecommendations)
	assert.Equal(t, getReasonCantEnable([]string{errNoSpend, errNoAssets}), errNoSpend)
	assert.Equal(t, getReasonCantEnable([]string{errNoAssets}), errNoAssets)
	assert.Equal(t, getReasonCantEnable([]string{}), errOther)
}

func TestMerge(t *testing.T) {
	aggregated := mergeSavingsHistory(&mocks.EnabledCacheOne, &mocks.EnabledCacheTwo)

	if !reflect.DeepEqual(aggregated, mocks.MergedCache.SavingsHistory) {
		t.Errorf("mergeSavingsHistory = %v, want %v", aggregated, mocks.MergedCache.SavingsHistory)
	}
}

func TestMergeSavingsSummary(t *testing.T) {
	var hourlyCommitmentFloat = 1.88

	mergedSavingsSummary := mergeSavingsSummary(&mocks.DedicatedTestCacheNotEnabled, &sharedCacheNotEnabled)
	assert.Equal(t, mergedSavingsSummary, &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month:            "7_2022",
			ProjectedSavings: 0,
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			HourlyCommitment: &hourlyCommitmentFloat,
			Savings:          469.6,
		},
	},
	)
}

func TestService_getAssetAndPayerInfo(t *testing.T) {
	type fields struct {
		AssetsDAL assetDalMocks.Assets
		mpaDAL    mpaMocks.MasterPayerAccounts
	}

	type args struct {
		ctx         context.Context
		customerRef *firestore.DocumentRef
	}

	var payerAccounts amazonwebservices.MasterPayerAccounts

	enabledAtEarlier := enabledAt.AddDate(0, -1, 0)

	payerAccounts.Accounts = map[string]*amazonwebservices.MasterPayerAccount{
		"9": {AccountNumber: "9",
			TenancyType: "dedicated",
			Features: &amazonwebservices.Features{
				BillingStartDate: &enabledAt,
			}},
		"10": {AccountNumber: "10",
			TenancyType: "dedicated",
			Features: &amazonwebservices.Features{
				BillingStartDate: &enabledAtEarlier,
			}},
		"5": {AccountNumber: "5",
			TenancyType: "dedicated"},
		"6": {AccountNumber: "6",
			TenancyType: "dedicated"},
		"7": {AccountNumber: "7",
			TenancyType: "dedicated"},
		"1": {AccountNumber: "1",
			TenancyType: "dedicated"},
		"13": {AccountNumber: "13",
			TenancyType: "dedicated",
			Status:      "retired",
		},
	}

	customerRef := &firestore.DocumentRef{ID: customerID1}

	var tests = []struct {
		name         string
		args         args
		wantPayerIDs []string
		wantTime     *time.Time
		wantAssetIDs []string
		wantErr      bool
		on           func(*fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerID1).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "9",
								},
							},
						},
					},
					{
						Properties: &assets.AWSProperties{
							AccountID: "bob",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "10",
								},
							},
						},
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{"9", "10"},
			wantTime:     &enabledAtEarlier,
			wantAssetIDs: []string{"bill", "bob"},
			wantErr:      false,
		},
		{
			name: "Happy path - payer ID does not exist in MPA collection",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "9",
								},
							},
						},
					},
					{
						Properties: &assets.AWSProperties{
							AccountID: "bob",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "6",
								},
							},
						},
					},
					{
						Properties: &assets.AWSProperties{
							AccountID: "erica",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "1000001",
								},
							},
						},
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{"9", "6"},
			wantTime:     &enabledAt,
			wantAssetIDs: []string{"bill", "bob"},
			wantErr:      false,
		},
		{
			name: "Happy path - one retired MPA",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID: "harry",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "13",
								},
							},
						},
					},
					{
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "9",
								},
							},
						},
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{"9"},
			wantTime:     &enabledAt,
			wantAssetIDs: []string{"bill"},
			wantErr:      false,
		},
		{
			name: "Happy path - does not add duplicate payer IDs",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID: "harry",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "9",
								},
							},
						},
					},
					{
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "9",
								},
							},
						},
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{"9"},
			wantTime:     &enabledAt,
			wantAssetIDs: []string{"harry", "bill"},
			wantErr:      false,
		},
		{
			name: "Error while fetching assets",
			args: args{
				ctx:         context.Background(),
				customerRef: &firestore.DocumentRef{ID: customerID1},
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return(nil, errors.New("fetch error"))
			},
			wantPayerIDs: []string{},
			wantTime:     nil,
			wantAssetIDs: []string{},
			wantErr:      true,
		},
		{
			name: "No assets for customer",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{}, nil)
			},
			wantPayerIDs: []string{},
			wantTime:     nil,
			wantAssetIDs: []string{},
			wantErr:      false,
		},
		{
			name: "Error while fetching payer accounts",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "5",
								},
							},
						},
					},
					{
						Properties: &assets.AWSProperties{
							AccountID: "bob",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "6",
								},
							},
						},
					},
				}, nil)

				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(nil, errors.New("fetch error"))
			},
			wantPayerIDs: []string{},
			wantTime:     nil,
			wantAssetIDs: []string{},
			wantErr:      true,
		},
		{
			name: "Asset has no properties",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: nil,
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{},
			wantTime:     nil,
			wantAssetIDs: []string{},
			wantErr:      false,
		},
		{
			name: "Asset has properties, but no organization info",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID: "bill",
						},
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{},
			wantTime:     nil,
			wantAssetIDs: []string{},
			wantErr:      false,
		},
		{
			name: "Asset has organization info, but no payer account",
			args: args{
				ctx:         context.Background(),
				customerRef: customerRef,
			},
			on: func(f *fields) {
				f.AssetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						Properties: &assets.AWSProperties{
							AccountID:        "bill",
							OrganizationInfo: &assets.OrganizationInfo{},
						},
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccounts, nil)
			},
			wantPayerIDs: []string{},
			wantTime:     nil,
			wantAssetIDs: []string{},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			ctx := context.Background()

			log, err := logger.NewLogging(ctx)
			if err != nil {
				t.Error(err)
			}

			conn, err := connection.NewConnection(ctx, log)
			if err != nil {
				t.Error(err)
			}

			s := &Service{
				Connection:             conn,
				AssetsDAL:              &fields.AssetsDAL,
				masterPayerAccountsDAL: &fields.mpaDAL,
			}

			gotPayerIDs, gotTime, gotAssetIDs, err := s.getAssetAndPayerInfo(tt.args.ctx, tt.args.customerRef)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.getAssetAndPayerInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotPayerIDs, tt.wantPayerIDs) {
				t.Errorf("Service.getAssetAndPayerInfo() gotPayerIDs = %v, want %v", gotPayerIDs, tt.wantPayerIDs)
			}

			if !reflect.DeepEqual(gotTime, tt.wantTime) {
				t.Errorf("Service.getAssetAndPayerInfo() gotTime = %v, want %v", gotTime, tt.wantTime)
			}

			if !reflect.DeepEqual(gotAssetIDs, tt.wantAssetIDs) {
				t.Errorf("Service.getAssetAndPayerInfo() gotAssetIDs = %v, want %v", gotAssetIDs, tt.wantAssetIDs)
			}
		})
	}
}

func TestService_hasActiveResold(t *testing.T) {
	type fields struct {
		payerService payerMocks.Service
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    bool
		wantErr bool
	}{
		{
			name: "returns error",
			on: func(f *fields) {
				f.payerService.On("GetPayerConfigsForCustomer", mock.Anything, mock.Anything).Return(nil, errors.New("error"))
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "returns true",
			on: func(f *fields) {
				f.payerService.On("GetPayerConfigsForCustomer", mock.Anything, mock.Anything).Return([]*types.PayerConfig{
					{
						Status: "active",
						Type:   resoldConfigType,
					},
				}, nil)
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "returns false",
			on: func(f *fields) {
				f.payerService.On("GetPayerConfigsForCustomer", mock.Anything, mock.Anything).Return([]*types.PayerConfig{
					{
						Status: "disabled",
						Type:   resoldConfigType,
					},
				}, nil)
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				payerService: payerMocks.Service{},
			}

			s := &Service{
				payer: &f.payerService,
			}

			tt.on(f)

			got, err := s.hasActiveResold(context.Background(), "test-1")

			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equalf(t, tt.want, got, "hasActiveResold")
		})
	}
}

package savingsplans

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	amazonwebservices "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	chtMocks "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal/mocks"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	bqmocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestSavingsPlanService_CustomerSavingsPlansCache(t *testing.T) {
	type fields struct {
		loggerProvider loggerMocks.ILogger
		*connection.Connection

		savingsPlansDAL mocks.SavingsPlansDAL
		bigQueryService bqmocks.BigQueryServiceInterface
		cloudHealthDAL  chtMocks.CloudHealthDAL
		customerDAL     customerMocks.Customers
		assetsDAL       assetDalMocks.Assets
		mpaDAL          mpaMocks.MasterPayerAccounts
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	ctx := context.Background()
	customerID := "mr_customer"
	cloudHealthCustomerID := "1"

	errExample := errors.New("something has gone wrong")

	savingsPlans := []types.SavingsPlanData{
		{
			SavingsPlanID:    "5631fb63-2450-4656-91ac-9c77efceb341",
			UpfrontPayment:   0,
			RecurringPayment: 15,
			Commitment:       20,
			ExpirationDate:   time.Date(2022, 7, 5, 0, 0, 0, 0, time.UTC),
		},
	}

	enabledAtNewCustomer := time.Now().UTC().AddDate(0, 0, -2)

	var payerAccountsNewCustomer amazonwebservices.MasterPayerAccounts

	payerAccountsNewCustomer.Accounts = map[string]*amazonwebservices.MasterPayerAccount{
		"9": {AccountNumber: "9",
			TenancyType:    "dedicated",
			OnboardingDate: &enabledAtNewCustomer,
		},
	}

	enabledAt := time.Now().UTC().AddDate(0, -1, 0)

	var payerAccounts amazonwebservices.MasterPayerAccounts

	payerAccounts.Accounts = map[string]*amazonwebservices.MasterPayerAccount{
		"9": {AccountNumber: "9",
			TenancyType:    "dedicated",
			OnboardingDate: &enabledAt,
		},
	}

	var payerAccountsSharedPayer amazonwebservices.MasterPayerAccounts

	payerAccountsSharedPayer.Accounts = map[string]*amazonwebservices.MasterPayerAccount{
		"1": {AccountNumber: "1",
			TenancyType: "shared",
		},
	}

	customerRef := &firestore.DocumentRef{ID: "mr_customer"}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    []types.SavingsPlanData
	}{
		{
			name: "happy path",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(nil)
				f.bigQueryService.On("GetCustomerSavingsPlanData", ctx, customerID).Return(savingsPlans, nil)
				f.savingsPlansDAL.On("CreateCustomerSavingsPlansCache", ctx, customerID, savingsPlans).Return(nil)
			},
			want: savingsPlans,
		},
		{
			name: "no recalculated customer table - shared payer",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(bq.ErrNoActiveTable)
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerRef)
				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID}},
						Properties: &assets.AWSProperties{
							AccountID: "bill",
							OrganizationInfo: &assets.OrganizationInfo{
								PayerAccount: &amazonwebservices.PayerAccount{
									AccountID: "1",
								},
							},
						},
					},
				}, nil)
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccountsSharedPayer, nil)
				f.cloudHealthDAL.On("GetCustomerCloudHealthID", ctx, customerRef).Return(cloudHealthCustomerID, nil)
				f.bigQueryService.On("GetCustomerSavingsPlanData", ctx, cloudHealthCustomerID).Return(savingsPlans, nil)
				f.savingsPlansDAL.On("CreateCustomerSavingsPlansCache", ctx, customerID, savingsPlans).Return(nil)
			},
			want: savingsPlans,
		},
		{
			name: "no recalculated customer table - no assets error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(bq.ErrNoActiveTable)
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerRef)
				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{}, nil)
				f.loggerProvider.On("Warningf", "no aws assets found for customer: %s", customerID)
			},
			wantErr: nil,
		},
		{
			name: "bigquery error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(errExample)
			},
			wantErr: errExample,
		},
		{
			name: "cloudhealth ID error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(bq.ErrNoActiveTable)
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerRef)
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerRef)
				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID}},
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
				f.cloudHealthDAL.On("GetCustomerCloudHealthID", ctx, customerRef).Return("", errExample)
			},
			wantErr: errExample,
		},
		{
			name: "new customer with no billing table",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(bq.ErrNoActiveTable)
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerRef)
				f.assetsDAL.On("GetCustomerAWSAssets", testutils.ContextBackgroundMock, customerRef.ID).Return([]*assets.AWSAsset{
					{
						BaseAsset: assets.BaseAsset{Customer: &firestore.DocumentRef{ID: customerID}},
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
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(&payerAccountsNewCustomer, nil)
				f.loggerProvider.On("Infof", "no billing table found for new customer: %v", customerID)
			},
		},
		{
			name: "savings plan data error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(nil)
				f.bigQueryService.On("GetCustomerSavingsPlanData", ctx, customerID).Return(nil, errExample)
			},
			wantErr: errExample,
		},
		{
			name: "empty savings plans data",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(nil)
				f.bigQueryService.On("GetCustomerSavingsPlanData", ctx, customerID).Return(nil, nil)
				f.savingsPlansDAL.On("CreateCustomerSavingsPlansCache", ctx, customerID, []types.SavingsPlanData(nil)).Return(nil)
			},
		},
		{
			name: "create error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(nil)
				f.bigQueryService.On("GetCustomerSavingsPlanData", ctx, customerID).Return(savingsPlans, nil)
				f.savingsPlansDAL.On("CreateCustomerSavingsPlansCache", ctx, customerID, savingsPlans).Return(errExample)
			},
			wantErr: errExample,
		},
		{
			name: "create error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", ctx, customerID).Return(nil)
				f.bigQueryService.On("GetCustomerSavingsPlanData", ctx, customerID).Return(savingsPlans, nil)
				f.savingsPlansDAL.On("CreateCustomerSavingsPlansCache", ctx, customerID, savingsPlans).Return(errExample)
			},
			wantErr: errExample,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			log, err := logger.NewLogging(ctx)
			if err != nil {
				t.Error(err)
			}

			conn, err := connection.NewConnection(ctx, log)
			if err != nil {
				t.Error(err)
			}

			s := &Service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				Connection:             conn,
				savingsPlansDAL:        &fields.savingsPlansDAL,
				bigQueryService:        &fields.bigQueryService,
				cloudHealthDAL:         &fields.cloudHealthDAL,
				customerDAL:            &fields.customerDAL,
				assetsDAL:              &fields.assetsDAL,
				masterPayerAccountsDAL: &fields.mpaDAL,
			}

			got, err := s.CustomerSavingsPlansCache(tt.args.ctx, tt.args.customerID)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SavingsPlansService.CustomerSavingsPlansCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

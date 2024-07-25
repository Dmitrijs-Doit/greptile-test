package aws

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/bigquery/iface"
	bqmocks "github.com/doitintl/bigquery/mocks"
	"github.com/doitintl/firestore/mocks"
	fspkg "github.com/doitintl/firestore/pkg"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	flexsave "github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	standaloneMocks "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/aws/iface/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

var enabledAt = time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC)
var enabledAtFourMonthsAgo = time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC).AddDate(0, -4, 0)
var customerID1 = "N7d7maRbHCuoqbLVlmrS"

var timeInstance = time.Date(2022, time.Month(5), 5, 4, 10, 30, 0, time.UTC)

var gapiErrNotFound = &googleapi.Error{
	Code: http.StatusNotFound,
}

var mockCostRow = []itemType{
	itemType{
		Cost:    0,
		Date:    timeInstance,
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -1),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -2),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -3),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -3),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -3),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -3),
		PayerID: "272170776985",
	},

	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, -1, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    10,
		Date:    timeInstance.AddDate(0, -2, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    10,
		Date:    timeInstance.AddDate(0, -3, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    10,
		Date:    timeInstance.AddDate(0, -4, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    10,
		Date:    timeInstance.AddDate(0, -5, 0),
		PayerID: "272170776985",
	},
}

var mockSavingsRow = []itemType{
	itemType{
		Cost:    0,
		Date:    timeInstance,
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -1),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -2),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -1),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -2),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -1),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    2,
		Date:    timeInstance.AddDate(0, -3, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    2,
		Date:    timeInstance.AddDate(0, -2, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    2,
		Date:    timeInstance.AddDate(0, -3, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    2,
		Date:    timeInstance.AddDate(0, -4, 0),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, -5, 0),
		PayerID: "272170776985",
	},
}

var mockRecurringFeeRow = []itemType{
	itemType{
		Cost:    0,
		Date:    timeInstance,
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -1),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -2),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -3),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -5),
		PayerID: "272170776985",
	},
	itemType{
		Cost:    0,
		Date:    timeInstance.AddDate(0, 0, -6),
		PayerID: "272170776985",
	},
}
var existingCache = &fspkg.FlexsaveConfiguration{
	AWS: fspkg.FlexsaveSavings{
		Enabled:     false,
		TimeEnabled: nil,
		Notified:    false,
		SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
			"4_2022": {
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
			"5_2022": {
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
		},
		SavingsSummary: &fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month:            "5_2022",
				ProjectedSavings: 0,
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
		},
	},
}

var cacheToUpdate = fspkg.FlexsaveSavings{
	Enabled:     false,
	TimeEnabled: nil,
	Notified:    false,
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"4_2022": {
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
		"5_2022": {
			Savings:       0,
			OnDemandSpend: 4513.65,
			SavingsRate:   0,
		},
	},
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month:            "5_2022",
			ProjectedSavings: 0,
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings:       1000,
			OnDemandSpend: 3513.6499999999996,
			SavingsRate:   22.155018665603226,
		},
	},
}

var existingCacheEnabled = &fspkg.FlexsaveConfiguration{
	AWS: fspkg.FlexsaveSavings{
		Enabled:     true,
		TimeEnabled: &enabledAt,
		Notified:    false,
		SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
			"5_2021": {
				Savings:       0,
				OnDemandSpend: 123443.39,
				SavingsRate:   0,
			},
			"6_2021": {
				Savings:       0,
				OnDemandSpend: 875575.56,
				SavingsRate:   0,
			},
			"7_2021": {
				Savings:       0,
				OnDemandSpend: 98349.02,
				SavingsRate:   0,
			},
			"8_2021": {
				Savings:       0,
				OnDemandSpend: 23453.13,
				SavingsRate:   0,
			},
			"9_2021": {
				Savings:       0,
				OnDemandSpend: 34566.97,
				SavingsRate:   0,
			},
			"10_2021": {
				Savings:       2340,
				OnDemandSpend: 23456.9,
				SavingsRate:   0,
			},
			"11_2021": {
				Savings:       4340,
				OnDemandSpend: 89009.9,
				SavingsRate:   0,
			},
			"12_2021": {
				Savings:       123240,
				OnDemandSpend: 14698.9,
				SavingsRate:   0,
			},
			"1_2022": {
				Savings:       1230,
				OnDemandSpend: 56773.9,
				SavingsRate:   0,
			},
			"2_2022": {
				Savings:       18983.87,
				OnDemandSpend: 1343.9,
				SavingsRate:   0,
			},
			"3_2022": {
				Savings:       9038.9,
				OnDemandSpend: 15012.9,
				SavingsRate:   0,
			},
			"4_2022": {
				Savings:       154343.87,
				OnDemandSpend: 243434.9,
				SavingsRate:   0,
			},
			"5_2022": {
				Savings:       19873.87,
				OnDemandSpend: 54213.9,
				SavingsRate:   0,
			},
		},
		SavingsSummary: &fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month:            "5_2022",
				ProjectedSavings: 0,
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
		},
	},
}

var cacheToUpdateSixMonths = &fspkg.FlexsaveConfiguration{
	AWS: fspkg.FlexsaveSavings{
		Enabled:     true,
		TimeEnabled: &enabledAtFourMonthsAgo,
		Notified:    false,
		SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
			"12_2021": {
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
			"1_2022": {
				Savings:       1230,
				OnDemandSpend: 56773.9,
				SavingsRate:   0,
			},
			"2_2022": {
				Savings:       18983.87,
				OnDemandSpend: 1343.9,
				SavingsRate:   0,
			},
			"3_2022": {
				Savings:       9038.9,
				OnDemandSpend: 15012.9,
				SavingsRate:   0,
			},
			"4_2022": {
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
			"5_2022": {
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
		},
		SavingsSummary: &fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month:            "5_2022",
				ProjectedSavings: 0,
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings:       0,
				OnDemandSpend: 0,
				SavingsRate:   0,
			},
		},
	},
}

var cacheToUpdateEnabled = fspkg.FlexsaveSavings{
	Enabled:     true,
	TimeEnabled: &enabledAt,
	Notified:    false,
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"5_2021": {
			Savings:       0,
			OnDemandSpend: 123443.39,
			SavingsRate:   0,
		},
		"6_2021": {
			Savings:       0,
			OnDemandSpend: 875575.56,
			SavingsRate:   0,
		},
		"7_2021": {
			Savings:       0,
			OnDemandSpend: 98349.02,
			SavingsRate:   0,
		},
		"8_2021": {
			Savings:       0,
			OnDemandSpend: 23453.13,
			SavingsRate:   0,
		},
		"9_2021": {
			Savings:       0,
			OnDemandSpend: 34566.97,
			SavingsRate:   0,
		},
		"10_2021": {
			Savings:       2340,
			OnDemandSpend: 23456.9,
			SavingsRate:   0,
		},
		"11_2021": {
			Savings:       4340,
			OnDemandSpend: 89009.9,
			SavingsRate:   0,
		},
		"12_2021": {
			Savings:       123240,
			OnDemandSpend: 14698.9,
			SavingsRate:   0,
		},
		"1_2022": {
			Savings:       1230,
			OnDemandSpend: 56773.9,
			SavingsRate:   0,
		},
		"2_2022": {
			Savings:       18983.87,
			OnDemandSpend: 1343.9,
			SavingsRate:   0,
		},
		"3_2022": {
			Savings:       9038.9,
			OnDemandSpend: 15012.9,
			SavingsRate:   0,
		},
		"4_2022": {
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
		"5_2022": {
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
	},
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month:            "5_2022",
			ProjectedSavings: 0,
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
	},
}

var sixMonthsUpdated = fspkg.FlexsaveSavings{
	Enabled:     true,
	TimeEnabled: &enabledAtFourMonthsAgo,
	Notified:    false,
	SavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
		"12_2021": {
			Savings:       0,
			OnDemandSpend: 10,
			SavingsRate:   0,
		},
		"1_2022": {
			Savings:       2,
			OnDemandSpend: 8,
			SavingsRate:   20,
		},
		"2_2022": {
			Savings:       4,
			OnDemandSpend: 6,
			SavingsRate:   40,
		},
		"3_2022": {
			Savings:       2,
			OnDemandSpend: 8,
			SavingsRate:   20,
		},
		"4_2022": {
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
		"5_2022": {
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
	},
	SavingsSummary: &fspkg.FlexsaveSavingsSummary{
		CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
			Month:            "5_2022",
			ProjectedSavings: 0,
		},
		NextMonth: &fspkg.FlexsaveMonthSummary{
			Savings:       0,
			OnDemandSpend: 0,
			SavingsRate:   0,
		},
	},
}

func TestUpdateStandaloneCustomerSpendSummary(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider        loggerMocks.ILogger
		flexsaveStandaloneDAL mocks.FlexsaveStandalone
		contractsDAL          mocks.Contracts
		cloudConnectDAL       mocks.CloudConnect
		entitiesDAL           mocks.Entities
		integrationsDAL       mocks.Integrations
		accountManagersDAL    mocks.AccountManagers
		assetsDAL             assetDalMocks.Assets
		customersDAL          customerMocks.Customers
		queryHandler          bqmocks.QueryHandler
		bqmh                  bqmocks.BigqueryManagerHandler
		flexRIService         flexsave.Service
		awsAccess             standaloneMocks.AWSAccess
		payers                payerMocks.Service
	}

	type args struct {
		ctx            *context.Context
		customerID     string
		numberOfMonths int
	}

	payerConfigs := []*types.PayerConfig{
		{
			CustomerID:  customerID1,
			AccountID:   "272170776985",
			Status:      "active",
			Type:        "aws-flexsave-standalone",
			TimeEnabled: nil,
		},
	}

	payerConfigsMultiple := []*types.PayerConfig{
		{
			CustomerID:  customerID1,
			AccountID:   "272170776985",
			Status:      "active",
			Type:        "aws-flexsave-standalone",
			TimeEnabled: &enabledAtFourMonthsAgo,
		},
	}

	floats := []string{"1.26", "1000", "4513.65", "4444.3"}

	mockSavingsPlansPurchaseRecommendationSummary := &costexplorer.SavingsPlansPurchaseRecommendationSummary{
		HourlyCommitmentToPurchase:                 &floats[0],
		EstimatedSavingsAmount:                     &floats[1],
		EstimatedOnDemandCostWithCurrentCommitment: &floats[2],
		CurrentOnDemandSpend:                       &floats[2],
	}

	var mockSavingsPlanRecommendation = costexplorer.SavingsPlansPurchaseRecommendation{
		SavingsPlansPurchaseRecommendationSummary: mockSavingsPlansPurchaseRecommendationSummary,
	}

	var mockRecommendationOutput = &costexplorer.GetSavingsPlansPurchaseRecommendationOutput{
		Metadata:                           nil,
		NextPageToken:                      nil,
		SavingsPlansPurchaseRecommendation: &mockSavingsPlanRecommendation,
	}

	ceInput := costexplorer.GetSavingsPlansPurchaseRecommendationInput{
		AccountScope:         &payer,
		TermInYears:          &oneYear,
		PaymentOption:        &noUpfront,
		LookbackPeriodInDays: &thirtyDays,
		SavingsPlansType:     &computeSP,
	}

	tests := []struct {
		name    string
		args    args
		wantErr error
		fields  fields
		on      func(*fields)
		assert  func(*testing.T, *fields)
	}{
		{
			name: "no table",
			args: args{
				ctx:            &ctx,
				customerID:     customerID1,
				numberOfMonths: 2,
			},

			on: func(f *fields) {

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID1).Return(payerConfigs, nil)

				f.bqmh.On("GetTableMetadata", testutils.ContextBackgroundMock, mock.AnythingOfType("*bigquery.Dataset"), getCustomerTable(customerID1)).Return(nil, gapiErrNotFound)
				f.loggerProvider.On("Errorf", "error: %v getting spend summary for customer: %v", ErrNoTable, "N7d7maRbHCuoqbLVlmrS")
				f.awsAccess.On("GetSavingsPlansPurchaseRecommendation", ceInput, "272170776985").Return(mockRecommendationOutput, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1).Return(existingCache, nil)
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1, map[string]*fspkg.FlexsaveSavings{"AWS": &cacheToUpdate}).Return(nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.payers.AssertNumberOfCalls(t, "GetPayerConfigsForCustomer", 1)

			},
			wantErr: nil,
		},
		{
			name: "existing table",
			args: args{
				ctx:            &ctx,
				customerID:     customerID1,
				numberOfMonths: 2,
			},

			on: func(f *fields) {

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID1).Return(payerConfigs, nil)

				f.bqmh.On("GetTableMetadata", testutils.ContextBackgroundMock, mock.AnythingOfType("*bigquery.Dataset"), getCustomerTable(customerID1)).Return(nil, nil)

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveCoveredUsage")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveCharges")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil).
					Once()

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveRecurringFee")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil).
					Once()

				f.awsAccess.On("GetSavingsPlansPurchaseRecommendation", ceInput, "272170776985").Return(mockRecommendationOutput, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1).Return(existingCacheEnabled, nil)
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1, map[string]*fspkg.FlexsaveSavings{"AWS": &cacheToUpdateEnabled}).Return(nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.payers.AssertNumberOfCalls(t, "GetPayerConfigsForCustomer", 1)

			},
			wantErr: nil,
		},
		{
			name: "existing table",
			args: args{
				ctx:            &ctx,
				customerID:     customerID1,
				numberOfMonths: 2,
			},

			on: func(f *fields) {

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID1).Return(payerConfigs, nil)

				f.bqmh.On("GetTableMetadata", testutils.ContextBackgroundMock, mock.AnythingOfType("*bigquery.Dataset"), getCustomerTable(customerID1)).Return(nil, nil)

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveCoveredUsage")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveCharges")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil).
					Once()

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveRecurringFee")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil).
					Once()

				f.awsAccess.On("GetSavingsPlansPurchaseRecommendation", ceInput, "272170776985").Return(mockRecommendationOutput, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1).Return(existingCacheEnabled, nil)
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1, map[string]*fspkg.FlexsaveSavings{"AWS": &cacheToUpdateEnabled}).Return(nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.payers.AssertNumberOfCalls(t, "GetPayerConfigsForCustomer", 1)

			},
			wantErr: nil,
		},
		{
			name: "6 months",
			args: args{
				ctx:            &ctx,
				customerID:     customerID1,
				numberOfMonths: 6,
			},

			on: func(f *fields) {

				f.payers.On("GetPayerConfigsForCustomer", testutils.ContextBackgroundMock, customerID1).Return(payerConfigsMultiple, nil)

				f.bqmh.On("GetTableMetadata", testutils.ContextBackgroundMock, mock.AnythingOfType("*bigquery.Dataset"), getCustomerTable(customerID1)).Return(nil, nil)

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveCoveredUsage")
				})).
					Return(func() iface.RowIterator {

						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[3]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[4]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[5]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[6]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[7]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[8]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[9]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[10]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockCostRow[11]

						}).Once()

						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil).
					Once()

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveCharges")
				})).
					Return(func() iface.RowIterator {
						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[3]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[4]

						}).Once()

						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[5]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[6]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[7]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[8]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[9]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockSavingsRow[10]

						}).Once()

						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil).
					Once()

				f.queryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "FlexsaveRecurringFee")
				})).
					Return(func() iface.RowIterator {
						q := &bqmocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[0]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[1]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[2]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[3]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[4]

						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

							arg := args.Get(0).(*itemType)
							*arg = mockRecurringFeeRow[5]

						}).Once()

						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1).Return(cacheToUpdateSixMonths, nil)
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer", testutils.ContextBackgroundMock, customerID1, map[string]*fspkg.FlexsaveSavings{"AWS": &sixMonthsUpdated}).Return(nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.payers.AssertNumberOfCalls(t, "GetPayerConfigsForCustomer", 1)

			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}

			ctx := context.Background()

			client, err := bigquery.NewClient(ctx,
				"project-id",
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if err != nil {
				panic(err)
			}

			nowFunc := func() time.Time {
				return time.Date(2022, time.Month(5), 5, 4, 10, 30, 0, time.UTC)
			}

			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				flexsaveStandaloneDAL: &f.flexsaveStandaloneDAL,
				contractsDAL:          &f.contractsDAL,
				cloudConnectDAL:       &f.cloudConnectDAL,
				entitiesDAL:           &f.entitiesDAL,
				integrationsDAL:       &f.integrationsDAL,
				accountManagersDAL:    &f.accountManagersDAL,
				assetsDAL:             &f.assetsDAL,
				customersDAL:          &f.customersDAL,
				queryHandler:          &f.queryHandler,
				bigQueryClient:        client,
				bqmh:                  &f.bqmh,
				AWSAccess:             &f.awsAccess,
				flexRIService:         &f.flexRIService,
				now:                   nowFunc,
				payers:                &f.payers,
			}

			if tt.on != nil {
				tt.on(f)
			}

			err = s.UpdateStandaloneCustomerSpendSummary(*tt.args.ctx, tt.args.customerID, tt.args.numberOfMonths)

			if err != nil {
				expectedError := tt.wantErr
				if err.Error() != expectedError.Error() {
					t.Errorf("UpdateStandaloneCustomerSpendSummary() error = %v, wantErr %v", err, &expectedError)
					return
				}
			}
		})
	}
}

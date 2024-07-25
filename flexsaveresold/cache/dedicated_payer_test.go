package cache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	shared_mocks "github.com/doitintl/firestore/mocks"
	fspkg "github.com/doitintl/firestore/pkg"
	dal_mocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	bqMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
	rmocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/mocks"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	flexAPIRec "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/recommendations"
	recMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/recommendations/mocks"
	common "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func TestDedicatedPayerGetCache(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider        loggerMocks.ILogger
		bigQueryService       bqMocks.BigQueryServiceInterface
		integrationsDAL       shared_mocks.Integrations
		recommendationService recMocks.Recommendations
	}

	type args struct {
		ctx        *context.Context
		configInfo pkg.CustomerInputAttributes
		timeParams pkg.TimeParams
	}

	customerAttributes := pkg.CustomerInputAttributes{
		CustomerID:              customerID2,
		TimeEnabled:             &enabledAt,
		IsEnabled:               true,
		PayerIDs:                []string{"1", "2"},
		AssetIDs:                []string{"asso", "assini"},
		DedicatedPayerStartTime: &enabledAt,
	}

	customerAttributesSharedPayer := pkg.CustomerInputAttributes{
		CustomerID:              customerID2,
		TimeEnabled:             &enabledAt,
		IsEnabled:               true,
		PayerIDs:                []string{},
		AssetIDs:                []string{"asso", "assini"},
		DedicatedPayerStartTime: nil,
	}

	customerAttributesNotEnabled := pkg.CustomerInputAttributes{
		CustomerID:              customerID2,
		TimeEnabled:             nil,
		IsEnabled:               false,
		PayerIDs:                []string{"1", "2"},
		AssetIDs:                []string{"asso", "assini"},
		DedicatedPayerStartTime: &enabledAt,
	}

	nowFunc := func() time.Time {
		return time.Date(2022, 7, 5, 0, 0, 0, 0, time.UTC)
	}

	newlyOnBoardedPayerStartTime := nowFunc().AddDate(0, 0, -2)

	customerAttributesNotEnabledNewlyOnboarded := pkg.CustomerInputAttributes{
		CustomerID:              customerID2,
		TimeEnabled:             nil,
		IsEnabled:               false,
		PayerIDs:                []string{"1", "2"},
		AssetIDs:                []string{"asso", "assini"},
		DedicatedPayerStartTime: &newlyOnBoardedPayerStartTime,
	}

	floats := []float64{1.88, 234.8, 4513.65, 4444, 1.26}
	costExplorerRecommendationsCantEnable := &flexAPIRec.RecommendationForDedicatedPayerResponse{
		HourlyCommitmentToPurchase: &floats[4],
		EstimatedSavingsAmount:     &floats[1],
	}

	costExplorerRecommendationsCanEnable := &flexAPIRec.RecommendationForDedicatedPayerResponse{
		HourlyCommitmentToPurchase: &floats[0],
		EstimatedSavingsAmount:     &floats[1],
	}

	timeParams := pkg.TimeParams{Now: nowFunc(), CurrentMonth: "7_2022", ApplicableMonths: []string{"7_2022", "6_2022", "5_2022", "4_2022", "3_2022", "2_2022", "1_2022", "12_2021", "11_2021", "10_2021", "9_2021", "8_2021", "7_2021"}, DaysInCurrentMonth: 31, DaysInNextMonth: 31, PreviousMonth: "6_2022"}

	bqParamsEnabled := bq.BigQueryParams{
		Context:             ctx,
		CustomerID:          customerID2,
		FirstOfCurrentMonth: time.Date(nowFunc().Year(), nowFunc().Month(), 1, 0, 0, 0, 0, time.UTC),
		NumberOfMonths:      flexsaveHistoryMonthAmount,
	}

	bqParamsNotEnabled := bq.BigQueryParams{
		Context:             ctx,
		CustomerID:          customerID2,
		FirstOfCurrentMonth: time.Date(nowFunc().Year(), nowFunc().Month(), 1, 0, 0, 0, 0, time.UTC),
		NumberOfMonths:      flexsaveHistoryMonthAmount,
	}

	cacheWithSavings := fspkg.FlexsaveConfiguration{
		AWS: fspkg.FlexsaveSavings{
			DailySavingsHistory: map[string]*fspkg.FlexsaveMonthSummary{
				"2022-06-04": {
					Savings: 1.0,
				},
				"2022-06-06": {
					Savings: 1.0,
				},
				"2022-06-05": {
					Savings: 1.0,
				},
				"2022-06-01": {
					Savings: 1.0,
				},
				"2022-06-02": {
					Savings: 1.0,
				},
				"2022-06-03": {
					Savings: 1.0,
				},
				"2022-06-07": {
					Savings: 1.0,
				},
			},
		},
	}

	tests := []struct {
		name    string
		args    args
		wantErr error
		want    *fspkg.FlexsaveSavings
		fields  *fields
		on      func(*fields)
		assert  func(*testing.T, *fields)
		now     func() time.Time
	}{
		{
			name: "enabled dedicated customer",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributes,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(nil)
				f.bigQueryService.On("GetPayerSpendSummary", bqParamsEnabled).Return(rmocks.SpendSummaryReal, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", ctx, bqParamsEnabled.CustomerID).Return(&cacheWithSavings, nil)
			},
			want: &rmocks.DedicatedTestCache,
		},
		{
			name: "not enabled dedicated customer",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributesNotEnabled,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(nil)
				f.bigQueryService.On("GetPayerSpendSummary", bqParamsNotEnabled).Return(rmocks.SpendSummaryReal, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", ctx, customerID2).Return(&cacheWithSavings, nil)
				f.bigQueryService.On("GetCustomerCredits", testutils.ContextBackgroundMock, customerID2, mock.AnythingOfType("time.Time")).
					Return(bq.CreditsResult{
						Credits: map[string]float64{},
						Err:     nil,
					})
				f.recommendationService.On("FetchComputeRecommendations", ctx, "1", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCanEnable, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, "2", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCantEnable, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.recommendationService.AssertNumberOfCalls(t, "FetchComputeRecommendations", 2)

			},
			want: &rmocks.DedicatedTestCacheNotEnabled,
		},
		{
			name: "enabled dedicated customer who should have bq table",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributes,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(bq.ErrNoActiveTable)
				f.loggerProvider.On("Infof",
					"active billing table does not exist for customer %v", "ImoC9XkrutBysJvyqlBm").
					Once()
			},
			want:    nil,
			wantErr: errors.New("no active billing table found"),
		},
		{
			name: "newly onboarded dedicated customer who does not have bq table",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributesNotEnabledNewlyOnboarded,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(bq.ErrNoActiveTable)
				f.bigQueryService.On("GetCustomerCredits", testutils.ContextBackgroundMock, customerID2, mock.AnythingOfType("time.Time")).
					Return(bq.CreditsResult{
						Credits: map[string]float64{},
						Err:     nil,
					})
				f.loggerProvider.On("Infof",
					"active billing table does not exist for customer %v", "ImoC9XkrutBysJvyqlBm").
					Once()
				f.bigQueryService.On("GetPayerSpendSummary", bqParamsNotEnabled).Return(rmocks.SpendSummaryReal, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, "1", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCanEnable, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, "2", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCantEnable, nil)
			},
			want:    &rmocks.DedicatedTestCacheNoSavingsHistory,
			wantErr: nil,
		},
		{
			name: "enable dedicated customer who has credits",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributesNotEnabled,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(nil)
				f.bigQueryService.On("GetPayerSpendSummary", bqParamsEnabled).Return(rmocks.SpendSummaryReal, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", ctx, bqParamsEnabled.CustomerID).Return(&cacheWithSavings, nil)
				f.bigQueryService.On("GetCustomerCredits", testutils.ContextBackgroundMock, customerID2, mock.AnythingOfType("time.Time")).
					Return(bq.CreditsResult{
						Credits: map[string]float64{customerID2: -12.34},
						Err:     nil,
					})
				f.recommendationService.On("FetchComputeRecommendations", ctx, "1", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCanEnable, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, "2", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCantEnable, nil)
			},
			want:    &rmocks.DedicatedTestCacheNotEnabledCreditsPresent,
			wantErr: errors.New("aws activate credits"),
		},
		{
			name: "shared payer only customer who does not have bq table",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributesSharedPayer,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(bq.ErrNoActiveTable)
				f.loggerProvider.On("Infof",
					"active billing table does not exist for customer %v", "ImoC9XkrutBysJvyqlBm").
					Once()
				f.bigQueryService.On("GetPayerSpendSummary", bqParamsNotEnabled).Return(rmocks.SpendSummaryReal, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, "1", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCanEnable, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, "2", mock.AnythingOfType("time.Time")).Return(costExplorerRecommendationsCantEnable, nil)
			},
			want:    nil,
			wantErr: nil,
		},
		{
			name: "fetching recommendations failed",
			args: args{
				ctx:        &ctx,
				configInfo: customerAttributesNotEnabled,
				timeParams: timeParams,
			},
			on: func(f *fields) {
				f.bigQueryService.On("CheckActiveBillingTableExists", testutils.ContextBackgroundMock, customerID2).Return(nil)
				f.bigQueryService.On("GetPayerSpendSummary", bqParamsNotEnabled).Return(rmocks.SpendSummaryReal, nil)
				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", ctx, customerID2).Return(&cacheWithSavings, nil)
				f.bigQueryService.On("GetCustomerCredits", testutils.ContextBackgroundMock, customerID2, mock.AnythingOfType("time.Time")).
					Return(bq.CreditsResult{
						Credits: map[string]float64{},
						Err:     nil,
					})
				f.recommendationService.On("FetchComputeRecommendations", ctx, "1", mock.AnythingOfType("time.Time")).Return(nil, errors.New("no billing table"))
				f.loggerProvider.On("Errorf", "error getting artifact recommendations for customer: %v with payer account: %v, with error: %v", customerAttributesNotEnabled.CustomerID, "1", "no billing table")
			},
			assert: func(t *testing.T, f *fields) {
				f.recommendationService.AssertNumberOfCalls(t, "FetchComputeRecommendations", 1)
			},
			want: &rmocks.DedicatedTestCacheNotEnabledRecommendationsFetchFailed,
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

			s := &DedicatedPayerService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},

				bigQueryService:        &f.bigQueryService,
				recommendationsService: &f.recommendationService,
				Connection:             conn,
				IntegrationsDAL:        &f.integrationsDAL,
			}

			if tt.on != nil {
				tt.on(f)
			}

			got, err := s.GetCache(*tt.args.ctx, tt.args.configInfo, tt.args.timeParams)

			if err != nil {
				expectedError := tt.wantErr
				if err.Error() != expectedError.Error() {
					t.Errorf("RunCacheForSingleCustomer() error = %v, wantErr %v", err, &expectedError)
					return
				}
			}

			assert.Equalf(t, tt.want, got, "GetCache() got = %v, want %v", got, tt.want)

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestDedicatedPayerService_shouldRunDailyCache(t *testing.T) {
	now := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	yearAgo := time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, now.Location())

	tests := []struct {
		name  string
		want  bool
		cache *fspkg.FlexsaveConfiguration
	}{
		{
			name: "should not run daily if not enabled",
			want: false,
			cache: &fspkg.FlexsaveConfiguration{
				AWS: fspkg.FlexsaveSavings{},
			},
		},
		{
			name: "should not run daily if not enabled long time ago",
			want: false,
			cache: &fspkg.FlexsaveConfiguration{
				AWS: fspkg.FlexsaveSavings{
					TimeEnabled: &yearAgo,
				},
			},
		},

		{
			name: "should run daily if enabled now",
			want: true,
			cache: &fspkg.FlexsaveConfiguration{
				AWS: fspkg.FlexsaveSavings{
					TimeEnabled: &firstDay,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRunDailyCache(tt.cache, now)

			if got != tt.want {
				t.Errorf("shouldRunDailyCache() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDedicatedPayerService_createSavingsSummary(t *testing.T) {
	type fields struct {
		loggerProvider        loggerMocks.ILogger
		recommendationService recMocks.Recommendations
		mpaDAL                dal_mocks.MasterPayerAccounts
	}

	ctx := context.Background()

	payerOne := "payer-one"
	payerTwo := "payer-two"
	payerThree := "payer-three"

	hrCommitmentBelowThreshold := common.MinimumHourlyCommitmentToPurchase - 0.1
	hrCommitmentAboveThreshold := common.MinimumHourlyCommitmentToPurchase + 0.1
	estimatedSavings := 0.5

	payerOneRec := flexAPIRec.RecommendationForDedicatedPayerResponse{
		HourlyCommitmentToPurchase: &hrCommitmentBelowThreshold,
		EstimatedSavingsAmount:     &estimatedSavings,
	}
	payerThreeRec := flexAPIRec.RecommendationForDedicatedPayerResponse{
		HourlyCommitmentToPurchase: &hrCommitmentAboveThreshold,
		EstimatedSavingsAmount:     &estimatedSavings,
	}
	payerTwoRec := flexAPIRec.RecommendationForDedicatedPayerResponse{
		HourlyCommitmentToPurchase: &hrCommitmentAboveThreshold,
		EstimatedSavingsAmount:     &estimatedSavings,
	}

	recOne := &payerOneRec
	recTwo := &payerTwoRec
	recThree := &payerThreeRec

	errDefault := errors.New("something went wrong")
	notFoundErr := fmt.Errorf("Not found: Table doitintl-cmp-flexsave-aws-dev:flexsave_artifact_pred.payer-one_artifact_pred was not found in location US, notFound")

	oneDollarEstimated := 1.0

	onboardingYesterday := time.Now().AddDate(0, 0, -1)
	onboardingFiveDaysAgo := time.Now().AddDate(0, 0, -5)

	type args struct {
		ctx        context.Context
		payerIDs   []string
		customerID string
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    fspkg.FlexsaveSavingsSummary
	}{
		{
			name: "single payer customer recommendation below one dollar hourly commitment",
			on: func(f *fields) {
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(recOne, nil)
			},
			args: args{
				ctx:        ctx,
				payerIDs:   []string{payerOne},
				customerID: "AZXIU",
			},
			want: fspkg.FlexsaveSavingsSummary{
				CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
				NextMonth: &fspkg.FlexsaveMonthSummary{
					Savings:          0.0,
					HourlyCommitment: &hrCommitmentBelowThreshold,
				},
			},
		},
		{
			name: "multiple payers for customer recommendation",
			on: func(f *fields) {
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(recOne, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerTwo, mock.AnythingOfType("time.Time")).Return(recTwo, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(recOne, nil)
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerThree, mock.AnythingOfType("time.Time")).Return(recThree, nil)
			},
			args: args{
				ctx:        ctx,
				payerIDs:   []string{payerOne, payerTwo, payerThree},
				customerID: "AZXIU",
			},
			want: fspkg.FlexsaveSavingsSummary{
				CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
				NextMonth: &fspkg.FlexsaveMonthSummary{
					Savings:          oneDollarEstimated,
					HourlyCommitment: &hrCommitmentAboveThreshold,
				},
			},
		},
		{
			name: "failed to get payer recommendation",
			on: func(f *fields) {
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(nil, errDefault)
				f.loggerProvider.On("Errorf", "error getting artifact recommendations for customer: %v with payer account: %v, with error: %v", "AZXIU", payerOne, errDefault.Error())
			},
			args: args{
				ctx:        ctx,
				payerIDs:   []string{payerOne},
				customerID: "AZXIU",
			},
			want: fspkg.FlexsaveSavingsSummary{
				CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
				NextMonth:    &fspkg.FlexsaveMonthSummary{},
			},
			wantErr: errDefault,
		},
		{
			name: "failed to get table for payer recommendation outside acceptable period",
			on: func(f *fields) {
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(nil, notFoundErr)

				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerOne).Return(&domain.MasterPayerAccount{
					OnboardingDate: &onboardingFiveDaysAgo,
				}, nil)

				f.loggerProvider.On("Errorf",
					"table not found, customer %s, payer account: %s, onboarding date: %v, err: [%v]",
					"AZXIU",
					payerOne,
					mock.MatchedBy(func(arg *time.Time) bool {
						return arg.Format(times.YearMonthDayLayout) == onboardingFiveDaysAgo.Format(times.YearMonthDayLayout)
					}),
					notFoundErr,
				)
			},
			args: args{
				ctx:        ctx,
				payerIDs:   []string{payerOne},
				customerID: "AZXIU",
			},
			want: fspkg.FlexsaveSavingsSummary{
				CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
				NextMonth:    &fspkg.FlexsaveMonthSummary{},
			},
			wantErr: notFoundErr,
		},
		{
			name: "failed to find table for payer recommendation within acceptable period",
			on: func(f *fields) {
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(nil, notFoundErr)

				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerOne).Return(&domain.MasterPayerAccount{
					OnboardingDate: &onboardingYesterday,
				}, nil)

				f.loggerProvider.On("Warningf",
					"table not found, customer %s, payer account: %s, onboarding date: %v, err: [%v]",
					"AZXIU",
					payerOne,
					mock.MatchedBy(func(arg *time.Time) bool {
						return arg.Format(times.YearMonthDayLayout) == onboardingYesterday.Format(times.YearMonthDayLayout)
					}),
					notFoundErr,
				)
			},
			args: args{
				ctx:        ctx,
				payerIDs:   []string{payerOne},
				customerID: "AZXIU",
			},
			want: fspkg.FlexsaveSavingsSummary{
				CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
				NextMonth:    &fspkg.FlexsaveMonthSummary{},
			},
			wantErr: notFoundErr,
		},
		{
			name: "failed to get mpa details",
			on: func(f *fields) {
				f.recommendationService.On("FetchComputeRecommendations", ctx, payerOne, mock.AnythingOfType("time.Time")).Return(nil, notFoundErr)
				f.mpaDAL.On("GetMasterPayerAccount", ctx, payerOne).Return(nil, errDefault)
			},
			args: args{
				ctx:        ctx,
				payerIDs:   []string{payerOne},
				customerID: "AZXIU",
			},
			want: fspkg.FlexsaveSavingsSummary{
				CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{},
				NextMonth:    &fspkg.FlexsaveMonthSummary{},
			},
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &DedicatedPayerService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				recommendationsService: &fields.recommendationService,
				mpaDAL:                 &fields.mpaDAL,
			}

			log := s.loggerProvider(context.Background())

			got, err := s.createSavingsSummary(tt.args.ctx, tt.args.payerIDs, tt.args.customerID, log)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DedicatedPayerService.getSavingsSummary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDaysWhenDailyIsApplicable(t *testing.T) {
	tests := []struct {
		name           string
		timeEnabled    time.Time
		dailyCutOffDay int
		wantStart      time.Time
		wantEnd        time.Time
	}{
		{
			name:           "start of the month",
			timeEnabled:    time.Date(2023, time.May, 1, 0, 0, 0, 0, time.UTC),
			dailyCutOffDay: 5,
			wantStart:      time.Date(2023, time.May, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:        time.Date(2023, time.June, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			name:           "middle of the month",
			timeEnabled:    time.Date(2023, time.May, 15, 0, 0, 0, 0, time.UTC),
			dailyCutOffDay: 10,
			wantStart:      time.Date(2023, time.May, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:        time.Date(2023, time.June, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			name:           "end of the year",
			timeEnabled:    time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC),
			dailyCutOffDay: 2,
			wantStart:      time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC),
			wantEnd:        time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := getDaysWhenDailyIsApplicable(tt.timeEnabled, tt.dailyCutOffDay)
			if !start.Equal(tt.wantStart) {
				t.Errorf("getDaysWhenDailyIsApplicable() start = %v, wantStart = %v", start, tt.wantStart)
			}

			if !end.Equal(tt.wantEnd) {
				t.Errorf("getDaysWhenDailyIsApplicable() end = %v, wantEnd = %v", end, tt.wantEnd)
			}
		})
	}
}

func TestHasGoodSpend(t *testing.T) {
	cases := []struct {
		totalWaste, timeframeHours, averageSavingsRate float64
		want                                           bool
	}{
		{totalWaste: -0.5, timeframeHours: 30.0, averageSavingsRate: 0.2, want: true},  // Case 1: Given values are within the required limits
		{totalWaste: -0.1, timeframeHours: 7.0, averageSavingsRate: 0.15, want: true},  // Case 2: Adjusted to make totalWaste less than timeframeHours*0.5*0.05
		{totalWaste: -18, timeframeHours: 30.0, averageSavingsRate: 0.2, want: false},  // Case 3: Total waste is not higher than required limit for 30 days
		{totalWaste: -4.2, timeframeHours: 7.0, averageSavingsRate: 0.2, want: false},  // Case 4: Total waste is not higher than required limit for 7 days
		{totalWaste: -0.5, timeframeHours: 7.0, averageSavingsRate: 0.05, want: false}, // Case 5: Average savings rate is below the required limit
	}

	for _, c := range cases {
		got := checkHasGoodSpend(c.totalWaste, c.timeframeHours, c.averageSavingsRate)
		if got != c.want {
			t.Errorf("checkHasGoodSpend(%v, %v, %v) == %v, want %v", c.totalWaste, c.timeframeHours, c.averageSavingsRate, got, c.want)
		}
	}
}

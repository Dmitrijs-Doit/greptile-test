package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery/mocks"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/executor"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func TestBigQueryService_GenerateStorageRecommendation(t *testing.T) {
	var (
		mockTime = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)

		ctx           = context.Background()
		mockStorageTB = 20.0
		mockCost      = 150.0
		discount      = 5.0

		mockTotalDaysAgo = 5.0

		mockScanPricePerPeriod = []bqmodels.ScanPricePerPeriod{
			{
				TotalUpTo30DaysAgo: bigquery.NullFloat64{Float64: mockTotalDaysAgo},
				TotalUpTo7DaysAgo:  bigquery.NullFloat64{Float64: mockTotalDaysAgo},
				TotalUpTo1DayAgo:   bigquery.NullFloat64{Float64: mockTotalDaysAgo},
			},
		}

		mockStorageRec = bqmodels.StorageRecommendationsResult{
			ProjectID:        mockProjectID,
			DatasetID:        mockDatasetID,
			TableID:          mockTableID,
			TableIDBaseName:  mockTableIDBase,
			TableCreateDate:  mockTime,
			StorageSizeTB:    bigquery.NullFloat64{Float64: mockStorageTB, Valid: true},
			Cost:             bigquery.NullFloat64{Float64: mockCost, Valid: true},
			TotalStorageCost: bigquery.NullFloat64{Float64: mockCost, Valid: true},
		}

		mockReplaceMents = domain.Replacements{
			StartDate: mockTime.AddDate(0, 0, -30).Format(times.YearMonthDayLayout),
		}

		periodTotalPriceWithoutStorage = domain.PeriodTotalPrice{
			bqmodels.TimeRangeMonth: {
				TotalScanPrice: mockScanPricePerPeriod[0].TotalUpTo30DaysAgo.Float64 * discount * executor.PricePerTBScan,
			},
			bqmodels.TimeRangeWeek: {
				TotalScanPrice: mockScanPricePerPeriod[0].TotalUpTo7DaysAgo.Float64 * discount * executor.PricePerTBScan,
			},
			bqmodels.TimeRangeDay: {
				TotalScanPrice: mockScanPricePerPeriod[0].TotalUpTo1DayAgo.Float64 * discount * executor.PricePerTBScan,
			},
		}
	)

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	assert.NoError(t, err)

	type fields struct {
		loggerProvider loggerMocks.ILogger
		dalBQ          mocks.Bigquery
	}

	type args struct {
		discount          float64
		now               time.Time
		hasTableDiscovery bool
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		want    domain.PeriodTotalPrice
		want1   dal.RecommendationSummary
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				discount:          discount,
				now:               mockTime,
				hasTableDiscovery: true,
			},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.AnythingOfType("map[string]string"))

				f.dalBQ.On("RunTotalScanPricePerPeriod", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(mockScanPricePerPeriod, nil)

				f.dalBQ.On("RunStorageRecommendationsQuery", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return([]bqmodels.StorageRecommendationsResult{mockStorageRec}, nil)
			},
			want: domain.PeriodTotalPrice{
				bqmodels.TimeRangeMonth: {
					TotalScanPrice:    mockScanPricePerPeriod[0].TotalUpTo30DaysAgo.Float64 * discount * executor.PricePerTBScan,
					TotalStoragePrice: 750,
					TotalPrice:        906.25,
				},
				bqmodels.TimeRangeWeek: {
					TotalScanPrice:    mockScanPricePerPeriod[0].TotalUpTo7DaysAgo.Float64 * discount * executor.PricePerTBScan,
					TotalStoragePrice: 175,
					TotalPrice:        331.25,
				},
				bqmodels.TimeRangeDay: {
					TotalScanPrice:    mockScanPricePerPeriod[0].TotalUpTo1DayAgo.Float64 * discount * executor.PricePerTBScan,
					TotalStoragePrice: 25,
					TotalPrice:        181.25,
				},
			},
		},
		{
			name: "should do early period pricing return when no table discovery is present",
			args: args{
				now:      mockTime,
				discount: discount,
			},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.AnythingOfType("map[string]string"))

				f.dalBQ.On("RunTotalScanPricePerPeriod", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(mockScanPricePerPeriod, nil)

				f.loggerProvider.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
			},
			want: periodTotalPriceWithoutStorage,
		},
		{
			name: "failed to get period total prices",
			args: args{
				discount:          discount,
				now:               mockTime,
				hasTableDiscovery: true,
			},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.AnythingOfType("map[string]string"))

				f.dalBQ.On("RunTotalScanPricePerPeriod", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(nil, someErr)
			},
			wantErr: someErr,
		},
		{
			name: "nil period total prices value received",
			args: args{
				discount:          discount,
				now:               mockTime,
				hasTableDiscovery: true,
			},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.AnythingOfType("map[string]string"))

				f.dalBQ.On("RunTotalScanPricePerPeriod", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(nil, nil)
			},
			wantErr: errScanPriceNotFound,
		},
		{
			name: "failed to get storage recs",
			args: args{
				discount:          discount,
				now:               mockTime,
				hasTableDiscovery: true,
			},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.AnythingOfType("map[string]string"))

				f.dalBQ.On("RunTotalScanPricePerPeriod", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(mockScanPricePerPeriod, nil)

				f.dalBQ.On("RunStorageRecommendationsQuery", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return([]bqmodels.StorageRecommendationsResult{mockStorageRec}, someErr)

				f.loggerProvider.On("Error", mock.AnythingOfType("string"))
			},
			want: periodTotalPriceWithoutStorage,
		},
		{
			name: "storage recs nil",
			args: args{
				discount:          discount,
				now:               mockTime,
				hasTableDiscovery: true,
			},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.AnythingOfType("map[string]string"))

				f.dalBQ.On("RunTotalScanPricePerPeriod", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(mockScanPricePerPeriod, nil)

				f.dalBQ.On("RunStorageRecommendationsQuery", ctx, testBQClient, mockReplaceMents, mock.AnythingOfType("time.Time")).
					Return(nil, nil)

				f.loggerProvider.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
			},
			want: periodTotalPriceWithoutStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &BigQueryService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				conn:  new(connection.Connection),
				bqDAL: &fields.dalBQ,
			}

			got, _, err := s.GenerateStorageRecommendation(ctx, "test-customer-id", testBQClient, tt.args.discount, domain.Replacements{}, tt.args.now, tt.args.hasTableDiscovery)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestBigQueryService_GetCustomerDiscounts(t *testing.T) {
	var (
		ctx = context.Background()

		mockDiscounts = []bqmodels.DiscountsAllCustomersResult{
			{
				CustomerID: "customerID-1",
				Discount:   .75,
			},
			{
				CustomerID: "customerID-2",
				Discount:   .85,
			},
		}

		dalError = errors.New("segmentation fault. core dumped")
	)

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	assert.NoError(t, err)

	type fields struct {
		loggerProvider loggerMocks.ILogger
		dalBQ          mocks.Bigquery
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    map[string]float64
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.dalBQ.On("RunDiscountsAllCustomersQuery", ctx, bqmodels.DiscountsAllCustomers, testBQClient).
					Return(mockDiscounts, nil)

			},
			want: map[string]float64{
				"customerID-1": 0.75,
				"customerID-2": 0.85,
			},
		},
		{
			name: "query fails",
			on: func(f *fields) {
				f.dalBQ.On("RunDiscountsAllCustomersQuery", ctx, bqmodels.DiscountsAllCustomers, testBQClient).
					Return(nil, dalError)

			},
			wantErr: dalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &BigQueryService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				conn:  new(connection.Connection),
				bqDAL: &fields.dalBQ,
			}

			got, err := s.GetCustomerDiscounts(ctx, testBQClient)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.EqualValues(t, tt.want, got)
		})
	}
}

func TestBigQueryService_GetBillingProjectsWithEditions(t *testing.T) {
	var (
		ctx = context.Background()

		mockReservations = []bqmodels.BillingProjectsWithReservationsResult{
			{
				CustomerID: "customerID-1",
				Project:    "project-1",
				Location:   "location-1",
			},
			{
				CustomerID: "customerID-2",
				Project:    "project-2",
				Location:   "location-2",
			},
			{
				CustomerID: "customerID-2",
				Project:    "project-3",
				Location:   "location-3",
			},
		}

		dalError = errors.New("segmentation fault. core dumped")
	)

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	assert.NoError(t, err)

	type fields struct {
		loggerProvider loggerMocks.ILogger
		dalBQ          mocks.Bigquery
	}

	tests := []struct {
		name    string
		on      func(*fields)
		want    map[string][]domain.BillingProjectWithReservation
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.dalBQ.On("RunBillingProjectsWithEditionsQuery", ctx, bqmodels.BillingProjectsWithEditionsQuery, testBQClient).
					Return(mockReservations, nil)

			},
			want: map[string][]domain.BillingProjectWithReservation{
				"customerID-1": {
					{
						Project:  "project-1",
						Location: "location-1",
					},
				},
				"customerID-2": {
					{
						Project:  "project-2",
						Location: "location-2",
					},
					{
						Project:  "project-3",
						Location: "location-3",
					},
				},
			},
		},
		{
			name: "query fails",
			on: func(f *fields) {
				f.dalBQ.On("RunBillingProjectsWithEditionsQuery", ctx, bqmodels.BillingProjectsWithEditionsQuery, testBQClient).
					Return(nil, dalError)

			},
			wantErr: dalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &BigQueryService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				conn:  new(connection.Connection),
				bqDAL: &fields.dalBQ,
			}

			got, err := s.GetBillingProjectsWithEditions(ctx, testBQClient)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}

			assert.EqualValues(t, tt.want, got)
		})
	}
}

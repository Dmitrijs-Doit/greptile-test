package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	optimizerMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore/mocks"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	serviceMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/mocks"
	pricebookDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
	pricebookMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGenerateSwitchToEditionsRecommendation(t *testing.T) {
	var (
		ctx             = context.Background()
		mockCustomerID  = "mock-customer-id"
		mockProjectID   = "mock-project-id"
		mockLocation    = "mock-location"
		mockTimeNow     = time.Date(2024, 06, 17, 0, 0, 0, 0, time.UTC)
		mockTimeNowFunc = func() time.Time { return mockTimeNow }

		testLocation1 = "test-location-1"
		testProject1  = "test-project-1"
		testLocation2 = "test-location-2"
		testProject2  = "test-project-2"
	)

	type args struct {
		customerID             string
		reservationAssignments []domain.ReservationAssignment
		bq                     *bigquery.Client
	}

	type fields struct {
		loggerProvider loggerMocks.ILogger
		serviceBQ      serviceMocks.Bigquery
		dalFS          optimizerMocks.Optimizer
		pricebook      pricebookMocks.Pricebook
		insights       dalMocks.Insights
	}

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	assert.NoError(t, err)

	aggregatedJobStatisticsErr := errors.New("error 1138")

	aggregatedJobsStatistics := []bqmodels.AggregatedJobStatistic{
		{
			Location:         testLocation1,
			ProjectID:        testProject1,
			TotalSlotsMS:     3600,
			TotalBilledBytes: 1024 * 1024,
		},
		{
			Location:         testLocation1,
			ProjectID:        testProject2,
			TotalSlotsMS:     7200,
			TotalBilledBytes: 2024 * 1024,
		},
		{
			Location:         testLocation2,
			ProjectID:        testProject1,
			TotalSlotsMS:     36000,
			TotalBilledBytes: 10240 * 1024,
		},
		{
			Location:         testLocation2,
			ProjectID:        testProject2,
			TotalSlotsMS:     72000,
			TotalBilledBytes: 20240 * 1024,
		},
	}

	getPricebooksErr := errors.New("error 31415")

	editionsPricebooks := pricebookDomain.PriceBooksByEdition{
		pricebookDomain.Standard: &pricebookDomain.PricebookDocument{
			string(pricebookDomain.OnDemand): map[string]float64{
				testLocation1: 12,
				testLocation2: 24,
			},
		},
		pricebookDomain.Enterprise: &pricebookDomain.PricebookDocument{
			string(pricebookDomain.OnDemand): map[string]float64{
				testLocation1: 20,
				testLocation2: 40,
			},
			string(pricebookDomain.Commit1Yr): map[string]float64{
				testLocation1: 15,
				testLocation2: 30,
			},
			string(pricebookDomain.Commit3Yr): map[string]float64{
				testLocation1: 10,
				testLocation2: 20,
			},
		},
	}

	getOnDemandPricebookErr := errors.New("error 14142")

	onDemandPricebook := map[string]float64{
		testLocation1: 10,
		testLocation2: 20,
	}

	postInsightResultsErr := errors.New("error 101")

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "GetAggregatedJobStatistics fails",
			args: args{
				customerID: mockCustomerID,
				bq:         testBQClient,
			},
			on: func(f *fields) {
				f.serviceBQ.On("GetAggregatedJobStatistics", ctx, testBQClient, mockProjectID, mockLocation).
					Return(nil, aggregatedJobStatisticsErr).Once()

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil).Once()
			},
			wantErr: wrapOperationError("GetAggregatedJobStatistics", mockCustomerID, aggregatedJobStatisticsErr),
		},
		{
			name: "GetPricebooks fails",
			args: args{
				customerID: mockCustomerID,
				bq:         testBQClient,
			},
			on: func(f *fields) {
				f.serviceBQ.On("GetAggregatedJobStatistics", ctx, testBQClient, mockProjectID, mockLocation).
					Return(aggregatedJobsStatistics, nil).Once()

				f.pricebook.On("GetPricebooks", ctx).
					Return(nil, getPricebooksErr).Once()

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil).Once()
			},
			wantErr: wrapOperationError("GetPricebooks", mockCustomerID, getPricebooksErr),
		},
		{
			name: "GetOnDemandPricebook fails",
			args: args{
				customerID: mockCustomerID,
				bq:         testBQClient,
			},
			on: func(f *fields) {
				f.serviceBQ.On("GetAggregatedJobStatistics", ctx, testBQClient, mockProjectID, mockLocation).
					Return(aggregatedJobsStatistics, nil).Once()

				f.pricebook.On("GetPricebooks", ctx).
					Return(editionsPricebooks, nil).Once()

				f.pricebook.On("GetOnDemandPricebook", ctx).
					Return(nil, getOnDemandPricebookErr).Once()

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil).Once()
			},
			wantErr: wrapOperationError("GetOnDemandPricebook", mockCustomerID, getOnDemandPricebookErr),
		},
		{
			name: "PostInsightResults fails",
			args: args{
				customerID: mockCustomerID,
				bq:         testBQClient,
			},
			on: func(f *fields) {
				f.serviceBQ.On("GetAggregatedJobStatistics", ctx, testBQClient, mockProjectID, mockLocation).
					Return(aggregatedJobsStatistics, nil).Once()

				f.pricebook.On("GetPricebooks", ctx).
					Return(editionsPricebooks, nil).Once()

				f.pricebook.On("GetOnDemandPricebook", ctx).
					Return(onDemandPricebook, nil).Once()

				f.insights.On("PostInsightResults", ctx, mock.AnythingOfType("[]sdk.InsightResponse")).
					Return(postInsightResultsErr).Once()

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil).Once()
			},
			wantErr: wrapOperationError("PostInsightResults", mockCustomerID, postInsightResultsErr),
		},
		{
			name: "Happy path",
			args: args{
				customerID: mockCustomerID,
				bq:         testBQClient,
			},
			on: func(f *fields) {
				f.serviceBQ.On("GetAggregatedJobStatistics", ctx, testBQClient, mockProjectID, mockLocation).
					Return(aggregatedJobsStatistics, nil).Once()

				f.pricebook.On("GetPricebooks", ctx).
					Return(editionsPricebooks, nil).Once()

				f.pricebook.On("GetOnDemandPricebook", ctx).
					Return(onDemandPricebook, nil).Once()

				f.insights.On("PostInsightResults", ctx, mock.AnythingOfType("[]sdk.InsightResponse")).
					Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &OptimizerService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				conn:        new(connection.Connection),
				serviceBQ:   &fields.serviceBQ,
				dalFS:       &fields.dalFS,
				pricebook:   &fields.pricebook,
				insights:    &fields.insights,
				timeNowFunc: mockTimeNowFunc,
			}

			err := s.GenerateSwitchToEditionsRecommendation(ctx, tt.args.customerID, tt.args.reservationAssignments, tt.args.bq, mockProjectID, mockLocation)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func TestGetProjectsByEdition(t *testing.T) {
	var (
		testProject1 = "test-project-1"
		testProject2 = "test-project-2"
		testProject3 = "test-project-3"
		testProject4 = "test-project-4"
		testProject5 = "test-project-5"
		testProject6 = "test-project-6"
	)

	tests := []struct {
		name                   string
		reservationAssignments []domain.ReservationAssignment
		want                   map[reservationpb.Edition][]string
	}{
		{
			name: "a couple of reservation assignments",
			reservationAssignments: []domain.ReservationAssignment{
				{
					Reservation: domain.Reservation{
						Edition: reservationpb.Edition_ENTERPRISE,
					},
					ProjectsList: []string{testProject1, testProject2},
				},
				{
					Reservation: domain.Reservation{
						Edition: reservationpb.Edition_STANDARD,
					},
					ProjectsList: []string{testProject3, testProject4},
				},
				{
					Reservation: domain.Reservation{
						Edition: reservationpb.Edition_STANDARD,
					},
					ProjectsList: []string{testProject5, testProject6},
				},
			},
			want: map[reservationpb.Edition][]string{
				reservationpb.Edition_STANDARD:   {testProject3, testProject4, testProject5, testProject6},
				reservationpb.Edition_ENTERPRISE: {testProject1, testProject2},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &OptimizerService{}

			got := s.getProjectsByEdition(tt.reservationAssignments)
			assert.Equal(t, tt.want, got)
		})
	}
}

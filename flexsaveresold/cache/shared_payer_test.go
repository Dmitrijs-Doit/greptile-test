package cache

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/googleapi"

	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/accounts"
	cloudHealthMock "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal/mocks"
	bigQueryCache "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	bigqueryMock "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
)

type inputOutput struct {
	currentMonth     string
	predictedSavings float64
	summary          *fspkg.FlexsaveSavingsSummary
}

var inputs = []inputOutput{
	{
		"3_2022",
		6234.54,
		&fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month: "3_2022",
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings: 6234.54,
			},
		},
	},
	{
		"9_2018",
		5276,
		&fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month: "9_2018",
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{
				Savings: 5276,
			},
		},
	},
	{
		"1_2023",
		0,
		&fspkg.FlexsaveSavingsSummary{
			CurrentMonth: &fspkg.FlexsaveCurrentMonthSummary{
				Month: "1_2023",
			},
			NextMonth: &fspkg.FlexsaveMonthSummary{},
		},
	},
}

func TestMakeSavingsSummary(t *testing.T) {
	for _, input := range inputs {
		res := makeSavingsSummary(input.currentMonth, input.predictedSavings)
		assert.Equal(t, input.summary, res)
	}
}

func TestService_hadNoSpendInOverThirtyDaysFromActivationOnSharedPayers(t *testing.T) {
	mockService := &accounts.MockService{}
	err := errors.New("oh no")
	ctx := context.Background()
	mockBQ := &bigqueryMock.BigQueryServiceInterface{}

	type args struct {
		ctx      context.Context
		assetIDs []string
	}

	tests := []struct {
		name string
		args args
		want bool
		on   func(s *accounts.MockService)
		err  error
	}{
		{
			name: "returns false if less than threshold (30 days)",
			args: args{
				ctx:      ctx,
				assetIDs: []string{"asset-abc"},
			},
			on: func(s *accounts.MockService) {
				s.On("GetOldestJoinTimestampAge", ctx, []string{"amazon-web-services-asset-abc"}, mock.Anything).Return(10, nil).Once()
			},
			err:  nil,
			want: false,
		},

		{
			name: "returns true if more than threshold (30 days)",
			args: args{
				ctx:      ctx,
				assetIDs: []string{"asset-abc"},
			},
			on: func(s *accounts.MockService) {
				s.On("GetOldestJoinTimestampAge", ctx, []string{"amazon-web-services-asset-abc"}, mock.Anything).Return(40, nil).Once()
			},
			err:  nil,
			want: true,
		},

		{
			name: "returns err if service does",
			args: args{
				ctx:      ctx,
				assetIDs: []string{"asset-abc"},
			},
			on: func(s *accounts.MockService) {
				s.On("GetOldestJoinTimestampAge", ctx, []string{"amazon-web-services-asset-abc"}, mock.Anything).Return(-1, err).Once()
			},
			err:  err,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SharedPayerService{
				awsAccountsService: mockService,
				bigQueryService:    mockBQ,
			}

			if tt.on != nil {
				tt.on(mockService)
			}

			got, err := s.getHasBeenLongSinceOnboarded(tt.args.ctx, tt.args.assetIDs)
			assert.Equal(t, tt.err, err)
			assert.Equalf(t, tt.want, got, "getHasBeenLongSinceOnboarded(%v, %v)", tt.args.ctx, tt.args.assetIDs)
		})
	}
}

func TestService_checkHasCloudHealthTable(t *testing.T) {
	mockCloudHealthDAL := &cloudHealthMock.CloudHealthDAL{}
	mockBQ := &bigqueryMock.BigQueryServiceInterface{}
	err := errors.New("oh no")
	ctx := context.Background()

	tests := []struct {
		name string
		want bool
		err  error
		on   func(s *cloudHealthMock.CloudHealthDAL, bq *bigqueryMock.BigQueryServiceInterface)
	}{
		{
			name: "service returns error if cloudhealth dal does",
			want: false,
			err:  err,
			on: func(s *cloudHealthMock.CloudHealthDAL, bq *bigqueryMock.BigQueryServiceInterface) {
				s.On("GetCustomerCloudHealthID", mock.Anything, mock.Anything).Return("", err).Once()
			},
		},

		{
			name: "service returns false if bigquery dal returns no table",
			want: false,
			err:  nil,
			on: func(s *cloudHealthMock.CloudHealthDAL, bq *bigqueryMock.BigQueryServiceInterface) {
				s.On("GetCustomerCloudHealthID", mock.Anything, mock.Anything).Return("ch-id-1", nil).Once()
				bq.On("CheckActiveBillingTableExists", mock.Anything, "ch-id-1").Return(bigQueryCache.ErrNoActiveTable).Once()
			},
		},

		{
			name: "service returns error if bigquery dal does",
			want: false,
			err:  err,
			on: func(s *cloudHealthMock.CloudHealthDAL, bq *bigqueryMock.BigQueryServiceInterface) {
				s.On("GetCustomerCloudHealthID", mock.Anything, mock.Anything).Return("ch-id-1", nil).Once()
				bq.On("CheckActiveBillingTableExists", mock.Anything, "ch-id-1").Return(err).Once()
			},
		},

		{
			name: "service returns true if all if fine",
			want: true,
			err:  nil,
			on: func(s *cloudHealthMock.CloudHealthDAL, bq *bigqueryMock.BigQueryServiceInterface) {
				s.On("GetCustomerCloudHealthID", mock.Anything, mock.Anything).Return("ch-id-1", nil).Once()
				bq.On("CheckActiveBillingTableExists", mock.Anything, "ch-id-1").Return(nil).Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SharedPayerService{
				cloudHealthDAL:  mockCloudHealthDAL,
				bigQueryService: mockBQ,
			}

			if tt.on != nil {
				tt.on(mockCloudHealthDAL, mockBQ)
			}

			got, err := s.checkHasCloudHealthTable(ctx, &firestore.DocumentRef{})
			assert.Equalf(t, tt.err, err, "checkHasCloudHealthTable(%s)", tt.name)

			assert.Equalf(t, tt.want, got, "checkHasCloudHealthTable(%s)", tt.name)
		})
	}
}

func Test_datasetNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "google API error with 404 status",
			err:  &googleapi.Error{Code: 404},
			want: true,
		},
		{
			name: "google API error with non-404 status",
			err:  &googleapi.Error{Code: 500},
			want: false,
		},
		{
			name: "non-google API error",
			err:  errors.New("some other error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, datasetNotFound(tt.err), "datasetNotFound(%v) returned unexpected result", tt.err)
		})
	}
}

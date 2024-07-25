package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/stretchr/testify/mock"

	sharedDalMocks "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/dal/mocks"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestRollback(t *testing.T) {
	type fields struct {
		Logger           *loggerMocks.ILogger
		Metadata         *mocks.Metadata
		Config           *mocks.PipelineConfig
		Table            *mocks.Table
		Bucket           *mocks.Bucket
		CustomerBQClient *mocks.ExternalBigQueryClient
		TQuery           *mocks.TableQuery
		ImportStatus     *sharedDalMocks.BillingImportStatus
	}

	testError := errors.New("test error")
	ctx := context.Background()

	testData := []struct {
		name          string
		requestParams *dataStructures.OnboardingRequestBody
		on            func(*fields)
		wantErr       bool
	}{
		{
			name: "Internal task metadata error",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, testError).Once()
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepInternalTaskMetadata, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepInternalTaskMetadata).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()

			},
			wantErr: true,
		},
		{
			name: "Internal task metadata already exists",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(&dataStructures.InternalTaskMetadata{}, nil).Once()
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepInternalTaskMetadata, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepInternalTaskMetadata).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "External task metadata error",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, testError).Once()
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepExternalTaskMetadata, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepExternalTaskMetadata).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()

			},
			wantErr: true,
		},
		{
			name: "External task metadata exists",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(&dataStructures.ExternalTaskMetadata{}, nil).Once()
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepExternalTaskMetadata, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepExternalTaskMetadata).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "GetCustomerBQClientWithParams fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil, testError).Once()
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepCustomerBQClient, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepCustomerBQClient).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "GetTableLocation fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("", testError)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepGetTableLocation, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepGetTableLocation).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "GetCustomersTableNewestRecordTime fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("test-location", nil)
				f.TQuery.
					On("GetCustomersTableNewestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, testError)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepGetCustomersTableLatestRecordTime, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepGetCustomersTableLatestRecordTime).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "GetCustomersTableOldestRecordTime fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("test-location", nil)
				f.TQuery.
					On("GetCustomersTableNewestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil).
					On("GetCustomersTableOldestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, testError)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepGetCustomersTableOldestRecordTime, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepGetCustomersTableOldestRecordTime).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "CreateMetadataForNewBillingID fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("CreateMetadataForNewBillingID", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).
					Return(testError).Once().
					On("DeleteInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("test-location", nil)
				f.TQuery.
					On("GetCustomersTableNewestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil).
					On("GetCustomersTableOldestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepCreateMetadataForNewBillingID, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepCreateMetadataForNewBillingID).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "FindOrCreateBucket fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("CreateMetadataForNewBillingID", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).
					Return(nil).Once().
					On("DeleteInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil).Once()
				f.Bucket.
					On("FindOrCreateBucket", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Return(testError).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once().
					On("GetCustomerBQClient", ctx, mock.AnythingOfType("string")).
					Return(nil, testError).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("test-location", nil)
				f.TQuery.
					On("GetCustomersTableNewestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil).
					On("GetCustomersTableOldestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepFindOrCreateBucket, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepFindOrCreateBucket).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "CreateLocalTable fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("CreateMetadataForNewBillingID", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).
					Return(nil).Once().
					On("DeleteInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil).Once()
				f.Config.
					On("GetRegionBucket", ctx, mock.AnythingOfType("string")).
					Return("test-region-bucket", nil).Once()
				f.Bucket.
					On("FindOrCreateBucket", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Return(nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once().
					On("GetCustomerBQClient", ctx, mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("test-location", nil).
					On("CreateLocalTable", ctx, mock.AnythingOfType("string")).
					Return(testError).Once().
					On("DeleteLocalTable", ctx, mock.AnythingOfType("string")).
					Return(nil).Once()
				f.TQuery.
					On("GetCustomersTableNewestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil).
					On("GetCustomersTableOldestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepCreateLocalTable, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepCreateLocalTable).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Once()
			},
			wantErr: true,
		},
		{
			name: "notifyStarted fails",
			requestParams: &dataStructures.OnboardingRequestBody{
				BillingAccountID: "test-billing-account",
			},
			on: func(f *fields) {
				f.Metadata.
					On("GetInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("GetExternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil, nil).Once().
					On("CreateMetadataForNewBillingID", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody"), mock.AnythingOfType("string"), mock.Anything, mock.Anything).
					Return(nil).Once().
					On("DeleteInternalTaskMetadata", ctx, mock.AnythingOfType("string")).
					Return(nil).Once()
				f.ImportStatus.
					On("GetBillingImportStatus", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil, testError)
				f.Config.
					On("GetRegionBucket", ctx, mock.AnythingOfType("string")).
					Return("test-region-bucket", nil).Once()
				f.Bucket.
					On("FindOrCreateBucket", ctx, mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Return(nil).Once()
				f.CustomerBQClient.
					On("GetCustomerBQClientWithParams", ctx, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once().
					On("GetCustomerBQClient", ctx, mock.AnythingOfType("string")).
					Return(&bigquery.Client{}, nil).Once()
				f.Table.
					On("GetTableLocation", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return("test-location", nil).
					On("CreateLocalTable", ctx, mock.AnythingOfType("string")).
					Return(nil).Once().
					On("DeleteLocalTable", ctx, mock.AnythingOfType("string")).
					Return(nil).Once()
				f.TQuery.
					On("GetCustomersTableNewestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil).
					On("GetCustomersTableOldestRecordTime", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("*dataStructures.BillingTableInfo")).
					Return(time.Time{}, nil)
				f.Logger.
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("*dataStructures.OnboardingRequestBody")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepNotifyStarted, mock.AnythingOfType("*errors.errorString")).
					Once().
					On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), onboardingStepNotifyStarted).
					Once().
					On("Error", mock.AnythingOfType("*errors.errorString")).Twice()
			},
			wantErr: true,
		},
	}

	for _, tt := range testData {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:           &loggerMocks.ILogger{},
				Metadata:         &mocks.Metadata{},
				Config:           &mocks.PipelineConfig{},
				CustomerBQClient: &mocks.ExternalBigQueryClient{},
				Table:            &mocks.Table{},
				Bucket:           &mocks.Bucket{},
				TQuery:           &mocks.TableQuery{},
				ImportStatus:     &sharedDalMocks.BillingImportStatus{},
			}

			o := &Onboarding{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				metadata:         f.Metadata,
				config:           f.Config,
				table:            f.Table,
				bqTable:          f.Table,
				bucket:           f.Bucket,
				customerBQClient: f.CustomerBQClient,
				tQuery:           f.TQuery,
				importStatus:     f.ImportStatus,
			}

			if tt.on != nil {
				tt.on(f)
			}

			gotErr := o.Onboard(ctx, tt.requestParams)
			if (gotErr != nil) != tt.wantErr {
				t.Errorf("Test: %q : Got error %v, wanted err=%v", tt.name, gotErr, tt.wantErr)
			}
		})
	}
}

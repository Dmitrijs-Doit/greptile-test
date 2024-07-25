package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	reservation "cloud.google.com/go/bigquery/reservation/apiv1"
	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/option"

	bq "github.com/doitintl/bigquery"
	crmMocks "github.com/doitintl/cloudresourcemanager/mocks"
	tiersPkg "github.com/doitintl/firestore/pkg"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	optimizerMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	serviceMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/mocks"
	cloudConnectMocks "github.com/doitintl/hello/scheduled-tasks/cloudconnect/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDALMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	tiersMocks "github.com/doitintl/tiers/service/mocks"
)

func TestOptimizerService_SingleCustomerOptimizer(t *testing.T) {
	var (
		ctx                 = context.Background()
		someErr             = errors.New("some error")
		customerID          = "mock-customer-id"
		location            = "mock-location"
		projectID           = "mock-project-id"
		reservationProject  = "mock-reservation-project"
		reservationLocation = "mock-reservation-location"
		setMinDate          = time.Date(2023, 04, 12, 0, 0, 0, 0, time.UTC)
		setMaxDate          = time.Date(2023, 04, 15, 0, 0, 0, 0, time.UTC)
		mockTimeNow         = time.Date(2023, 05, 27, 0, 0, 0, 0, time.UTC)
		mockTimeNowFunc     = func() time.Time { return mockTimeNow }

		rsvClient = &reservation.Client{}

		checkDaysResult = []bqmodels.CheckCompleteDaysResult{
			{
				Min: bigquery.NullTimestamp{Timestamp: setMinDate, Valid: true},
				Max: bigquery.NullTimestamp{Timestamp: setMaxDate, Valid: true},
			},
		}

		invalidDaysResult = bqmodels.CheckCompleteDaysResult{
			Min: bigquery.NullTimestamp{Timestamp: time.Time{}, Valid: false},
			Max: bigquery.NullTimestamp{Timestamp: time.Time{}, Valid: false},
		}

		discount = 1.95

		billingProjects = []domain.BillingProjectWithReservation{
			{Project: reservationProject, Location: reservationLocation},
		}

		periodTotalPriceMapping = domain.PeriodTotalPrice{
			bqmodels.TimeRangeMonth: domain.TotalPrice{
				TotalScanPrice: 5.0,
			},
			bqmodels.TimeRangeWeek: domain.TotalPrice{
				TotalScanPrice: 5.0,
			},
			bqmodels.TimeRangeDay: domain.TotalPrice{
				TotalScanPrice: 5.0,
			},
		}

		storageRec = dal.RecommendationSummary{
			bqmodels.StorageSavings: {
				bqmodels.TimeRangeMonth: "storage-rec-data",
			},
		}

		mockPayload = domain.Payload{
			BillingProjectWithReservation: billingProjects,
			Discount:                      discount,
		}

		projectsWithReservations = []string{"project1", "project2"}

		reservationAssignments = []domain.ReservationAssignment{
			{
				Reservation: domain.Reservation{
					Name:    reservationName,
					Edition: mockEdition,
				},
				ProjectsList: []string{"project1", "project2"},
			},
		}

		projectsByEdition = map[reservationpb.Edition][]string{
			mockEdition: {"project1", "project2"},
		}

		replacements = domain.Replacements{
			ProjectID:                projectID,
			DatasetID:                bqLensDomain.DoitCmpDatasetID,
			TablesDiscoveryTable:     bqLensDomain.DoitCmpTablesTable,
			HistoricalJobs:           nil,
			ProjectsWithReservations: projectsWithReservations,
			ProjectsByEdition:        projectsByEdition,
			MinDate:                  setMinDate,
			MaxDate:                  setMaxDate,
			Location:                 location,
		}

		transformerCtx = domain.TransformerContext{
			Discount:                discount,
			TotalScanPricePerPeriod: periodTotalPriceMapping,
		}

		executorData      = dal.RecommendationSummary{}
		hasTableDiscovery = true

		customerDALGetRef = &firestore.DocumentRef{}

		getCustomerTierResponse = &tiersPkg.Tier{
			Name: string(tiersPkg.Premium),
		}

		getCustomerTierErr = errors.New("getCustomerTier error")
	)

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	assert.NoError(t, err)

	connClients := &pkg.ConnectClients{
		CRM: crmMocks.NewCloudResourceManager(t),
		BQ: &bq.Service{
			BigqueryService: testBQClient,
		},
	}

	clientOptions := []option.ClientOption{}

	type fields struct {
		loggerProvider loggerMocks.ILogger
		serviceBQ      serviceMocks.Bigquery
		dalFS          optimizerMocks.Optimizer
		cloudConnect   cloudConnectMocks.CloudConnectService
		executor       serviceMocks.Executor
		reservations   serviceMocks.Reservations
		tiers          tiersMocks.TierServiceIface
		customerDAL    customerDALMocks.Customers
	}

	type args struct {
		payload domain.Payload
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "successfully processed and saved recommendations",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&checkDaysResult[0], nil)

				f.reservations.On("GetProjectsWithReservations",
					ctx, customerID, rsvClient, connClients.CRM, billingProjects,
				).Return(projectsWithReservations, reservationAssignments)

				f.serviceBQ.On("GenerateStorageRecommendation",
					ctx, customerID, connClients.BQ.BigqueryService, discount, replacements, mockTimeNow, hasTableDiscovery,
				).Return(periodTotalPriceMapping, storageRec, nil)

				f.executor.On("Execute", ctx, connClients.BQ.BigqueryService, replacements, transformerCtx, bqmodels.QueriesPerMode, hasTableDiscovery).
					Return(executorData, nil)

				f.dalFS.On("SetRecommendationDataIncrementally", ctx, customerID, mergeMaps(executorData, storageRec)).
					Return(nil)

				f.loggerProvider.On("SetLabels", mock.Anything).Once()
			},
		},
		{
			name: "continue to process recommendations without table discovery",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(nil, someErr)

				f.loggerProvider.On("Error", wrapOperationError("GetTableDiscoveryMetadata", customerID, someErr).Error())

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&checkDaysResult[0], nil)

				f.reservations.On("GetProjectsWithReservations",
					ctx, customerID, rsvClient, connClients.CRM, billingProjects,
				).Return(projectsWithReservations, reservationAssignments)

				f.serviceBQ.On("GenerateStorageRecommendation",
					ctx, customerID, connClients.BQ.BigqueryService, discount, replacements, mockTimeNow, false,
				).Return(periodTotalPriceMapping, storageRec, nil)

				f.executor.On("Execute", ctx, connClients.BQ.BigqueryService, replacements, transformerCtx, bqmodels.QueriesPerMode, false).
					Return(executorData, nil)

				f.dalFS.On("SetRecommendationDataIncrementally", ctx, customerID, mergeMaps(executorData, storageRec)).
					Return(nil)

				f.loggerProvider.On("SetLabels", mock.Anything).Once()
			},
		},
		{
			name: "updated progress when invalid min and max date present",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&invalidDaysResult, nil)

				f.loggerProvider.On("Infof", "min and max days are not defined for customer '%s'", customerID)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
		},
		{
			name: "failed to get connection clients",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(nil, nil, someErr)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("NewGCPClients", customerID, someErr),
		},
		{
			name: "saved recommendations when executor returns errors",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&checkDaysResult[0], nil)

				f.reservations.On("GetProjectsWithReservations",
					ctx, customerID, rsvClient, connClients.CRM, billingProjects,
				).Return(projectsWithReservations, reservationAssignments)

				f.serviceBQ.On("GenerateStorageRecommendation",
					ctx, customerID, connClients.BQ.BigqueryService, discount, replacements, mockTimeNow, hasTableDiscovery,
				).Return(periodTotalPriceMapping, storageRec, nil)

				f.executor.On("Execute", ctx, connClients.BQ.BigqueryService, replacements, transformerCtx, bqmodels.QueriesPerMode, hasTableDiscovery).
					Return(executorData, []error{someErr})

				f.loggerProvider.On("Error", wrapOperationError("Execute", customerID, someErr)).Once()

				f.dalFS.On("SetRecommendationDataIncrementally", ctx, customerID, mergeMaps(executorData, storageRec)).
					Return(nil)

			},
			wantErr: nil,
		},
		{
			name: "failed to get reservation client",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, someErr)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("NewClient", customerID, someErr),
		},
		{
			name: "failed to get dataset id and location",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, someErr)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("GetDatasetLocationAndProjectID", customerID, someErr),
		},
		{
			name: "failed to get min and max dates",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&checkDaysResult[0], someErr)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("GetMinAndMaxDates", customerID, someErr),
		},
		{
			name: "failed update progress for invalid min max dates",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&invalidDaysResult, nil)

				f.loggerProvider.On("Infof", "min and max days are not defined for customer '%s'", customerID)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(someErr)

				f.loggerProvider.On("Errorf", "failed to update simulation details for customer '%s': %w", customerID, someErr)
			},
			wantErr: nil,
		},
		{
			name: "failed to retrieve storage recommendations and period total mapping",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&checkDaysResult[0], nil)

				f.reservations.On("GetProjectsWithReservations",
					ctx, customerID, rsvClient, connClients.CRM, billingProjects,
				).Return(projectsWithReservations, reservationAssignments)

				f.serviceBQ.On("GenerateStorageRecommendation",
					ctx, customerID, connClients.BQ.BigqueryService, discount, replacements, mockTimeNow, hasTableDiscovery,
				).Return(nil, nil, someErr)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("GenerateStorageRecommendation", customerID, someErr),
		},
		{
			name: "failed to save recommendations data",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(getCustomerTierResponse, nil).Once()

				f.cloudConnect.On("NewGCPClients", ctx, customerID).Return(connClients, clientOptions, nil)

				f.reservations.On("NewClient", ctx, clientOptions).Return(rsvClient, nil)

				f.serviceBQ.On("GetTableDiscoveryMetadata",
					ctx, connClients.BQ.BigqueryService,
				).Return(&bigquery.TableMetadata{}, nil)

				f.serviceBQ.On("GetDatasetLocationAndProjectID",
					ctx, connClients.BQ.BigqueryService, bqLensDomain.DoitCmpDatasetID,
				).Return(location, projectID, nil)

				f.serviceBQ.On("GetMinAndMaxDates", ctx, connClients.BQ.BigqueryService, projectID, location).
					Return(&checkDaysResult[0], nil)

				f.reservations.On("GetProjectsWithReservations",
					ctx, customerID, rsvClient, connClients.CRM, billingProjects,
				).Return(projectsWithReservations, reservationAssignments)

				f.serviceBQ.On("GenerateStorageRecommendation",
					ctx, customerID, connClients.BQ.BigqueryService, discount, replacements, mockTimeNow, hasTableDiscovery,
				).Return(periodTotalPriceMapping, storageRec, nil)

				f.executor.On("Execute", ctx, connClients.BQ.BigqueryService, replacements, transformerCtx, bqmodels.QueriesPerMode, hasTableDiscovery).
					Return(executorData, nil)

				f.loggerProvider.On("SetLabels", mock.Anything).Once()

				f.dalFS.On("SetRecommendationDataIncrementally", ctx, customerID, mergeMaps(executorData, storageRec)).
					Return(someErr)

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("SetRecommendationDataIncrementally", customerID, someErr),
		},
		{
			name: "failed to get customer tier",
			args: args{payload: mockPayload},
			on: func(f *fields) {
				f.loggerProvider.On("SetLabels", mock.Anything).Once()
				f.customerDAL.On("GetRef", ctx, customerID).Return(customerDALGetRef).Once()

				f.tiers.On("GetCustomerTier", ctx, customerDALGetRef, tiersPkg.NavigatorPackageTierType).
					Return(nil, getCustomerTierErr).Once()

				f.dalFS.On("UpdateSimulationDetails",
					ctx,
					customerID,
					map[string]interface{}{"progress": 100, "status": "END"}).
					Return(nil)
			},
			wantErr: wrapOperationError("GetCustomerTier", customerID, getCustomerTierErr),
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
				conn:         new(connection.Connection),
				serviceBQ:    &fields.serviceBQ,
				dalFS:        &fields.dalFS,
				cloudConnect: &fields.cloudConnect,
				tiers:        &fields.tiers,
				executor:     &fields.executor,
				customerDAL:  &fields.customerDAL,
				reservations: &fields.reservations,
				timeNowFunc:  mockTimeNowFunc,
			}

			err := s.SingleCustomerOptimizer(ctx, customerID, tt.args.payload)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func Test_mergeMaps(t *testing.T) {
	type args struct {
		executorData           dal.RecommendationSummary
		storageRecommendations dal.RecommendationSummary
	}
	tests := []struct {
		name string
		args args
		want dal.RecommendationSummary
	}{
		{
			name: "both maps are empty",
			args: args{
				executorData:           dal.RecommendationSummary{},
				storageRecommendations: dal.RecommendationSummary{},
			},
			want: dal.RecommendationSummary{},
		},
		{
			name: "executor data is empty",
			args: args{
				executorData:           dal.RecommendationSummary{},
				storageRecommendations: dal.RecommendationSummary{bqmodels.StorageSavings: {bqmodels.TimeRangeMonth: "result1"}},
			},
			want: dal.RecommendationSummary{bqmodels.StorageSavings: {bqmodels.TimeRangeMonth: "result1"}},
		},
		{
			name: "storageRecommendations is empty",
			args: args{
				executorData:           dal.RecommendationSummary{bqmodels.TableScanPrice: {bqmodels.TimeRangeMonth: "result1"}},
				storageRecommendations: dal.RecommendationSummary{},
			},
			want: dal.RecommendationSummary{bqmodels.TableScanPrice: {bqmodels.TimeRangeMonth: "result1"}},
		},
		{
			name: "values for executor and storageRecommendations are present",
			args: args{
				executorData:           dal.RecommendationSummary{bqmodels.TableScanPrice: {bqmodels.TimeRangeMonth: "result1"}},
				storageRecommendations: dal.RecommendationSummary{bqmodels.StorageSavings: {bqmodels.TimeRangeMonth: "result2"}},
			},
			want: dal.RecommendationSummary{
				bqmodels.TableScanPrice: {bqmodels.TimeRangeMonth: "result1"},
				bqmodels.StorageSavings: {bqmodels.TimeRangeMonth: "result2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, mergeMaps(tt.args.executorData, tt.args.storageRecommendations), "mergeMaps(%v, %v)", tt.args.executorData, tt.args.storageRecommendations)
		})
	}
}

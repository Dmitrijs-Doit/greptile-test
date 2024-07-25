package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"gotest.tools/assert"

	bq "github.com/doitintl/bigquery"
	cloudResourceManagerDomain "github.com/doitintl/cloudresourcemanager/domain"
	crmMocks "github.com/doitintl/cloudresourcemanager/mocks"
	bqlensBigqueryMocks "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal/mocks"
	discoveryDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/domain"
	cloudConnectServiceMocks "github.com/doitintl/hello/scheduled-tasks/cloudconnect/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestTablesDiscovery(t *testing.T) {
	type fields struct {
		loggerMocks       *loggerMocks.ILogger
		discoveryBigquery *bqlensBigqueryMocks.Bigquery
		cloudConnect      *cloudConnectServiceMocks.CloudConnectService
		crm               *crmMocks.CloudResourceManager
	}

	ctx := context.Background()
	customerID := "test-customer"
	testProjectID := "test-project-id"
	testProject := cloudResourceManagerDomain.Project{
		ID: testProjectID,
	}
	testExcludedProjectID := "sys-1239872193"
	testExcludedProject := cloudResourceManagerDomain.Project{
		ID: testExcludedProjectID,
	}
	testRegion := "US"

	newGCPClientsError := errors.New("newGCPClientsForClientID error")
	getRegionsForProjectError := errors.New("getRegionsForProject error")
	ensureTableIsCorrectError := errors.New("ensureTableIsCorrect error")
	runQueryError := errors.New("runQuery error")
	runQueryNotFound := &googleapi.Error{
		Code: http.StatusNotFound,
	}
	runQueryForbidden := &googleapi.Error{
		Code: http.StatusForbidden,
	}

	testDatasetStorageBillingModel := make(discoveryDomain.DatasetStorageBillingModel)
	testDestinationTable := &bigquery.Table{}

	crmMock := crmMocks.NewCloudResourceManager(t)
	clientOptions := []option.ClientOption{}

	testBQClient, err := bigquery.NewClient(ctx, common.TestProjectID)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		input TablesDiscoveryPayload
	}

	tests := []struct {
		name      string
		args      args
		on        func(*fields)
		wantedErr error
	}{
		{
			name: "NewGCPClients fails",
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(nil, nil, newGCPClientsError).Once()
			},
			wantedErr: newGCPClientsError,
		},
		{
			name: "GetRegionsForProject fails",
			args: args{input: TablesDiscoveryPayload{[]*cloudResourceManagerDomain.Project{&testProject}}},
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(&pkg.ConnectClients{
						BQ: &bq.Service{
							BigqueryService: testBQClient,
						},
						CRM: crmMock,
					}, clientOptions, nil).Once()
				f.discoveryBigquery.On("GetRegionsAndStorageBillingModelForProject", ctx, testProjectID, testBQClient).
					Return(nil, nil, getRegionsForProjectError).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Errorf", errGetRegionsAndStorageBillingModelForProjectTpl, testProjectID, getRegionsForProjectError)

			},
		},
		{
			name: "EnsureTableExists fails",
			args: args{input: TablesDiscoveryPayload{[]*cloudResourceManagerDomain.Project{&testProject}}},
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(&pkg.ConnectClients{
						BQ: &bq.Service{
							BigqueryService: testBQClient,
						},
						CRM: crmMock,
					}, clientOptions, nil).Once()
				f.discoveryBigquery.On("GetRegionsAndStorageBillingModelForProject", ctx, testProjectID, testBQClient).
					Return(testDatasetStorageBillingModel, []string{testRegion}, nil).Once()
				f.discoveryBigquery.On("EnsureTableIsCorrect", ctx, testBQClient).
					Return(nil, ensureTableIsCorrectError).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedErr: ensureTableIsCorrectError,
		},
		{
			name: "RunDiscoveryQuery fails",
			args: args{input: TablesDiscoveryPayload{[]*cloudResourceManagerDomain.Project{&testProject}}},
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(&pkg.ConnectClients{
						BQ: &bq.Service{
							BigqueryService: testBQClient,
						},
						CRM: crmMock,
					}, clientOptions, nil).Once()
				f.discoveryBigquery.On("GetRegionsAndStorageBillingModelForProject", ctx, testProjectID, testBQClient).
					Return(testDatasetStorageBillingModel, []string{testRegion}, nil).Once()
				f.discoveryBigquery.On("EnsureTableIsCorrect", ctx, testBQClient).
					Return(testDestinationTable, nil).Once()
				f.discoveryBigquery.On("RunDiscoveryQuery",
					ctx, testBQClient, mock.AnythingOfType("string"),
					testDestinationTable, mock.AnythingOfType("dal.RowProcessor")).
					Return(runQueryError).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
			wantedErr: runQueryError,
		},
		{
			name: "RunDiscoveryQuery not found",
			args: args{input: TablesDiscoveryPayload{[]*cloudResourceManagerDomain.Project{&testProject}}},
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(&pkg.ConnectClients{
						BQ: &bq.Service{
							BigqueryService: testBQClient,
						},
						CRM: crmMock,
					}, clientOptions, nil).Once()
				f.discoveryBigquery.On("GetRegionsAndStorageBillingModelForProject", ctx, testProjectID, testBQClient).
					Return(testDatasetStorageBillingModel, []string{testRegion}, nil).Once()
				f.discoveryBigquery.On("EnsureTableIsCorrect", ctx, testBQClient).
					Return(testDestinationTable, nil).Once()
				f.discoveryBigquery.On("RunDiscoveryQuery",
					ctx, testBQClient, mock.AnythingOfType("string"),
					testDestinationTable, mock.AnythingOfType("dal.RowProcessor")).
					Return(runQueryNotFound).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
		},
		{
			name: "RunDiscoveryQuery forbidden",
			args: args{input: TablesDiscoveryPayload{[]*cloudResourceManagerDomain.Project{&testProject}}},
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(&pkg.ConnectClients{
						BQ: &bq.Service{
							BigqueryService: testBQClient,
						},
						CRM: crmMock,
					}, clientOptions, nil).Once()
				f.discoveryBigquery.On("GetRegionsAndStorageBillingModelForProject", ctx, testProjectID, testBQClient).
					Return(testDatasetStorageBillingModel, []string{testRegion}, nil).Once()
				f.discoveryBigquery.On("EnsureTableIsCorrect", ctx, testBQClient).
					Return(testDestinationTable, nil).Once()
				f.discoveryBigquery.On("RunDiscoveryQuery",
					ctx, testBQClient, mock.AnythingOfType("string"),
					testDestinationTable, mock.AnythingOfType("dal.RowProcessor")).
					Return(runQueryForbidden).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
				f.loggerMocks.On("Warningf",
					mock.AnythingOfType("string"),
					mock.AnythingOfType("[]string"),
					runQueryForbidden,
				).Once()
			},
		},
		{
			name: "Happy path, one project is excluded",
			args: args{input: TablesDiscoveryPayload{[]*cloudResourceManagerDomain.Project{&testProject, &testExcludedProject}}},
			on: func(f *fields) {
				f.cloudConnect.On("NewGCPClients", ctx, customerID).
					Return(&pkg.ConnectClients{
						BQ: &bq.Service{
							BigqueryService: testBQClient,
						},
						CRM: crmMock,
					}, clientOptions, nil).Once()
				f.discoveryBigquery.On("GetRegionsAndStorageBillingModelForProject", ctx, testProjectID, testBQClient).
					Return(testDatasetStorageBillingModel, []string{testRegion}, nil).Once()
				f.discoveryBigquery.On("EnsureTableIsCorrect", ctx, testBQClient).
					Return(testDestinationTable, nil).Once()
				f.discoveryBigquery.On("RunDiscoveryQuery",
					ctx, testBQClient, mock.AnythingOfType("string"),
					testDestinationTable, mock.AnythingOfType("dal.RowProcessor")).
					Return(nil).Once()
				f.loggerMocks.On("SetLabels", mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerMocks:       loggerMocks.NewILogger(t),
				discoveryBigquery: bqlensBigqueryMocks.NewBigquery(t),
				cloudConnect:      cloudConnectServiceMocks.NewCloudConnectService(t),
				crm:               crmMock,
			}

			s := &DiscoveryService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return fields.loggerMocks
				},
				discoveryBigquery: fields.discoveryBigquery,
				cloudConnect:      fields.cloudConnect,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			gotErr := s.TablesDiscovery(ctx, customerID, tt.args.input)
			if gotErr != nil && tt.wantedErr == nil {
				t.Errorf("BQLens.TablesDiscovery() error = %v, wantErr %v", gotErr, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), gotErr.Error())
			}
		})
	}
}

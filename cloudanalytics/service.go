package cloudanalytics

import (
	"context"

	slackgo "github.com/slack-go/slack"

	doitBQ "github.com/doitintl/bigquery"
	bqLensProxyClient "github.com/doitintl/bq-lens-proxy/client"
	bqLensProxyClientIface "github.com/doitintl/bq-lens-proxy/client/iface"
	bqLensOptimizerDAL "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal"
	bqLensDAL "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/bigquery"
	bqLensOptimizer "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service"
	bqLensOptimizerIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/service/iface"
	bqLensPricebook "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/service"
	forecast "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/service"
	forecastIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/service/iface"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	cloudConnectServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudconnect/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CloudAnalyticsService struct {
	loggerProvider       logger.Provider
	conn                 *connection.Connection
	fixerService         *fixer.FixerService
	customersDAL         customerDAL.Customers
	reportDAL            reportDAL.Reports
	doitEmployeesService doitemployees.ServiceInterface
	forecastService      forecastIface.Service
	bqLensPricebook      bqLensPricebook.Pricebook
	bqLensReservations   bqLensOptimizerIface.Reservations
	cloudConnect         cloudConnectServiceIface.CloudConnectService
	optmizerBQ           bqLensOptimizerIface.Bigquery
	proxyClient          bqLensProxyClientIface.BQLensProxyClient
}

func NewCloudAnalyticsService(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	reportsDAL reportDAL.Reports,
	customersDAL customerDAL.Customers,
) (*CloudAnalyticsService, error) {
	ctx := context.Background()

	fixerService, err := fixer.NewFixerService(loggerProvider, conn)
	if err != nil {
		return nil, err
	}

	forecastService, err := forecast.NewService(loggerProvider)
	if err != nil {
		return nil, err
	}

	bqLensRservations := bqLensOptimizer.NewReservations(
		loggerProvider,
		bqLensOptimizerDAL.NewReservations(),
		bqLensOptimizerDAL.NewCloudResourceManager())

	bqLensPricebook := bqLensPricebook.NewPricebook(loggerProvider, conn)

	cloudConnect := cloudconnect.NewCloudConnectService(loggerProvider, conn)

	optimizerBQDAL := bqLensDAL.NewBigquery(loggerProvider, &doitBQ.QueryHandler{})
	optimizerBQ := bqLensOptimizer.NewBigQueryService(loggerProvider, conn, optimizerBQDAL)

	proxyClient, err := bqLensProxyClient.NewClient(ctx, common.ProjectID, common.IsLocalhost)
	if err != nil {
		return nil, err
	}

	return &CloudAnalyticsService{
		loggerProvider,
		conn,
		fixerService,
		customersDAL,
		reportsDAL,
		doitemployees.NewService(conn),
		forecastService,
		bqLensPricebook,
		bqLensRservations,
		cloudConnect,
		optimizerBQ,
		proxyClient,
	}, nil
}

//go:generate mockery --name CloudAnalytics --output ./mocks
type CloudAnalytics interface {
	DeleteStaleDraftReports(ctx context.Context) error
	GetAccounts(ctx context.Context, customerID string, cloudProviders *[]string, filters []*report.ConfigFilter) ([]string, error)
	GetAttributions(ctx context.Context, filters []*domainQuery.QueryRequestX, rows []string, cols []string, customerID string) ([]*domainQuery.QueryRequestX, error)
	GetQueryRequest(ctx context.Context, customerID, reportID string) (*QueryRequest, *report.Report, error)
	GetQueryResult(ctx context.Context, qr *QueryRequest, customerID, email string) (QueryResult, error)
	GetReport(ctx context.Context, customerID, reportID string, presentationModeEnabled bool) (*report.Report, error)
	GetReportSlackUnfurl(ctx context.Context, reportID, customerID, URL, imageURL string) (*report.Report, map[string]slackgo.Attachment, error)
	NewQueryRequestFromFirestoreReport(ctx context.Context, customerID string, report *report.Report) (*QueryRequest, error)
	RunQuery(ctx context.Context, qr *QueryRequest, params RunQueryInput) (*QueryResult, error)
	UpdateCurrenciesTable(ctx context.Context) error
	UpdateCustomersInfoTable(ctx context.Context) error
}

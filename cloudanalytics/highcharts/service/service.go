package service

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	httpClient "github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	prodHighchartsServerURL = "https://highchart-server-alqysnpjoq-uc.a.run.app"
	devHighchartsServerURL  = "https://highchart-server-wsqwprteya-uc.a.run.app"
)

type Highcharts struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	cloudAnalytics         cloudanalytics.CloudAnalytics
	budgetService          service.IBudgetsService
	highchartsExportClient httpClient.IClient
}

func NewHighcharts(
	loggerProvider logger.Provider,
	conn *connection.Connection,
	budgetService service.IBudgetsService,
) (*Highcharts, error) {
	ctx := context.Background()

	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(logger.FromContext, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	highchartsExportClient, err := getHighchartsExportClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Highcharts{
		loggerProvider,
		conn,
		cloudAnalytics,
		budgetService,
		highchartsExportClient,
	}, nil
}

func getHighchartsExportClient(ctx context.Context) (httpClient.IClient, error) {
	baseURL := devHighchartsServerURL
	if common.Production {
		baseURL = prodHighchartsServerURL
	}

	tokenSource, err := idtoken.New().GetTokenSource(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	priorityProcedureClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL:     baseURL,
		Timeout:     3 * time.Minute,
		TokenSource: tokenSource,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	return priorityProcedureClient, err
}

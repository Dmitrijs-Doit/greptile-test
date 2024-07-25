package flexsaveresold

import (
	"context"
	"math/rand"
	"net/http/httptest"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/gin-gonic/gin"

	bqMocks "github.com/doitintl/bigquery/mocks"
	"github.com/doitintl/firestore/mocks"
	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	asset "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	flexAPIMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func createTestLogging(ctx context.Context) (*logger.Logging, error) {
	log, err := logger.NewLogging(ctx)
	if err != nil {
		return nil, err
	}

	return log, nil
}

func randomNumberString() string {
	rand.Seed(time.Now().UnixNano())
	someRandomValue := strconv.FormatInt(rand.Int63n(1000), 10)

	return someRandomValue
}

func createTestService(gcloudProjectName string, httpTestResponseRecorder *httptest.ResponseRecorder) (
	*Service, *gin.Context, *gin.Engine, logger.ILogger, error) {
	gin.SetMode(gin.TestMode)
	ctx, eng := gin.CreateTestContext(httpTestResponseRecorder)

	common.ProjectID = gcloudProjectName
	log, err := createTestLogging(ctx)

	if err != nil {
		return nil, ctx, eng, nil, err
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		return nil, ctx, eng, nil, err
	}

	flexsaveGCPUsageTable := devFlexsaveGCPUsageTable
	flexsaveGlobalTable := devFlexsaveGlobalTable

	if common.Production {
		flexsaveGCPUsageTable = prodFlexsaveGCPUsageTable
		flexsaveGlobalTable = prodFlexsaveGlobalTable
	}

	logProvider := logger.FromContext

	assetsDAL := asset.NewAssetsFirestoreWithClient(conn.Firestore)

	bigqueryClient, err := bigquery.NewClient(ctx, gcloudProjectName)
	if err != nil {
		return nil, ctx, eng, nil, err
	}

	flexRIService := &Service{
		logProvider,
		conn,
		flexsaveGCPUsageTable,
		flexsaveGlobalTable,
		&flexAPIMocks.FlexAPI{},
		assetsDAL,
		&mocks.Integrations{},
		&customerMocks.Customers{},
		&mpaMocks.MasterPayerAccounts{},
		bigqueryClient,
		&bqMocks.QueryHandler{},
		&bqMocks.JobHandler{},
	}

	return flexRIService, ctx, eng, log.Logger(ctx), nil
}

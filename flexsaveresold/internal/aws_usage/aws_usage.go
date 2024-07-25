package aws_usage

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	cloudHealthIface "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	bigQueryCache "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	consts "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Cht struct {
	ID       int64                  `firestore:"id"`
	Customer *firestore.DocumentRef `firestore:"customer"`
	Disabled bool                   `firestore:"disabled"`
}

type awsUsageService struct {
	loggerProvider logger.Provider
	*connection.Connection
	bigqueryInterface bigQueryCache.BigQueryServiceInterface
	cloudHealthDAL    cloudHealthIface.CloudHealthDAL
	customerDAL       customerDal.Customers
}

func NewAWSUsageService(log logger.Provider, conn *connection.Connection, bigqueryInterface bigQueryCache.BigQueryServiceInterface, cloudHealthDAL cloudHealthIface.CloudHealthDAL, customerDAL customerDal.Customers) *awsUsageService {
	return &awsUsageService{
		log,
		conn,
		bigqueryInterface,
		cloudHealthDAL,
		customerDAL,
	}
}

func (s *awsUsageService) GetMonthlyOnDemand(ctx context.Context, customerID string, applicableMonths []string) (map[string]float64, error) {
	applicableSpendByMonth := map[string]float64{}

	log := s.loggerProvider(ctx)

	var sharedPayerOndemandMonthlyData []types.SharedPayerOndemandMonthlyData

	t := time.Now().UTC()

	endDateTime := t.AddDate(0, 0, -consts.DaysToOffset)

	startDateTime := t.AddDate(0, -consts.FlexsaveHistoryMonthAmount+1, 0)

	startDate := startDateTime.Format("2006-01-02")
	endDate := endDateTime.Format("2006-01-02")

	customerRef := s.customerDAL.GetRef(ctx, customerID)

	var queryID string

	if err := s.bigqueryInterface.CheckActiveBillingTableExists(ctx, customerID); err == bigQueryCache.ErrNoActiveTable {
		chtCustomerID, err := s.cloudHealthDAL.GetCustomerCloudHealthID(ctx, customerRef)
		if err != nil {
			return applicableSpendByMonth, err
		}

		log.Infof("cht customerId %s", chtCustomerID)
		queryID = chtCustomerID
	} else {
		queryID = customerID
	}

	sharedPayerOndemandMonthlyData, err := s.bigqueryInterface.GetSharedPayerOndemandMonthlyData(ctx, queryID, startDate, endDate)
	if err != nil {
		return applicableSpendByMonth, err
	}

	monthlyData := make(map[string]float64)

	for _, monthlySpend := range sharedPayerOndemandMonthlyData {
		monthlyData[monthlySpend.MonthYear] = common.Round(monthlySpend.OndemandCost)
	}

	return monthlyData, err
}

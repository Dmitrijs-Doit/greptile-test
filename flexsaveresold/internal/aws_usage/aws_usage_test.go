package aws_usage

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	chtMocks "github.com/doitintl/hello/scheduled-tasks/cloudhealth/dal/mocks"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	bq "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery"
	bqMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/bigquery/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	consts "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func Test_awsUsageService_GetMonthlyOnDemand(t *testing.T) {
	var contextMock = mock.MatchedBy(func(_ context.Context) bool { return true })

	var someErr = errors.New("something wrong")

	type fields struct {
		loggerProvider    loggerMocks.ILogger
		Connection        *connection.Connection
		bigqueryInterface bqMocks.BigQueryServiceInterface
		cloudHealthDAL    chtMocks.CloudHealthDAL
		customerDAL       customerMocks.Customers
	}

	timeNow := time.Now().UTC()

	endDateTime := timeNow.AddDate(0, 0, -consts.DaysToOffset)

	startDateTime := timeNow.AddDate(0, -consts.FlexsaveHistoryMonthAmount+1, 0)

	startDate := startDateTime.Format("2006-01-02")
	endDate := endDateTime.Format("2006-01-02")

	SharedPayerOndemandMonthlyDataRow1 := types.SharedPayerOndemandMonthlyData{
		OndemandCost: 1887.40,
		MonthYear:    "1_2022",
	}
	SharedPayerOndemandMonthlyDataRow2 := types.SharedPayerOndemandMonthlyData{
		OndemandCost: 314.91,
		MonthYear:    "2_2022",
	}

	SharedPayerOndemandMonthlyDataRow3 := types.SharedPayerOndemandMonthlyData{
		OndemandCost: 381.94,
		MonthYear:    "3_2022",
	}

	var sharedPayerMonthlyOndemand = []types.SharedPayerOndemandMonthlyData{
		SharedPayerOndemandMonthlyDataRow1,
		SharedPayerOndemandMonthlyDataRow2,
		SharedPayerOndemandMonthlyDataRow3,
	}

	var result = make(map[string]float64)
	result["3_2022"] = 381.94
	result["2_2022"] = 314.91
	result["1_2022"] = 1887.40

	applicableMonths := []string{"3_2022", "2_2022", "1_2022"}

	customerRef := &firestore.DocumentRef{ID: "mr_customer"}

	var customerID = "11532"

	tests := []struct {
		name    string
		on      func(*fields)
		want    map[string]float64
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).Return(customerRef)
				f.bigqueryInterface.On("CheckActiveBillingTableExists", contextMock, customerID).Return(bq.ErrNoActiveTable)
				f.cloudHealthDAL.On("GetCustomerCloudHealthID", contextMock, customerRef).Return("11532", nil)
				f.loggerProvider.On("Infof", "cht customerId %s", "11532")
				f.bigqueryInterface.On("GetSharedPayerOndemandMonthlyData", contextMock, customerID, startDate, endDate).Return(sharedPayerMonthlyOndemand, nil)
			},
			want:    result,
			wantErr: nil,
		},
		{
			name: "CHT Id not available",
			on: func(f *fields) {
				f.customerDAL.On("GetRef", contextMock, customerID).Return(customerRef)
				f.loggerProvider.On("Infof", "Getting shared payer cache for customer Id %s startDate %s and endDate %s", customerID, startDate, endDate)
				f.bigqueryInterface.On("CheckActiveBillingTableExists", contextMock, customerID).Return(bq.ErrNoActiveTable)
				f.cloudHealthDAL.On("GetCustomerCloudHealthID", contextMock, customerRef).Return("", someErr)
			},
			want:    make(map[string]float64),
			wantErr: someErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &awsUsageService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				Connection:        fields.Connection,
				bigqueryInterface: &fields.bigqueryInterface,
				cloudHealthDAL:    &fields.cloudHealthDAL,
				customerDAL:       &fields.customerDAL,
			}
			applicableSpendByMonth, err := s.GetMonthlyOnDemand(context.Background(), customerID, applicableMonths)

			if tt.wantErr != nil {
				assert.Contains(t, err.Error(), tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(applicableSpendByMonth, tt.want) {
				t.Errorf("GetMonthlyOnDemand() got = %v, want %v", applicableSpendByMonth, tt.want)
			}
		})
	}
}

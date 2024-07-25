package bq

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/bigquery/mocks"
	fspkg "github.com/doitintl/firestore/pkg"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

const expectedSharedPayerOndemandMonthlyQuery = `
CREATE TEMP FUNCTION
	isKeyFromSystemLabelsPresent(labels ARRAY<STRUCT<key STRING,
	  value STRING>>,
	  key STRING)
	RETURNS bool
	LANGUAGE js AS """
  let labelPresent = false
  try {
	  labels.forEach(x => {
		  if (x["key"] === key) {

			labelPresent = true
		  }
	  })

  } catch(e) {
	  // Nowhere to go from here.
  }
  return labelPresent
  """;
WITH res AS (
SELECT cost, FORMAT_DATE("%m_%Y", DATE(usage_date_time)) AS month_year,
isKeyFromSystemLabelsPresent(system_labels, "cmp/flexsave_eligibility") AS is_flexsave_eligibility_label_present,
FORMAT_DATE("%Y", DATE(usage_date_time)) AS year,
FORMAT_DATE("%m", DATE(usage_date_time)) AS month
FROM aws_billing_11531.doitintl_billing_export_v1_11531
WHERE cost_type = "Usage" AND operation LIKE 'RunInstances%'
AND (sku_description LIKE '%Box%')
AND NOT REGEXP_CONTAINS(sku_description, r"(\:m1\.|\:m2\.|\:m3\.|\:c1\.|\:c3\.|\:i2\.|\:cr1\.|\:r3\.|\:hs1\.|\:g2\.|\:t1\.)")
AND service_id = "AmazonEC2" AND DATE(usage_date_time) BETWEEN "2022-02-01" AND "2023-02-09"
AND DATE(export_time) BETWEEN "2022-02-01" AND "2023-02-09")
SELECT IFNULL(SUM(cost),0) AS ondemand_cost, TRIM(month_year, '0') AS month_year
FROM res WHERE is_flexsave_eligibility_label_present is False GROUP BY month_year, year, month ORDER BY year, month`

func setup() (*BigQueryService, *mocks.QueryHandler) {
	client, err := bigquery.NewClient(context.Background(),
		"flextest",
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	qh := &mocks.QueryHandler{}

	return &BigQueryService{
		BigqueryClient: client,
		ProjectID:      "flextest",
		QueryHandler:   qh,
	}, qh
}

var timeInstance = time.Date(2022, time.Month(6), 21, 1, 10, 30, 0, time.UTC)

var mockCostRow = []pkg.ItemType{
	{
		Cost: 139,
		Date: timeInstance,
	},
	{
		Cost: 1232,
		Date: timeInstance.AddDate(0, -1, 0),
	},
	{
		Cost: 561,
		Date: timeInstance.AddDate(0, -3, -8),
	},
}

func TestNewBigQueryService(t *testing.T) {
	_, err := NewBigQueryService()
	assert.NoError(t, err)
}

func TestGetPayerSpendSummaryRunsTheFunctionWithoutError(t *testing.T) {
	ctx := context.Background()
	d, qh := setup()

	qh.
		On("Read", mock.Anything, mock.Anything).
		Return(func() iface.RowIterator {
			q := &mocks.RowIterator{}
			q.On("Next", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					arg := args.Get(0).(*pkg.ItemType)
					*arg = mockCostRow[0]
				}).
				Once()
			q.On("Next", mock.Anything).Return(iterator.Done).Once()
			return q
		}(), nil).
		Once()

	qh.
		On("Read", mock.Anything, mock.Anything).
		Return(func() iface.RowIterator {
			q := &mocks.RowIterator{}
			q.On("Next", mock.Anything).
				Return(nil).
				Run(func(args mock.Arguments) {
					arg := args.Get(0).(*pkg.ItemType)
					*arg = mockCostRow[0]
				}).
				Once()
			q.On("Next", mock.Anything).Return(iterator.Done).Once()
			return q
		}(), nil).
		Once()

	params := BigQueryParams{
		Context:             ctx,
		CustomerID:          "456",
		FirstOfCurrentMonth: time.Date(timeInstance.Year(), timeInstance.Month(), 1, 0, 0, 0, 0, time.UTC),
		NumberOfMonths:      5,
	}

	spendSummary, _ := d.GetPayerSpendSummary(params)

	assert.Equal(t, spendSummary, pkg.SpendDataMonthly{
		"6_2022": &fspkg.FlexsaveMonthSummary{
			Savings:       139,
			OnDemandSpend: 0,
		},
		"5_2022": &fspkg.FlexsaveMonthSummary{
			Savings:       0,
			OnDemandSpend: 0,
		},
		"4_2022": &fspkg.FlexsaveMonthSummary{
			Savings:       0,
			OnDemandSpend: 0,
		},
		"3_2022": &fspkg.FlexsaveMonthSummary{
			Savings:       0,
			OnDemandSpend: 0,
		},
		"2_2022": &fspkg.FlexsaveMonthSummary{
			Savings:       0,
			OnDemandSpend: 0,
		},
	},
	)
}

func TestSavingsPlanService_CustomerSavingsPlansCache(t *testing.T) {
	type fields struct {
		BigqueryClient *bigquery.Client
		ProjectID      string
		QueryHandler   mocks.QueryHandler
	}

	type args struct {
		ctx        context.Context
		customerID string
	}

	ctx := context.Background()
	customerID := "mr_customer"

	savingsPlansData := []types.SavingsPlanData{
		{
			SavingsPlanID:        "saving-plan/5631fb63-2450-4656-91ac-9c77efceb341",
			UpfrontPayment:       0,
			RecurringPayment:     0,
			Commitment:           20,
			Term:                 "1yr",
			ExpirationDateString: "2024-08-28T00:00:00Z",
			StartDateString:      "2021-08-28T00:00:00Z",
		},
	}

	savingsPlans := []types.SavingsPlanData{
		{
			SavingsPlanID:        "5631fb63-2450-4656-91ac-9c77efceb341",
			RecurringPayment:     0,
			Commitment:           20,
			Term:                 "1yr",
			ExpirationDateString: "2024-08-28T00:00:00Z",
			StartDateString:      "2021-08-28T00:00:00Z",
			ExpirationDate:       time.Date(2024, 8, 28, 0, 0, 0, 0, time.UTC),
			StartDate:            time.Date(2021, 8, 28, 0, 0, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    []types.SavingsPlanData
	}{
		{
			name: "happy path",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "SavingsPlanRecurringFee")
				})).Return(func() iface.RowIterator {

					q := &mocks.RowIterator{}
					q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {

						arg := args.Get(0).(*types.SavingsPlanData)
						*arg = savingsPlansData[0]

					}).Once()
					q.On("Next", mock.Anything).Return(iterator.Done).Once()
					return q
				}(), nil).
					Once()
			},
			want: savingsPlans,
		},

		{
			name: "bigquery error",
			args: args{
				ctx,
				customerID,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "SavingsPlanRecurringFee")
				})).Return(func() iface.RowIterator {

					q := &mocks.RowIterator{}

					q.On("Next", mock.Anything).Return(errors.New("iteration has failed"))
					return q
				}(), nil).
					Once()
			},
			wantErr: errors.New("iteration has failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			client, err := bigquery.NewClient(context.Background(),
				"flextest",
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if err != nil {
				panic(err)
			}

			s := &BigQueryService{
				BigqueryClient: client,
				ProjectID:      "flextest",
				QueryHandler:   &fields.QueryHandler,
			}

			got, err := s.GetCustomerSavingsPlanData(tt.args.ctx, tt.args.customerID)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BigqueryService.GetCustomerSavingsPlanData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSharedPayerOndemandMonthlyQuery(t *testing.T) {
	customerID := "11531"

	startDate := "2022-02-01"

	endDate := "2023-02-09"

	query := BuildSharedPayerOndemandMonthlyQuery(customerID, startDate, endDate)

	assert.Equal(t, strings.Join(strings.Fields(expectedSharedPayerOndemandMonthlyQuery), ""), strings.Join(strings.Fields(query), ""))
}

func TestGetSharedPayerOndemandMonthlyData(t *testing.T) {
	type fields struct {
		BigqueryClient *bigquery.Client
		QueryHandler   mocks.QueryHandler
	}

	type args struct {
		ctx        context.Context
		customerID string
		startDate  string
		endDate    string
	}

	ctx := context.Background()
	customerID := "mr_customer"
	startDate := "2022-01-01"
	endDate := "2023-01-31"

	SharedPayerOnDemandMonthlyDataRow1 := types.SharedPayerOndemandMonthlyData{
		OndemandCost: 1887.40,
		MonthYear:    "01_2022",
	}
	SharedPayerOnDemandMonthlyDataRow2 := types.SharedPayerOndemandMonthlyData{
		OndemandCost: 314.91,
		MonthYear:    "02_2022",
	}

	SharedPayerOnDemandMonthlyDataRow3 := types.SharedPayerOndemandMonthlyData{
		OndemandCost: 381.94,
		MonthYear:    "03_2022",
	}

	var sharedPayerMonthlyOndemand = []types.SharedPayerOndemandMonthlyData{
		SharedPayerOnDemandMonthlyDataRow1,
		SharedPayerOnDemandMonthlyDataRow2,
		SharedPayerOnDemandMonthlyDataRow3,
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    []types.SharedPayerOndemandMonthlyData
	}{
		{
			name: "happy path",
			args: args{
				ctx,
				customerID,
				startDate,
				endDate,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					ctx,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						return strings.Contains(query.QueryConfig.Q, "RunInstances")
					})).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*types.SharedPayerOndemandMonthlyData)
							arg.MonthYear = SharedPayerOnDemandMonthlyDataRow1.MonthYear
							arg.OndemandCost = SharedPayerOnDemandMonthlyDataRow1.OndemandCost

						}).Once()

						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*types.SharedPayerOndemandMonthlyData)
							arg.MonthYear = SharedPayerOnDemandMonthlyDataRow2.MonthYear
							arg.OndemandCost = SharedPayerOnDemandMonthlyDataRow2.OndemandCost
						}).Once()

						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*types.SharedPayerOndemandMonthlyData)
							arg.MonthYear = SharedPayerOnDemandMonthlyDataRow3.MonthYear
							arg.OndemandCost = SharedPayerOnDemandMonthlyDataRow3.OndemandCost

						}).Once()

						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil)
			},
			want: sharedPayerMonthlyOndemand,
		},
		{
			name: "bigquery error",
			args: args{
				ctx,
				customerID,
				startDate,
				endDate,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read", mock.Anything, mock.MatchedBy(func(query *bigquery.Query) bool {
					return strings.Contains(query.QueryConfig.Q, "RunInstances")
				})).Return(func() iface.RowIterator {

					q := &mocks.RowIterator{}

					q.On("Next", mock.Anything).Return(errors.New("iteration has failed"))
					return q
				}(), nil).
					Once()
			},
			wantErr: errors.New("iteration has failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			client, err := bigquery.NewClient(context.Background(),
				"flextest",
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if err != nil {
				panic(err)
			}

			s := &BigQueryService{
				BigqueryClient: client,
				QueryHandler:   &fields.QueryHandler,
			}

			got, err := s.GetSharedPayerOndemandMonthlyData(tt.args.ctx, tt.args.customerID, tt.args.startDate, tt.args.endDate)

			if tt.wantErr != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BigqueryService.GetSharedPayerOndemandMonthlyData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCustomerCredits(t *testing.T) {
	type fields struct {
		BigqueryClient *bigquery.Client
		QueryHandler   mocks.QueryHandler
	}

	type args struct {
		ctx        context.Context
		customerID string
		nowTime    time.Time
	}

	ctx := context.Background()
	customerID := "cust1"

	nowTime := time.Date(2023, 5, 20, 13, 01, 01, 0, time.UTC)

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    CreditsResult
	}{
		{
			name: "customer has activate credits",
			args: args{
				ctx,
				customerID,
				nowTime,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					ctx,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						return strings.Contains(query.QueryConfig.Q, "aws activate|startup migrate credit issuance")
					})).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*pkg.CreditRow)
							arg.BillingAccountID = "cust1"
							arg.Cost = -12.34
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil)
			},
			want: CreditsResult{
				Credits: map[string]float64{
					"cust1": -12.34,
				},
				Err: nil,
			},
		},
		{
			name: "customer does Not have activate credits",
			args: args{
				ctx,
				customerID,
				nowTime,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					ctx,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						return strings.Contains(query.QueryConfig.Q, "aws activate|startup migrate credit issuance")
					})).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).Return(iterator.Done).Once()

						return q
					}(), nil)
			},
			want: CreditsResult{
				Credits: map[string]float64{},
				Err:     nil,
			},
		},
		{
			name: "error when fetching activate credits",
			args: args{
				ctx,
				customerID,
				nowTime,
			},
			on: func(f *fields) {
				f.QueryHandler.On("Read",
					ctx,
					mock.MatchedBy(func(query *bigquery.Query) bool {
						return strings.Contains(query.QueryConfig.Q, "aws activate|startup migrate credit issuance")
					})).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).Return(errors.New("someError"))

						return q
					}(), nil)
			},
			want: CreditsResult{
				Credits: map[string]float64{},
				Err:     errors.New("someError"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			client, err := bigquery.NewClient(context.Background(),
				"flextest",
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if err != nil {
				panic(err)
			}

			s := &BigQueryService{
				BigqueryClient: client,
				QueryHandler:   &fields.QueryHandler,
			}

			credits := s.GetCustomerCredits(tt.args.ctx, tt.args.customerID, tt.args.nowTime)

			if tt.wantErr != nil {
				assert.ErrorContains(t, credits.Err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(credits, tt.want) {
				t.Errorf("BigqueryService.GetCustomerCredits() = %v, want %v", credits, tt.want)
			}
		})
	}
}

func TestCheckActiveBillingTableExists(t *testing.T) {
	type fields struct {
		BigqueryClient *bigquery.Client
		ManagerHandler *mocks.BigqueryManagerHandler
	}

	type args struct {
		ctx          context.Context
		chCustomerID string
	}

	ctx := context.Background()
	customerID := "mr_customer"

	metadataNotUpdated := &bigquery.TableMetadata{
		LastModifiedTime: timeInstance,
	}

	metadataUpdated := &bigquery.TableMetadata{
		LastModifiedTime: time.Now().UTC(),
	}

	gapiErrorGroupNotFound := &googleapi.Error{
		Code: http.StatusNotFound,
	}

	tests := []struct {
		name     string
		fields   fields
		on       func(*fields)
		args     args
		expected error
	}{
		{
			name: "Table exists and was modified within the last 5 days",
			args: args{
				ctx:          ctx,
				chCustomerID: customerID,
			},
			on: func(f *fields) {
				f.ManagerHandler.On("GetTableMetadata",
					ctx,
					f.BigqueryClient.Dataset(getCustomerDataset(customerID)), "doitintl_billing_export_v1_mr_customer").
					Return(metadataUpdated, nil)
			},
			expected: nil,
		},
		{
			name: "Table does not exist",
			args: args{
				ctx:          ctx,
				chCustomerID: customerID,
			},
			on: func(f *fields) {
				f.ManagerHandler.On("GetTableMetadata",
					ctx,
					f.BigqueryClient.Dataset(getCustomerDataset(customerID)), "doitintl_billing_export_v1_mr_customer").
					Return(nil, gapiErrorGroupNotFound)
			},
			expected: ErrNoActiveTable,
		},
		{
			name: "Table is stale",
			args: args{
				ctx:          ctx,
				chCustomerID: customerID,
			},
			on: func(f *fields) {
				f.ManagerHandler.On("GetTableMetadata",
					ctx,
					f.BigqueryClient.Dataset(getCustomerDataset(customerID)), "doitintl_billing_export_v1_mr_customer").
					Return(metadataNotUpdated, nil)
			},
			expected: ErrNoActiveTable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := bigquery.NewClient(context.Background(),
				"flextest",
				option.WithoutAuthentication(),
				option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
			if err != nil {
				panic(err)
			}

			fields := &fields{
				BigqueryClient: client,
				ManagerHandler: &mocks.BigqueryManagerHandler{},
			}

			if tt.on != nil {
				tt.on(fields)
			}

			s := &BigQueryService{
				BigqueryClient:         fields.BigqueryClient,
				BigqueryManagerHandler: fields.ManagerHandler,
			}

			err = s.CheckActiveBillingTableExists(tt.args.ctx, tt.args.chCustomerID)

			if err != tt.expected {
				t.Errorf("Unexpected error. Expected: %v, Got: %v", tt.expected, err)
			}
		})
	}
}

func TestBigQueryService_GetAWSSupportedSKUs(t *testing.T) {
	type fields struct {
		queryHandler *mocks.QueryHandler
	}

	client, err := bigquery.NewClient(context.Background(),
		"flextest",
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name    string
		on      func(*fields)
		fields  fields
		want    []AWSSupportedSKU
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "successfully gets data",
			on: func(f *fields) {
				f.queryHandler.
					On("Read", mock.Anything, mock.Anything).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).
							Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*AWSSupportedSKU)
								*arg = AWSSupportedSKU{
									Operation:           "RunInstances",
									InstanceType:        "m5.large",
									Region:              "us-east-1",
									ActivationThreshold: 100,
									Database:            "MySQL",
								}
							}).
							Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()

			},
			fields: fields{
				queryHandler: &mocks.QueryHandler{},
			},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
			want: []AWSSupportedSKU{
				{
					Operation:           "RunInstances",
					InstanceType:        "m5.large",
					Region:              "us-east-1",
					ActivationThreshold: 100,
					Database:            "MySQL",
				},
			},
		},

		{
			name: "unable to get data due to error",
			on: func(f *fields) {
				f.queryHandler.
					On("Read", mock.Anything, mock.Anything).
					Return(nil, errors.New("error")).
					Once()

			},
			fields: fields{
				queryHandler: &mocks.QueryHandler{},
			},
			wantErr: assert.ErrorAssertionFunc(assert.Error),
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			s := &BigQueryService{
				BigqueryClient: client,
				QueryHandler:   tt.fields.queryHandler,
			}

			got, err := s.GetAWSSupportedSKUs(context.Background())
			if !tt.wantErr(t, err, fmt.Sprintf("GetAWSSupportedSKUs(%v)", context.Background())) {
				return
			}

			assert.Equalf(t, tt.want, got, "GetAWSSupportedSKUs(%v)", context.Background())
		})
	}
}

func TestBigQueryService_CheckIfPayerHasRecentActiveCredits(t *testing.T) {
	type fields struct {
		queryHandler *mocks.QueryHandler
	}

	client, err := bigquery.NewClient(context.Background(),
		"flexi",
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name    string
		on      func(*fields)
		fields  fields
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "confirms that there were credits active recently",
			on: func(f *fields) {
				f.queryHandler.
					On("Read", testutils.ContextBackgroundMock, mock.Anything).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).
							Return(nil).
							Run(func(args mock.Arguments) {
								arg := args.Get(0).(*struct{})
								*arg = struct{}{}
							}).
							Once()
						return q
					}(), nil).
					Once()

			},
			fields: fields{
				queryHandler: &mocks.QueryHandler{},
			},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
			want:    true,
		},

		{
			name: "no results means no credits active recently",
			on: func(f *fields) {
				f.queryHandler.
					On("Read", testutils.ContextBackgroundMock, mock.Anything).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).
							Return(iterator.Done).
							Once()
						return q
					}(), nil).
					Once()

			},
			fields: fields{
				queryHandler: &mocks.QueryHandler{},
			},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
			want:    false,
		},

		{
			name: "for any other errors we return false with error",
			on: func(f *fields) {
				f.queryHandler.
					On("Read", testutils.ContextBackgroundMock, mock.Anything).
					Return(func() iface.RowIterator {
						q := &mocks.RowIterator{}
						q.On("Next", mock.Anything).
							Return(errors.New("meh")).
							Once()
						return q
					}(), nil).
					Once()
			},
			fields: fields{
				queryHandler: &mocks.QueryHandler{},
			},
			wantErr: assert.ErrorAssertionFunc(assert.Error),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			s := &BigQueryService{
				BigqueryClient: client,
				QueryHandler:   tt.fields.queryHandler,
			}

			got, err := s.CheckIfPayerHasRecentActiveCredits(context.Background(), "cutomer-1", "payer-1")
			if !tt.wantErr(t, err, fmt.Sprintf("CheckIfPayerHasRecentActiveCredits(%v)", context.Background())) {
				return
			}

			assert.Equalf(t, tt.want, got, "CheckIfPayerHasRecentActiveCredits(%v)", context.Background())
		})
	}
}

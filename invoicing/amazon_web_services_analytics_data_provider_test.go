package invoicing

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/bigquery/iface"
	bigqueryMocks "github.com/doitintl/bigquery/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	analyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/mocks"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func Test_billingDataService_GetCustomerBillingData(t *testing.T) {
	type fields struct {
		Logger                 loggerMocks.ILogger
		billingDataQuery       mocks.BillingDataQuery
		billingDataTransformer mocks.BillingDataTransformer
		cloudAnalytics         analyticsMocks.CloudAnalytics
	}

	type args struct {
		ctx          *gin.Context
		customerID   string
		invoiceMonth time.Time
	}

	ctx := gin.Context{}
	testCustomer := "test-testCustomer-id"
	testInvoiceMonth := dateAsTime("2022-01-01")
	resultMap := map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
		dateAsTime("2022-01-01"): {
			pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 20.0, Savings: 4.0, FlexsaveComputeNegations: 0.0},
		},
	}
	resultAccounts := []string{"account1"}

	var tests = []struct {
		name                    string
		args                    *args
		on                      func(*fields)
		assert                  func(*testing.T, *fields)
		outDaysToAccountsToCost map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem
		outAccountIDs           []string
		outErr                  error
	}{
		{
			name: "Happy path",
			args: &args{
				ctx:          &ctx,
				customerID:   testCustomer,
				invoiceMonth: testInvoiceMonth,
			},
			on: func(f *fields) {
				provider := "amazon-web-services"
				f.cloudAnalytics.
					On("GetAccounts", &ctx, testCustomer, &[]string{provider}, mock.AnythingOfType("[]*report.ConfigFilter")).
					Return([]string{"test_account_1", "test_account_2"}, nil).
					Once()
				request := cloudanalytics.QueryRequest{}
				f.billingDataQuery.
					On("GetBillingDataQuery", &ctx, testInvoiceMonth, []string{"test_account_1", "test_account_2"}, provider).
					Return(&request, nil).
					Once()
				result := cloudanalytics.QueryResult{Rows: [][]bigquery.Value{{"1", "2"}, {"3", "4"}}}
				f.cloudAnalytics.
					On("GetQueryResult", &ctx, &request, testCustomer, "").
					Return(result, nil).
					Once()

				resultMap := resultMap
				f.billingDataTransformer.
					On("TransformToDaysToAccountsToCostAndAccountIDs", result.Rows).
					Return(resultMap, resultAccounts, nil).
					Once()
			},
			outDaysToAccountsToCost: resultMap,
			outAccountIDs:           resultAccounts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{}

			s := &BillingDataService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				billingDataQuery:       &f.billingDataQuery,
				billingDataTransformer: &f.billingDataTransformer,
				cloudAnalytics:         &f.cloudAnalytics,
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			resultMap, resultAccounts, err := s.GetCustomerBillingData(tt.args.ctx, tt.args.customerID, tt.args.invoiceMonth)

			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.outDaysToAccountsToCost, resultMap)
			assert.Equal(t, tt.outAccountIDs, resultAccounts)
		})
	}
}

func Test_billingDataService_getCustomerBillingRows(t *testing.T) {
	type fields struct {
		Logger                 loggerMocks.ILogger
		billingDataQuery       mocks.BillingDataQuery
		billingDataTransformer mocks.BillingDataTransformer
		analyticsWrapper       analyticsMocks.CloudAnalytics
	}

	type args struct {
		ctx          *gin.Context
		customerID   string
		invoiceMonth time.Time
	}

	ctx := gin.Context{}
	testCustomer := "test-testCustomer-id"
	testInvoiceMonth := dateAsTime("2022-01-01")

	var tests = []struct {
		name    string
		args    *args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		outData [][]bigquery.Value
		outErr  error
	}{
		{
			name: "Happy path",
			args: &args{
				ctx:          &ctx,
				customerID:   testCustomer,
				invoiceMonth: testInvoiceMonth,
			},
			on: func(f *fields) {
				provider := "amazon-web-services"
				f.analyticsWrapper.
					On("GetAccounts", &ctx, testCustomer, &[]string{provider}, mock.AnythingOfType("[]*report.ConfigFilter")).
					Return([]string{"test_account_1", "test_account_2"}, nil).
					Once()
				request := cloudanalytics.QueryRequest{}
				f.billingDataQuery.
					On("GetBillingDataQuery", &ctx, testInvoiceMonth, []string{"test_account_1", "test_account_2"}, provider).
					Return(&request, nil).
					Once()
				result := cloudanalytics.QueryResult{Rows: [][]bigquery.Value{{"1", "2"}, {"3", "4"}}}
				f.analyticsWrapper.
					On("GetQueryResult", &ctx, &request, testCustomer, "").
					Return(result, nil).
					Once()
			},
			outData: [][]bigquery.Value{{"1", "2"}, {"3", "4"}},
		},
		{
			name: "GetAccounts error",
			args: &args{
				ctx:          &ctx,
				customerID:   testCustomer,
				invoiceMonth: testInvoiceMonth,
			},
			on: func(f *fields) {
				provider := "amazon-web-services"
				f.analyticsWrapper.
					On("GetAccounts", &ctx, testCustomer, &[]string{provider}, mock.AnythingOfType("[]*report.ConfigFilter")).
					Return(nil, errors.New("GetAccounts-error")).
					Once()
			},
			outErr: errors.New("GetAccounts-error"),
			assert: func(t *testing.T, f *fields) {
				f.billingDataQuery.AssertNumberOfCalls(t, "GetBillingDataQuery", 0)
				f.analyticsWrapper.AssertNumberOfCalls(t, "GetQueryResult", 0)
			},
		},
		{
			name: "GetBillingDataQuery error",
			args: &args{
				ctx:          &ctx,
				customerID:   testCustomer,
				invoiceMonth: testInvoiceMonth,
			},
			on: func(f *fields) {
				provider := "amazon-web-services"
				f.analyticsWrapper.
					On("GetAccounts", &ctx, testCustomer, &[]string{provider}, mock.AnythingOfType("[]*report.ConfigFilter")).
					Return([]string{"test_account_1", "test_account_2"}, nil).
					Once()
				f.billingDataQuery.
					On("GetBillingDataQuery", &ctx, testInvoiceMonth, []string{"test_account_1", "test_account_2"}, provider).
					Return(nil, errors.New("GetBillingDataQuery-error")).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.analyticsWrapper.AssertNumberOfCalls(t, "GetQueryResult", 0)
			},
			outErr: errors.New("GetBillingDataQuery-error"),
		},
		{
			name: "GetQueryResult error returned",
			args: &args{
				ctx:          &ctx,
				customerID:   testCustomer,
				invoiceMonth: testInvoiceMonth,
			},
			on: func(f *fields) {
				provider := "amazon-web-services"
				f.analyticsWrapper.
					On("GetAccounts", &ctx, testCustomer, &[]string{provider}, mock.AnythingOfType("[]*report.ConfigFilter")).
					Return([]string{"test_account_1", "test_account_2"}, nil).
					Once()
				request := cloudanalytics.QueryRequest{}
				f.billingDataQuery.
					On("GetBillingDataQuery", &ctx, testInvoiceMonth, []string{"test_account_1", "test_account_2"}, provider).
					Return(&request, nil).
					Once()
				f.analyticsWrapper.
					On("GetQueryResult", &ctx, &request, testCustomer, "").
					Return(cloudanalytics.QueryResult{}, errors.New("GetQueryResult-error")).
					Once()
			},
			outErr: errors.New("GetQueryResult-error"),
		},
		{
			name: "GetQueryResult error in result",
			args: &args{
				ctx:          &ctx,
				customerID:   testCustomer,
				invoiceMonth: testInvoiceMonth,
			},
			on: func(f *fields) {
				provider := "amazon-web-services"
				f.analyticsWrapper.
					On("GetAccounts", &ctx, testCustomer, &[]string{provider}, mock.AnythingOfType("[]*report.ConfigFilter")).
					Return([]string{"test_account_1", "test_account_2"}, nil).
					Once()
				request := cloudanalytics.QueryRequest{}
				f.billingDataQuery.
					On("GetBillingDataQuery", &ctx, testInvoiceMonth, []string{"test_account_1", "test_account_2"}, provider).
					Return(&request, nil).
					Once()
				result := cloudanalytics.QueryResult{Error: &cloudanalytics.QueryResultError{
					Code:    "123",
					Message: "bigqeury error",
				}}
				f.analyticsWrapper.
					On("GetQueryResult", &ctx, &request, testCustomer, "").
					Return(result, nil).
					Once()
			},
			outErr: errors.New("monthly billing data query failed for customer: test-testCustomer-id and invoiceMonth: 2022-01-01 00:00:00 +0000 UTC resulted in error: cloudanalytics.QueryResultError{Code:\"123\", Status:0, Message:\"bigqeury error\"}"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{}

			s := &BillingDataService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				billingDataQuery:       &f.billingDataQuery,
				billingDataTransformer: &f.billingDataTransformer,
				cloudAnalytics:         &f.analyticsWrapper,
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			rows, err := s.getCustomerBillingRows(tt.args.ctx, tt.args.customerID, tt.args.invoiceMonth)

			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.outData, rows)
		})
	}
}

func Test_billingDataService_GetBillableAssetIDs(t *testing.T) {
	type fields struct {
		Logger       loggerMocks.ILogger
		queryHandler bigqueryMocks.QueryHandler
	}

	type args struct {
		ctx          context.Context
		invoiceMonth time.Time
	}

	invoiceMonthAsTime := dateAsTime("2022-01-01")
	tests := []struct {
		name   string
		args   args
		on     func(*fields)
		assert func(*testing.T, *fields)
		want   []string
		outErr error
	}{
		{
			name: "Happy path",
			args: args{
				ctx:          nil,
				invoiceMonth: invoiceMonthAsTime,
			},
			on: func(f *fields) {
				f.Logger.On("Infof", mock.Anything, mock.Anything).Return()
				f.Logger.On("Infof", mock.Anything, mock.Anything, mock.Anything).Return()

				f.queryHandler.On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {

					expected := []bigquery.QueryParameter{
						{Name: "invoice_month", Value: "202201"},
						{Name: "partition_start_date", Value: invoiceMonthAsTime},
						{Name: "partition_end_date", Value: dateAsTime("2022-02-06")},
					}

					return assert.Equalf(t, query.Parameters, expected, "what")
				})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryProjectIDRow)
							arg.ProjectID = "project1"
						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryProjectIDRow)
							arg.ProjectID = "project2"
						}).Once()

						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
			},
			assert: nil,
			want:   []string{"amazon-web-services-project1", "amazon-web-services-project2"},
			outErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{}

			s := &BillingDataService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				billingDataQuery:       nil,
				billingDataTransformer: nil,
				cloudAnalytics:         nil,
				queryHandler:           &f.queryHandler,
				bigQueryClientFunc: func(ctx context.Context) *bigquery.Client {
					client, err := bigquery.NewClient(context.Background(),
						"",
						option.WithoutAuthentication(),
						option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
					if err != nil {
						panic(err)
					}
					return client
				},
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			gotAssetIDs, err := s.GetBillableAssetIDs(tt.args.ctx, tt.args.invoiceMonth)

			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.want, gotAssetIDs)
		})
	}
}

func Test_billingDataService_GetCustomerInvoicingReadiness(t *testing.T) {
	type fields struct {
		Logger       loggerMocks.ILogger
		queryHandler bigqueryMocks.QueryHandler
	}

	type args struct {
		ctx                    context.Context
		customerID             string
		invoiceMonth           time.Time
		invoicingDaySwitchOver int
	}

	testCustomer := "test-testCustomer-id"
	now := time.Now().UTC()
	oldMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -2, 0)
	lastMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		args   args
		on     func(*fields)
		assert func(*testing.T, *fields)
		want   bool
		outErr error
	}{
		{
			name: "Old month",
			args: args{
				ctx:                    nil,
				customerID:             testCustomer,
				invoiceMonth:           oldMonth,
				invoicingDaySwitchOver: 10,
			},
			on: func(f *fields) {
				f.queryHandler.
					On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {
						// TODO: figure out why this doesn't work (two queries get mixed up)
						// expected := []bigquery.QueryParameter{
						// 	{Name: "partition_start_date_incl", Value: lastMonth},
						// 	{Name: "partition_end_date_excl", Value: thisMonth},
						// 	{Name: "customer_id", Value: testCustomer},
						// }
						// return assert.Equalf(t, query.Parameters, expected, "query params are messed up")
						return true
					})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryCustomerHasSharedPayerAssetsRow)
							arg.HasAssets = false
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
				f.Logger.On("Infof", mock.AnythingOfType("string"), false)
				f.queryHandler.
					On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "invoice_month", Value: oldMonth.Format("200601")},
							{Name: "partition_start_date_incl", Value: oldMonth},
							{Name: "partition_end_date_excl", Value: lastMonth},
						}
						return assert.Equalf(t, query.Parameters, expected, "query params are messed up")
					})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingMonthReadinessRow)
							arg.Ready = true
						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingMonthReadinessRow)
							arg.Ready = false
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
				f.Logger.On("Infof", mock.AnythingOfType("string"), true)
				f.Logger.On("Warningf", mock.AnythingOfType("string"), testCustomer)
			},
			assert: nil,
			want:   true,
			outErr: nil,
		},
		{
			name: "Last month switched over w/o shared payer assets",
			args: args{
				ctx:                    nil,
				customerID:             testCustomer,
				invoiceMonth:           lastMonth,
				invoicingDaySwitchOver: now.Day(),
			},
			on: func(f *fields) {
				f.queryHandler.
					On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {
						return true
					})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryCustomerHasSharedPayerAssetsRow)
							arg.HasAssets = false
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
				f.Logger.On("Infof", mock.AnythingOfType("string"), false)
				f.queryHandler.
					On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "invoice_month", Value: lastMonth.Format("200601")},
							{Name: "partition_start_date_incl", Value: lastMonth},
							{Name: "partition_end_date_excl", Value: thisMonth},
						}
						return assert.Equalf(t, query.Parameters, expected, "query params are messed up")
					})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingMonthReadinessRow)
							arg.Ready = true
						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingMonthReadinessRow)
							arg.Ready = true
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
				f.Logger.On("Infof", mock.AnythingOfType("string"), true)
			},
			assert: nil,
			want:   true,
			outErr: nil,
		},
		{
			name: "Last month not switched over w/o shared payer assets",
			args: args{
				ctx:                    nil,
				customerID:             testCustomer,
				invoiceMonth:           lastMonth,
				invoicingDaySwitchOver: now.Day() + 1,
			},
			on: func(f *fields) {
				f.queryHandler.
					On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {
						return true
					})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryCustomerHasSharedPayerAssetsRow)
							arg.HasAssets = false
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
				f.Logger.On("Infof", mock.AnythingOfType("string"), false)
				f.queryHandler.
					On("Read", nil, mock.MatchedBy(func(query *bigquery.Query) bool {
						expected := []bigquery.QueryParameter{
							{Name: "invoice_month", Value: lastMonth.Format("200601")},
							{Name: "partition_start_date_incl", Value: lastMonth},
							{Name: "partition_end_date_excl", Value: thisMonth},
						}
						return assert.Equalf(t, query.Parameters, expected, "query params are messed up")
					})).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingMonthReadinessRow)
							arg.Ready = true
						}).Once()
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingMonthReadinessRow)
							arg.Ready = true
						}).Once()
						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
				f.Logger.On("Infof", mock.AnythingOfType("string"), true)
			},
			assert: nil,
			want:   true,
			outErr: nil,
		},
		{
			name: "This month",
			args: args{
				ctx:                    nil,
				customerID:             testCustomer,
				invoiceMonth:           thisMonth,
				invoicingDaySwitchOver: 10,
			},
			on:     nil,
			assert: nil,
			want:   false,
			outErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{}

			s := &BillingDataService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				billingDataQuery:       nil,
				billingDataTransformer: nil,
				cloudAnalytics:         nil,
				queryHandler:           &f.queryHandler,
				bigQueryClientFunc: func(ctx context.Context) *bigquery.Client {
					client, err := bigquery.NewClient(ctx,
						"",
						option.WithoutAuthentication(),
						option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
					if err != nil {
						panic(err)
					}
					return client
				},
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			gotReady, err := s.GetCustomerInvoicingReadiness(tt.args.ctx, tt.args.customerID, tt.args.invoiceMonth, tt.args.invoicingDaySwitchOver)

			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.want, gotReady)
		})
	}
}

func Test_billingDataService_GetCustomerBillingSessionID(t *testing.T) {
	type fields struct {
		Logger       loggerMocks.ILogger
		queryHandler bigqueryMocks.QueryHandler
	}

	type args struct {
		ctx          context.Context
		customerID   string
		invoiceMonth time.Time
	}

	invoiceMonthAsTime := dateAsTime("2022-01-01")
	tests := []struct {
		name   string
		args   args
		on     func(*fields)
		assert func(*testing.T, *fields)
		want   string
		outErr error
	}{
		{
			name: "Happy path",
			args: args{
				ctx:          nil,
				invoiceMonth: invoiceMonthAsTime,
				customerID:   testCustomer,
			},
			on: func(f *fields) {
				f.queryHandler.On("Read", nil, mock.Anything).
					Return(func() iface.RowIterator {
						q := &bigqueryMocks.RowIterator{}
						q.On("Next", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
							arg := args.Get(0).(*QueryBillingSessionIDRow)
							arg.SessionID = "test-session-id"
						}).Once()

						q.On("Next", mock.Anything).Return(iterator.Done).Once()
						return q
					}(), nil).
					Once()
			},
			assert: nil,
			want:   "test-session-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			f := &fields{}

			s := &BillingDataService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				billingDataQuery:       nil,
				billingDataTransformer: nil,
				cloudAnalytics:         nil,
				queryHandler:           &f.queryHandler,
				bigQueryClientFunc: func(ctx context.Context) *bigquery.Client {
					client, err := bigquery.NewClient(context.Background(),
						"",
						option.WithoutAuthentication(),
						option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
					if err != nil {
						panic(err)
					}
					return client
				},
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			gotSessionID := s.GetCustomerBillingSessionID(tt.args.ctx, tt.args.customerID, tt.args.invoiceMonth)

			// assert

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.want, gotSessionID)
		})
	}
}

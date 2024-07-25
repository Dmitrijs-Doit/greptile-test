package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	analyticsMocks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/mocks"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

var (
	filterErr = errors.New("some error getting flex save query on filter build")
)

func Test_billingDataService_GetCustomerBillingRows(t *testing.T) {
	type fields struct {
		loggerProvider loggerMocks.ILogger
		queryBuilder   mocks.BillingDataQuery
		cloudAnalytics analyticsMocks.CloudAnalytics
	}

	type args struct {
		ctx          *gin.Context
		customerID   string
		invoiceMonth time.Time
	}

	ctx := &gin.Context{}
	ctx.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginInvoicingAws)

	parsedMonth, _ := time.Parse(times.YearMonthDayLayout, testInvoiceMonth)

	qr := mockQueryRequest(parsedMonth)
	queryResultErr := errors.New("some error obtaining result")
	errQueryResultStruct := cloudanalytics.QueryResult{
		Error: &cloudanalytics.QueryResultError{
			Code:    cloudanalytics.ErrorCodeQueryTimeout,
			Status:  500,
			Message: "we are sorry something went wrong",
		},
	}
	emptyQueryResultStruct := cloudanalytics.QueryResult{
		Rows: [][]bigquery.Value{},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		outData [][]bigquery.Value
		outErr  error
	}{
		{
			name: "success",
			args: args{
				ctx:          ctx,
				customerID:   customerID1,
				invoiceMonth: parsedMonth,
			},
			on: func(f *fields) {
				result := cloudanalytics.QueryResult{Rows: [][]bigquery.Value{{"asset_id", 34, 58, 0}, {"asset_id2", 23, 321, 6}}}
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(qr.Filters, nil).
					Once()
				f.queryBuilder.
					On("GetTimeSettings", parsedMonth).
					Return(qr.TimeSettings).
					Once()
				f.cloudAnalytics.
					On("GetQueryResult", ctx, &qr, customerID1, "").
					Return(result, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.loggerProvider.AssertNumberOfCalls(t, "Debugf", 0)
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 1)
				f.cloudAnalytics.AssertNumberOfCalls(t, "GetQueryResult", 1)
			},
			outData: [][]bigquery.Value{{"asset_id", 34, 58, 0}, {"asset_id2", 23, 321, 6}},
		},
		{
			name: "GetFlexsaveQuery error",
			args: args{
				ctx:          ctx,
				customerID:   customerID1,
				invoiceMonth: parsedMonth,
			},
			on: func(f *fields) {
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(nil, filterErr).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.loggerProvider.AssertNumberOfCalls(t, "Debugf", 0)
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 0)
				f.cloudAnalytics.AssertNumberOfCalls(t, "GetQueryResult", 0)
			},
			outErr: filterErr,
		},
		{
			name: "GetQueryResult error",
			args: args{
				ctx:          ctx,
				customerID:   customerID1,
				invoiceMonth: parsedMonth,
			},
			on: func(f *fields) {
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(qr.Filters, nil).
					Once()
				f.queryBuilder.
					On("GetTimeSettings", parsedMonth).
					Return(qr.TimeSettings).
					Once()
				f.cloudAnalytics.
					On("GetQueryResult", ctx, &qr, customerID1, "").
					Return(cloudanalytics.QueryResult{}, queryResultErr).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.loggerProvider.AssertNumberOfCalls(t, "Debugf", 0)
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 1)
				f.cloudAnalytics.AssertNumberOfCalls(t, "GetQueryResult", 1)
			},
			outErr: queryResultErr,
		},
		{
			name: "GetQueryResult error on result struct",
			args: args{
				ctx:          ctx,
				customerID:   customerID1,
				invoiceMonth: parsedMonth,
			},
			on: func(f *fields) {
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(qr.Filters, nil).
					Once()
				f.queryBuilder.
					On("GetTimeSettings", parsedMonth).
					Return(qr.TimeSettings).
					Once()
				f.cloudAnalytics.
					On("GetQueryResult", ctx, &qr, customerID1, "").
					Return(errQueryResultStruct, nil).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.loggerProvider.AssertNumberOfCalls(t, "Debugf", 0)
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 1)
				f.cloudAnalytics.AssertNumberOfCalls(t, "GetQueryResult", 1)
			},
			outErr: fmt.Errorf("monthly billing data query failed for customer: %s and invoiceMonth: %v resulted in error: %#v", customerID1, parsedMonth, *errQueryResultStruct.Error),
		},
		{
			name: "GetQueryResult no rows",
			args: args{
				ctx:          ctx,
				customerID:   customerID1,
				invoiceMonth: parsedMonth,
			},
			on: func(f *fields) {
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(qr.Filters, nil).
					Once()
				f.queryBuilder.
					On("GetTimeSettings", parsedMonth).
					Return(qr.TimeSettings).
					Once()
				f.cloudAnalytics.
					On("GetQueryResult", ctx, &qr, customerID1, "").
					Return(emptyQueryResultStruct, nil).
					Once()
				f.loggerProvider.
					On("Debugf", "customer %s: billing data query returned 0 rows", customerID1)
			},
			assert: func(t *testing.T, f *fields) {
				f.loggerProvider.AssertNumberOfCalls(t, "Debugf", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 1)
				f.cloudAnalytics.AssertNumberOfCalls(t, "GetQueryResult", 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			b := &BillingDataService{
				LoggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				QueryBuilder:   &f.queryBuilder,
				CloudAnalytics: &f.cloudAnalytics,
			}

			if tt.on != nil {
				tt.on(f)
			}

			rows, err := b.GetCustomerBillingRows(tt.args.ctx, tt.args.customerID, tt.args.invoiceMonth, common.Assets.AmazonWebServices)
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

func Test_billingDataService_GetFlexsaveQuery(t *testing.T) {
	type fields struct {
		loggerProvider   loggerMocks.ILogger
		queryBuilder     mocks.BillingDataQuery
		AnalyticsWrapper analyticsMocks.CloudAnalytics
	}

	type args struct {
		ctx          context.Context
		invoiceMonth time.Time
		accounts     []string
		provider     string
	}

	ctx := context.WithValue(context.Background(), domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginInvoicingAws)
	parsedMonth, _ := time.Parse(times.YearMonthDayLayout, testInvoiceMonth)

	qr := mockQueryRequest(parsedMonth)

	tests := []struct {
		name    string
		args    args
		outData *cloudanalytics.QueryRequest
		outErr  error
		on      func(*fields)
		assert  func(*testing.T, *fields)
	}{
		{
			name: "success",
			args: args{
				ctx:          ctx,
				invoiceMonth: parsedMonth,
				accounts:     []string{customerID1},
				provider:     common.Assets.AmazonWebServices,
			},
			on: func(f *fields) {
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(qr.Filters, nil).
					Once()
				f.queryBuilder.
					On("GetTimeSettings", parsedMonth).
					Return(qr.TimeSettings).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 1)
			},
			outData: &qr,
		},
		{
			name: "GetBillingQueryFilters error",
			args: args{
				ctx:          ctx,
				invoiceMonth: parsedMonth,
				accounts:     []string{customerID1},
				provider:     common.Assets.AmazonWebServices,
			},
			on: func(f *fields) {
				f.queryBuilder.
					On("GetBillingQueryFilters", common.Assets.AmazonWebServices, []string{flexSaveCharges}, false, "", []string(nil), false).
					Return(nil, filterErr).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.queryBuilder.AssertNumberOfCalls(t, "GetBillingQueryFilters", 1)
				f.queryBuilder.AssertNumberOfCalls(t, "GetTimeSettings", 0)
			},
			outErr: filterErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}

			b := &BillingDataService{
				LoggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				QueryBuilder: &f.queryBuilder,
			}

			if tt.on != nil {
				tt.on(f)
			}

			qr, err := b.GetFlexsaveQuery(tt.args.ctx, tt.args.invoiceMonth, tt.args.accounts, tt.args.provider)
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.outData, qr)
		})
	}
}

func mockQueryRequest(invoiceMonth time.Time) cloudanalytics.QueryRequest {
	awsProviderFilter, _ := domainQuery.NewFilter(domainQuery.FieldCloudProvider, domainQuery.WithValues([]string{common.Assets.AmazonWebServices}))
	costTypeFilter, _ := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{flexSaveCharges}))
	accountID, _ := domainQuery.NewRow(domainQuery.FieldProjectID)
	payerAccountID, _ := domainQuery.NewRowConstituentField("system_labels", "aws/payer_account_id")

	from := time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, invoiceMonth.Location())
	to := from.AddDate(0, 1, -1)
	timeSettings := &cloudanalytics.QueryRequestTimeSettings{
		Interval: "day",
		From:     &from,
		To:       &to,
	}

	return cloudanalytics.QueryRequest{
		Origin:         domainOrigin.QueryOriginInvoicingAws,
		Type:           "report",
		CloudProviders: &[]string{common.Assets.AmazonWebServices},
		Accounts:       []string{customerID1},
		TimeSettings:   timeSettings,
		Rows:           []*domainQuery.QueryRequestX{accountID, payerAccountID},
		Filters:        []*domainQuery.QueryRequestX{awsProviderFilter, costTypeFilter},
		Currency:       fixer.USD,
		NoAggregate:    true,
	}
}

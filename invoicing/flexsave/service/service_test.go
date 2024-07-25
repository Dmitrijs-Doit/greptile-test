package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/cloudtasks/iface"
	ctMocks "github.com/doitintl/cloudtasks/mocks"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assetsPkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/dal/mocks"
	billingDataMocks "github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/mocks"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	customerID1            = "s5x8qLHg0EGIJVBFayAb"
	customerID2            = "6QJMHMUaIYdEShihSweH"
	testInvoiceMonth       = "2022-01-01"
	testInvoiceMonthString = "2022-01"
	nonDateString          = "hello world"
)

func TestFlexsaveInvoiceService_UpdateFlexsaveInvoicingData(t *testing.T) {
	type fields struct {
		loggerProvider  loggerMocks.ILogger
		parser          mocks.InvoiceMonthParser
		assetDAL        assetDalMocks.Assets
		cloudTaskClient ctMocks.CloudTaskClient
	}

	type args struct {
		ctx               context.Context
		invoiceMonthInput string
		provider          string
	}

	ctx := context.Background()
	parsedMonth, _ := time.Parse(times.YearMonthDayLayout, testInvoiceMonth)

	tests := []struct {
		name    string
		args    args
		wantErr error
		on      func(*fields)
		assert  func(*testing.T, *fields)
		outErr  error
	}{
		{
			name: "invalid month",
			args: args{
				ctx:               ctx,
				invoiceMonthInput: "whatever",
				provider:          common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", "whatever").Return(time.Time{}, errors.New("failed"))
			},
			outErr: errors.New("failed"),
		},
		{
			name: "fail to get assets",
			args: args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
				provider:          common.Assets.BetterCloud,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(parsedMonth, nil)
				f.loggerProvider.On("Infof", "invoiceMonth %v", parsedMonth)
				f.assetDAL.On("ListBaseAssets", ctx, common.Assets.BetterCloud).Return(nil, errors.New("no assets found"))
			},
			outErr: errors.New("no assets found"),
		},
		{
			name: "run flow without error",
			args: args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
				provider:          common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(parsedMonth, nil)
				f.loggerProvider.On("Infof", "invoiceMonth %v", parsedMonth)
				f.assetDAL.On("ListBaseAssets", ctx, common.Assets.AmazonWebServicesStandalone).Return([]*assetsPkg.BaseAsset{
					{
						Customer: &firestore.DocumentRef{ID: customerID1},
					},
					{
						Customer: &firestore.DocumentRef{ID: customerID2},
					},
					{
						Customer: &firestore.DocumentRef{ID: customerID2},
					},
				}, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "s5x8qLHg0EGIJVBFayAb")
						})).
					Return(nil, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "6QJMHMUaIYdEShihSweH")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 2)
			},
			outErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			s := &FlexsaveInvoiceService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				invoiceMonthParser: &f.parser,
				assetDAL:           &f.assetDAL,
				cloudTaskClient:    &f.cloudTaskClient,
			}

			if tt.on != nil {
				tt.on(f)
			}

			err := s.UpdateFlexsaveInvoicingData(tt.args.ctx, tt.args.invoiceMonthInput, tt.args.provider)
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestFlexsaveInvoiceService_FlexsaveDataWorker(t *testing.T) {
	type fields struct {
		loggerProvider  loggerMocks.ILogger
		parser          mocks.InvoiceMonthParser
		dal             dalMocks.FlexsaveStandalone
		customersDal    customerMocks.Customers
		cloudTaskClient ctMocks.CloudTaskClient
		BillingData     billingDataMocks.BillingData
	}

	type args struct {
		ctx               *gin.Context
		customerID        string
		invoiceMonthInput string
		provider          string
	}

	ctx := gin.Context{}
	parsedMonth, _ := time.Parse(times.YearMonthDayLayout, testInvoiceMonth)
	customerRef := &firestore.DocumentRef{ID: customerID1}
	accountID := "023946476650"
	payerAccountID := "123"
	bigQueryRows := [][]bigquery.Value{{accountID, payerAccountID, 35.0}, {accountID, payerAccountID, 53.0}}
	nonDateError := errors.New("does not contain date")
	noCustomerErr := errors.New("customer not found")
	invalidPayloadForBatch := errors.New("cannot execute batch save")

	assetMap := map[string]pkg.MonthlyBillingFlexsaveStandalone{
		getAssetDocID(payerAccountID): {
			Customer: customerRef,
			Spend: map[string]float64{
				accountID: 88.0,
			},
			Type:         common.Assets.AmazonWebServicesStandalone,
			InvoiceMonth: testInvoiceMonthString,
		},
	}

	tests := []struct {
		name    string
		args    args
		wantErr error
		on      func(*fields)
		assert  func(*testing.T, *fields)
	}{
		{
			name: "success",
			args: args{
				ctx:               &ctx,
				customerID:        customerID1,
				invoiceMonthInput: testInvoiceMonth,
				provider:          common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(parsedMonth, nil)
				f.loggerProvider.On("Infof", "invoiceMonth %v", parsedMonth)
				f.loggerProvider.On("Infof", "Starting Analytics %sStandalone InvoicingDataWorker for customer %s", common.Assets.AmazonWebServicesStandalone, customerID1)
				f.BillingData.On("GetCustomerBillingRows", &ctx, customerID1, parsedMonth, common.Assets.AmazonWebServices).Return(bigQueryRows, nil)
				f.customersDal.On("GetRef", &ctx, customerID1).Return(customerRef)
				f.dal.On("BatchSetFlexsaveBillingData", &ctx, assetMap).Return(nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.parser.AssertNumberOfCalls(t, "GetInvoiceMonth", 1)
				f.loggerProvider.AssertNumberOfCalls(t, "Infof", 2)
				f.BillingData.AssertNumberOfCalls(t, "GetCustomerBillingRows", 1)
				f.customersDal.AssertNumberOfCalls(t, "GetRef", 1)
				f.dal.AssertNumberOfCalls(t, "BatchSetFlexsaveBillingData", 1)
			},
		},
		{
			name: "invalid date",
			args: args{
				ctx:               &ctx,
				customerID:        customerID1,
				invoiceMonthInput: nonDateString,
				provider:          common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", nonDateString).Return(time.Time{}, nonDateError)
			},
			assert: func(t *testing.T, f *fields) {
				f.parser.AssertNumberOfCalls(t, "GetInvoiceMonth", 1)
				f.loggerProvider.AssertNumberOfCalls(t, "Infof", 0)
				f.BillingData.AssertNumberOfCalls(t, "GetCustomerBillingRows", 0)
				f.customersDal.AssertNumberOfCalls(t, "GetRef", 0)
				f.dal.AssertNumberOfCalls(t, "BatchSetFlexsaveBillingData", 0)
			},
			wantErr: nonDateError,
		},
		{
			name: "invalid customer",
			args: args{
				ctx:               &ctx,
				customerID:        customerID1,
				invoiceMonthInput: testInvoiceMonth,
				provider:          common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(parsedMonth, nil)
				f.loggerProvider.On("Infof", "invoiceMonth %v", parsedMonth)
				f.loggerProvider.On("Infof", "Starting Analytics %sStandalone InvoicingDataWorker for customer %s", common.Assets.AmazonWebServicesStandalone, customerID1)
				f.BillingData.On("GetCustomerBillingRows", &ctx, customerID1, parsedMonth, common.Assets.AmazonWebServices).Return(nil, noCustomerErr)
			},
			assert: func(t *testing.T, f *fields) {
				f.parser.AssertNumberOfCalls(t, "GetInvoiceMonth", 1)
				f.loggerProvider.AssertNumberOfCalls(t, "Infof", 2)
				f.BillingData.AssertNumberOfCalls(t, "GetCustomerBillingRows", 1)
				f.customersDal.AssertNumberOfCalls(t, "GetRef", 0)
				f.dal.AssertNumberOfCalls(t, "BatchSetFlexsaveBillingData", 0)
			},
			wantErr: noCustomerErr,
		},
		{
			name: "invalid batch operation",
			args: args{
				ctx:               &ctx,
				customerID:        customerID1,
				invoiceMonthInput: testInvoiceMonth,
				provider:          common.Assets.AmazonWebServicesStandalone,
			},
			on: func(f *fields) {
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(parsedMonth, nil)
				f.loggerProvider.On("Infof", "invoiceMonth %v", parsedMonth)
				f.loggerProvider.On("Infof", "Starting Analytics %sStandalone InvoicingDataWorker for customer %s", common.Assets.AmazonWebServicesStandalone, customerID1)
				f.BillingData.On("GetCustomerBillingRows", &ctx, customerID1, parsedMonth, common.Assets.AmazonWebServices).Return(bigQueryRows, nil)
				f.customersDal.On("GetRef", &ctx, customerID1).Return(customerRef)
				f.dal.On("BatchSetFlexsaveBillingData", &ctx, assetMap).Return(invalidPayloadForBatch)
			},
			assert: func(t *testing.T, f *fields) {
				f.parser.AssertNumberOfCalls(t, "GetInvoiceMonth", 1)
				f.loggerProvider.AssertNumberOfCalls(t, "Infof", 2)
				f.BillingData.AssertNumberOfCalls(t, "GetCustomerBillingRows", 1)
				f.customersDal.AssertNumberOfCalls(t, "GetRef", 1)
				f.dal.AssertNumberOfCalls(t, "BatchSetFlexsaveBillingData", 1)
			},
			wantErr: invalidPayloadForBatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			s := &FlexsaveInvoiceService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.loggerProvider
				},
				invoiceMonthParser: &f.parser,
				dal:                &f.dal,
				custometrDAL:       &f.customersDal,
				cloudTaskClient:    &f.cloudTaskClient,
				billingData:        &f.BillingData,
			}

			if tt.on != nil {
				tt.on(f)
			}

			if err := s.FlexsaveDataWorker(tt.args.ctx, tt.args.customerID, tt.args.invoiceMonthInput, tt.args.provider); err != nil && err != tt.wantErr {
				t.Errorf("FlexsaveInvoiceService.FlexsaveDataWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func getAssetDocID(assetID string) string {
	return common.Assets.AmazonWebServicesStandalone + "-" + assetID
}

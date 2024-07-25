package invoicing

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/cloudtasks/iface"
	ctMocks "github.com/doitintl/cloudtasks/mocks"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	assetpkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	flexApiMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/mocks"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/aws"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/mocks"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

const (
	testCustomer           = "test-testCustomer-id"
	testInvoiceMonth       = "2022-01-01"
	testInvoiceMonthString = "2022-01"
)

func TestAnalyticsAWSInvoicingService_UpdateAmazonWebServicesInvoicingData(t *testing.T) {
	type fields struct {
		Logger           loggerMocks.ILogger
		common           mocks.CommonAWSInvoicing
		parser           mocks.InvoiceMonthParser
		billingData      mocks.BillingData
		assetsDAL        assetDalMocks.Assets
		assetSettingsDAL assetDalMocks.AssetSettings
		customers        customerDalMocks.Customers
		cloudTaskClient  ctMocks.CloudTaskClient
		flexsaveAPI      flexApiMocks.FlexAPI
	}

	type args struct {
		ctx               context.Context
		invoiceMonthInput string
	}

	ctx := context.Background()
	err := errors.New("something went wrong")

	var tests = []struct {
		name   string
		args   *args
		on     func(*fields)
		assert func(*testing.T, *fields)
		outErr error
	}{
		{
			name: "Happy path",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("float64"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"aa"}, []string{"bb"}, []string{"cc"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"11", "22"}, nil)

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11")
						})).
					Return(nil, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "22")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 2)
			},
			outErr: nil,
		},
		{
			name: "Create tasks even if we didn't find find all assets or customers in firestore",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"aa"}, []string{"bb"}, []string{"cc"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				err := errors.New("some assets were not found")
				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"11", "22"}, err)

				f.Logger.On("Warningf", "only found some customers for assets, continuing,  err: %v ", err)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11")
						})).
					Return(nil, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "22")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 2)
			},
			outErr: nil,
		},
		{
			name: "No customers found",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"aa"}, []string{"bb"}, []string{"cc"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{}, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 0)
			},
			outErr: errors.New("no customers with AWS assets were found"),
		},
		{
			name: "Fail to create second task path",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("float64"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"aa"}, []string{"bb"}, []string{"cc"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"11", "22"}, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"aa"}, []string{"bb"}, []string{"cc"}, nil)

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11")
						})).
					Return(nil, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "22")
						})).
					Return(nil, err)
				f.Logger.On("Errorf",
					mock.AnythingOfType("string"), "22", err).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 2)
			},
			outErr: err,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			s := &AnalyticsAWSInvoicingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				common:             &f.common,
				invoiceMonthParser: &f.parser,
				billingData:        &f.billingData,
				assetsDAL:          &f.assetsDAL,
				assetSettingsDAL:   &f.assetSettingsDAL,
				customers:          &f.customers,
				cloudTaskClient:    &f.cloudTaskClient,
				flexsaveAPI:        &f.flexsaveAPI,
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			err := s.UpdateAmazonWebServicesInvoicingData(tt.args.ctx, tt.args.invoiceMonthInput, "v1", false, false)
			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestAnalyticsAWSInvoicingService_UpdateAmazonWebServicesInvoicingData_v2_with_validate(t *testing.T) {
	type fields struct {
		Logger           loggerMocks.ILogger
		common           mocks.CommonAWSInvoicing
		parser           mocks.InvoiceMonthParser
		billingData      mocks.BillingData
		assetsDAL        assetDalMocks.Assets
		assetSettingsDAL assetDalMocks.AssetSettings
		customers        customerDalMocks.Customers
		cloudTaskClient  ctMocks.CloudTaskClient
		flexsaveAPI      flexApiMocks.FlexAPI
	}

	type args struct {
		ctx               context.Context
		invoiceMonthInput string
	}

	ctx := context.Background()
	err := errors.New("something went wrong")

	var tests = []struct {
		name   string
		args   *args
		on     func(*fields)
		assert func(*testing.T, *fields)
		outErr error
	}{
		{
			name: "Happy path - fetch customers from Audit table",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("float64"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"11", "22"}, []string{"00"}, []string{"s1", "s2"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"aa", "bb"}, nil)

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11") ||
								strings.Contains(cloudTaskConfig.URL, "22") ||
								strings.Contains(cloudTaskConfig.URL, "00") ||
								strings.Contains(cloudTaskConfig.URL, "s1") ||
								strings.Contains(cloudTaskConfig.URL, "s2") ||
								strings.Contains(cloudTaskConfig.URL, "aa") ||
								strings.Contains(cloudTaskConfig.URL, "bb")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 7)
			},
			outErr: nil,
		},
		{
			name: "Create tasks even if we didn't find find all assets or customers in firestore",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"11", "22"}, []string{"00"}, []string{"s1"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				err := errors.New("some assets were not found")
				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"aa", "bb"}, err)

				f.Logger.On("Warningf", "only found some customers for assets, continuing,  err: %v ", err)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11") ||
								strings.Contains(cloudTaskConfig.URL, "22") ||
								strings.Contains(cloudTaskConfig.URL, "00") ||
								strings.Contains(cloudTaskConfig.URL, "s1") ||
								strings.Contains(cloudTaskConfig.URL, "aa") ||
								strings.Contains(cloudTaskConfig.URL, "bb")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 6)
			},
			outErr: nil,
		},
		{
			name: "No customers found",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{}, []string{"00"}, []string{"s1"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))
				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				err := errors.New("some assets were not found")
				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"aa", "bb"}, err)

				f.Logger.On("Warningf", "only found some customers for assets, continuing,  err: %v ", err)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "00") ||
								strings.Contains(cloudTaskConfig.URL, "s1") ||
								strings.Contains(cloudTaskConfig.URL, "aa") ||
								strings.Contains(cloudTaskConfig.URL, "bb")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 4)
			},
			outErr: nil,
		},
		{
			name: "Fail to create second task path",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("float64"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"11", "22"}, []string{"00"}, []string{"s1"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"aa", "bb"}, nil)

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11") ||
								strings.Contains(cloudTaskConfig.URL, "00")
						})).
					Return(nil, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "22")
						})).
					Return(nil, err)
				f.Logger.On("Errorf",
					mock.AnythingOfType("string"), "22", err).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 3)
			},
			outErr: err,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			s := &AnalyticsAWSInvoicingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				common:             &f.common,
				invoiceMonthParser: &f.parser,
				billingData:        &f.billingData,
				assetsDAL:          &f.assetsDAL,
				assetSettingsDAL:   &f.assetSettingsDAL,
				customers:          &f.customers,
				cloudTaskClient:    &f.cloudTaskClient,
				flexsaveAPI:        &f.flexsaveAPI,
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			err := s.UpdateAmazonWebServicesInvoicingData(tt.args.ctx, tt.args.invoiceMonthInput, "v2", true, false)
			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestAnalyticsAWSInvoicingService_UpdateAmazonWebServicesInvoicingData_v2(t *testing.T) {
	type fields struct {
		Logger           loggerMocks.ILogger
		common           mocks.CommonAWSInvoicing
		parser           mocks.InvoiceMonthParser
		billingData      mocks.BillingData
		assetsDAL        assetDalMocks.Assets
		assetSettingsDAL assetDalMocks.AssetSettings
		customers        customerDalMocks.Customers
		cloudTaskClient  ctMocks.CloudTaskClient
		flexsaveAPI      flexApiMocks.FlexAPI
	}

	type args struct {
		ctx               context.Context
		invoiceMonthInput string
	}

	ctx := context.Background()
	err := errors.New("something went wrong")

	var tests = []struct {
		name   string
		args   *args
		on     func(*fields)
		assert func(*testing.T, *fields)
		outErr error
	}{
		{
			name: "Happy path - fetch customers from Audit table",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"11", "22"}, []string{"00"}, []string{"s1", "s2"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11") ||
								strings.Contains(cloudTaskConfig.URL, "22") ||
								strings.Contains(cloudTaskConfig.URL, "00") ||
								strings.Contains(cloudTaskConfig.URL, "s1") ||
								strings.Contains(cloudTaskConfig.URL, "s2")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 5)
			},
			outErr: nil,
		},
		{
			name: "Create tasks even if we didn't find find all assets or customers in firestore",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"11", "22"}, []string{"00"}, []string{"s1"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11") ||
								strings.Contains(cloudTaskConfig.URL, "22") ||
								strings.Contains(cloudTaskConfig.URL, "00") ||
								strings.Contains(cloudTaskConfig.URL, "s1")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 4)
			},
			outErr: nil,
		},
		{
			name: "No customers found",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{}, []string{"00"}, []string{"s1"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))
				assetIDs := []string{"asset1", "asset2"}
				f.billingData.On("GetBillableAssetIDs", ctx, invoiceMonthAsTime).Return(assetIDs, nil)

				err := errors.New("some assets were not found")
				f.assetSettingsDAL.On("GetCustomersForAssets", ctx, assetIDs).Return([]string{"aa", "bb"}, err)

				f.Logger.On("Warningf", "only found some customers for assets, continuing,  err: %v ", err)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "00") ||
								strings.Contains(cloudTaskConfig.URL, "s1")
						})).
					Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 2)
			},
			outErr: nil,
		},
		{
			name: "Fail to create second task path",
			args: &args{
				ctx:               ctx,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer)
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("float64"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				f.parser.On("GetInvoiceMonth", testInvoiceMonth).Return(invoiceMonthAsTime, nil)

				f.billingData.On("GetBillableCustomerIDs", ctx, invoiceMonthAsTime).Return([]string{"11", "22"}, []string{"00"}, []string{"s1"}, nil)
				f.Logger.On("Infof", mock.AnythingOfType("string"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("[]string"))

				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "11") ||
								strings.Contains(cloudTaskConfig.URL, "00")
						})).
					Return(nil, nil)
				f.cloudTaskClient.
					On("CreateTask",
						ctx,
						mock.MatchedBy(func(cloudTaskConfig *iface.Config) bool {
							return strings.Contains(cloudTaskConfig.URL, "22")
						})).
					Return(nil, err)
				f.Logger.On("Errorf",
					mock.AnythingOfType("string"), "22", err).
					Once()
			},
			assert: func(t *testing.T, f *fields) {
				f.cloudTaskClient.AssertNumberOfCalls(t, "CreateTask", 3)
			},
			outErr: err,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			s := &AnalyticsAWSInvoicingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				common:             &f.common,
				invoiceMonthParser: &f.parser,
				billingData:        &f.billingData,
				assetsDAL:          &f.assetsDAL,
				assetSettingsDAL:   &f.assetSettingsDAL,
				customers:          &f.customers,
				cloudTaskClient:    &f.cloudTaskClient,
				flexsaveAPI:        &f.flexsaveAPI,
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			err := s.UpdateAmazonWebServicesInvoicingData(tt.args.ctx, tt.args.invoiceMonthInput, "v2", false, false)
			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func TestAnalyticsAWSInvoicingService_AmazonWebServicesInvoicingDataWorker(t *testing.T) {
	type fields struct {
		Logger                loggerMocks.ILogger
		common                mocks.CommonAWSInvoicing
		parser                mocks.InvoiceMonthParser
		billingData           mocks.BillingData
		assetsDAL             assetDalMocks.Assets
		assetSettingsDAL      assetDalMocks.AssetSettings
		monthlyBillingDataDAL mocks.MonthlyBillingData
		customers             customerDalMocks.Customers
		flexsaveAPI           flexApiMocks.FlexAPI
	}

	type args struct {
		ginCtx            *gin.Context
		customerID        string
		invoiceMonthInput string
	}

	ctx := &gin.Context{}

	var tests = []struct {
		name   string
		args   *args
		on     func(*fields)
		assert func(*testing.T, *fields)
		outErr error
	}{
		{
			name: "Happy path",
			args: &args{
				ginCtx:            ctx,
				customerID:        testCustomer,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer, mock.AnythingOfType("time.Time"), mock.AnythingOfType("bool"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Warningf", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				mockGetInvoiceMonth := f.parser.On("GetInvoiceMonth", testInvoiceMonth)
				mockGetInvoiceMonth.RunFn = func(args mock.Arguments) {
					dateString := args[0].(string)
					mockGetInvoiceMonth.ReturnArguments = mock.Arguments{dateAsTime(dateString), nil}
				}

				resultMap := map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
					dateAsTime("2022-01-01"): {
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "usage", Label: ""}:     &pkg.CostAndSavingsAwsLineItem{Costs: 15.0, Savings: 3.0, FlexsaveComputeNegations: 3.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "fs-account1", PayerAccountID: "account1", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 1.0, Savings: 0.0, FlexsaveComputeNegations: 0.0},
					},
				}
				resultAccounts := []string{"account1", "fs-account1"}
				f.billingData.
					On("GetCustomerBillingData", ctx, testCustomer, invoiceMonthAsTime).
					Return(resultMap, resultAccounts, nil).
					Once()
				assetID := "amazon-web-services-account1"
				fsAssetID := "amazon-web-services-fs-account1"
				assetRef := &firestore.DocumentRef{ID: assetID}
				errAssetRef := &firestore.DocumentRef{ID: fsAssetID}

				f.assetsDAL.
					On("GetRef", ctx, assetID).
					Return(assetRef, nil)
				f.assetsDAL.
					On("GetRef", ctx, fsAssetID).
					Return(errAssetRef, nil)

				entityRef := &firestore.DocumentRef{ID: "entity1"}
				assetSettings := &assetpkg.AWSAssetSettings{}

				assetSettings.Entity = entityRef

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, assetID).
					Return(assetSettings, nil).
					Once()

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, fsAssetID).
					Return(nil, errors.New("any")).
					Once()
				f.flexsaveAPI.
					On("ListFlexsaveAccountsWithCache", ctx, time.Minute*30).Return([]string{"fs-account1"}, nil)
				f.monthlyBillingDataDAL.
					On("GetCustomerAWSAssetIDtoMonthlyBillingData", ctx, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("time.Time"), true).
					Return(map[string]*pkg.MonthlyBillingAmazonWebServices{"amazon-web-services-account1": {}}, nil)

				customerRef := &firestore.DocumentRef{ID: testCustomer}
				f.customers.
					On("GetRef", ctx, testCustomer).
					Return(customerRef).
					Once()
				credits := []*aws.CustomerCreditAmazonWebServices{{Name: "credit1"}}
				f.common.
					On("GetAmazonWebServicesCredits", ctx, invoiceMonthAsTime, customerRef, []string{"account1"}).
					Return(credits, nil).
					Once()
				accountSpendMap := map[string]float64{"account1": float64(20)}
				accountToCreditAllocation := map[string]map[string]float64{"account1": {"credit1": float64(5)}}

				calculateSpendAndCreditsDataCall := f.common.
					On("CalculateSpendAndCreditsData",
						"2022-01",
						"account1",
						invoiceMonthAsTime,
						float64(15),
						assetSettings.Entity,
						assetRef,
						credits,
						mock.AnythingOfType("map[string]float64"),
						mock.AnythingOfType("map[string]map[string]float64"))
				calculateSpendAndCreditsDataCall.RunFn = func(args mock.Arguments) {
					accountID := args[1].(string)

					accountSpendArg := args[7].(map[string]float64)
					accountSpendArg[accountID] = accountSpendMap[accountID]

					accountToCreditAllocationArg := args[8].(map[string]map[string]float64)
					accountToCreditAllocationArg[accountID] = accountToCreditAllocation[accountID]
				}

				f.parser.On("GetInvoicingDaySwitchOver").Return(10)
				f.billingData.On("GetCustomerBillingSessionID", ctx, testCustomer, invoiceMonthAsTime).Return("test-session-id")
				f.billingData.On("GetCustomerInvoicingReadiness", ctx, testCustomer, invoiceMonthAsTime, f.parser.GetInvoicingDaySwitchOver()).Return(true, nil)
				f.billingData.On("HasCustomerInvoiceBeenIssued", ctx, testCustomer, invoiceMonthAsTime).Return(false, nil)
				f.billingData.On("SnapshotCustomerBillingTable", ctx, testCustomer, invoiceMonthAsTime).Return(nil)
				f.billingData.On("SaveCreditUtilizationToFS", ctx, invoiceMonthAsTime, mock.AnythingOfType("[]*aws.CustomerCreditAmazonWebServices")).Return(nil)

				f.monthlyBillingDataDAL.
					On("BatchUpdateMonthlyBillingData",
						ctx, "2022-01",
						mock.MatchedBy(func(assetIDToBillingDataMap map[*firestore.DocumentRef]interface{}) bool {

							assetBillingData := assetIDToBillingDataMap[assetRef].(pkg.MonthlyBillingAmazonWebServices)
							return len(assetIDToBillingDataMap) == 1 &&
								assetBillingData.Customer == customerRef &&
								reflect.DeepEqual(assetBillingData.Spend, float64(20)) &&
								reflect.DeepEqual(assetBillingData.Credits, map[string]float64{"credit1": float64(5)}) &&
								assetBillingData.Type == common.Assets.AmazonWebServices &&
								assetBillingData.Verified == true
						}), true).
					Return(nil)
			},
		},
		{
			name: "ProcessPayerStatusTransition flexsave management cost and credit rows",
			args: &args{
				ginCtx:            ctx,
				customerID:        testCustomer,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer, mock.AnythingOfType("time.Time"), mock.AnythingOfType("bool"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Warningf", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				mockGetInvoiceMonth := f.parser.On("GetInvoiceMonth", testInvoiceMonth)
				mockGetInvoiceMonth.RunFn = func(args mock.Arguments) {
					dateString := args[0].(string)
					mockGetInvoiceMonth.ReturnArguments = mock.Arguments{dateAsTime(dateString), nil}
				}

				resultMap := map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
					dateAsTime("2022-01-01"): {
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "Usage", Label: ""}:                               &pkg.CostAndSavingsAwsLineItem{Costs: 15.0, Savings: 0.0, FlexsaveComputeNegations: 0.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "Credit", Label: ""}:                              &pkg.CostAndSavingsAwsLineItem{Costs: -3.0, Savings: 0.0, FlexsaveComputeNegations: 0.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "fs-account1", PayerAccountID: "account1", CostType: "Credit", Label: SkuComputeSp3yrNoUpfrontFmt}: &pkg.CostAndSavingsAwsLineItem{Costs: -7.0, Savings: 0.0, FlexsaveComputeNegations: 0.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "fs-account1", PayerAccountID: "account1", CostType: "Anything", Label: ""}:                        &pkg.CostAndSavingsAwsLineItem{Costs: 0.75, Savings: 0.0, FlexsaveComputeNegations: 0.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "fs-account1", PayerAccountID: "account1", CostType: "Credit", Label: ""}:                          &pkg.CostAndSavingsAwsLineItem{Costs: -100.0, Savings: 0.0, FlexsaveComputeNegations: 0.0},
					},
				}
				resultAccounts := []string{"account1", "fs-account1"}
				f.billingData.
					On("GetCustomerBillingData", ctx, testCustomer, invoiceMonthAsTime).
					Return(resultMap, resultAccounts, nil).
					Once()
				assetID := "amazon-web-services-account1"
				fsAssetID := "amazon-web-services-fs-account1"
				assetRef := &firestore.DocumentRef{ID: assetID}
				errAssetRef := &firestore.DocumentRef{ID: fsAssetID}

				f.assetsDAL.
					On("GetRef", ctx, assetID).
					Return(assetRef, nil)
				f.assetsDAL.
					On("GetRef", ctx, fsAssetID).
					Return(errAssetRef, nil)

				entityRef := &firestore.DocumentRef{ID: "entity1"}
				assetSettings := &assetpkg.AWSAssetSettings{}

				assetSettings.Entity = entityRef

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, assetID).
					Return(assetSettings, nil).
					Once()

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, fsAssetID).
					Return(nil, errors.New("any")).
					Once()
				f.flexsaveAPI.
					On("ListFlexsaveAccountsWithCache", ctx, time.Minute*30).Return([]string{"fs-account1"}, nil)
				f.monthlyBillingDataDAL.
					On("GetCustomerAWSAssetIDtoMonthlyBillingData", ctx, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("time.Time"), true).
					Return(map[string]*pkg.MonthlyBillingAmazonWebServices{"amazon-web-services-account1": {}}, nil)

				customerRef := &firestore.DocumentRef{ID: testCustomer}
				f.customers.
					On("GetRef", ctx, testCustomer).
					Return(customerRef).
					Once()
				var credits []*aws.CustomerCreditAmazonWebServices
				f.common.
					On("GetAmazonWebServicesCredits", ctx, invoiceMonthAsTime, customerRef, []string{"account1"}).
					Return(credits, nil).
					Once()

				calculateSpendAndCreditsDataCall := f.common.
					On("CalculateSpendAndCreditsData",
						"2022-01",
						"account1",
						invoiceMonthAsTime,
						mock.AnythingOfType("float64"),
						assetSettings.Entity,
						assetRef,
						credits,
						mock.AnythingOfType("map[string]float64"),
						mock.AnythingOfType("map[string]map[string]float64"))
				calculateSpendAndCreditsDataCall.RunFn = func(args mock.Arguments) {
					accountID := args[1].(string)
					spend := args[3].(float64)
					accountSpendArg := args[7].(map[string]float64)
					accountSpendArg[accountID] += spend
				}

				f.parser.On("GetInvoicingDaySwitchOver").Return(10)
				f.billingData.On("GetCustomerBillingSessionID", ctx, testCustomer, invoiceMonthAsTime).Return("test-session-id")
				f.billingData.On("GetCustomerInvoicingReadiness", ctx, testCustomer, invoiceMonthAsTime, f.parser.GetInvoicingDaySwitchOver()).Return(false, nil)
				f.billingData.On("HasCustomerInvoiceBeenIssued", ctx, testCustomer, invoiceMonthAsTime).Return(false, nil)
				f.billingData.On("SnapshotCustomerBillingTable", ctx, testCustomer, invoiceMonthAsTime).Return(nil)
				f.billingData.On("SaveCreditUtilizationToFS", ctx, invoiceMonthAsTime, mock.AnythingOfType("[]*aws.CustomerCreditAmazonWebServices")).Return(nil)

				f.monthlyBillingDataDAL.
					On("BatchUpdateMonthlyBillingData",
						ctx, "2022-01",
						mock.MatchedBy(func(assetIDToBillingDataMap map[*firestore.DocumentRef]interface{}) bool {

							assetBillingData := assetIDToBillingDataMap[assetRef].(pkg.MonthlyBillingAmazonWebServices)
							return len(assetIDToBillingDataMap) == 1 &&
								assetBillingData.Customer == customerRef &&
								reflect.DeepEqual(assetBillingData.Spend, float64(12)) &&
								reflect.DeepEqual(assetBillingData.Credits, map[string]float64{}) &&
								assetBillingData.Type == common.Assets.AmazonWebServices &&
								reflect.DeepEqual(assetBillingData.Flexsave.FlexsaveSpCredits, float64(-7)) &&
								reflect.DeepEqual(assetBillingData.Flexsave.ManagementCosts, float64(-99.25)) &&
								assetBillingData.Verified == false
						}), true).
					Return(nil)
			},
		},
		{
			name: "Unknown AWS asset",
			args: &args{
				ginCtx:            ctx,
				customerID:        testCustomer,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer, mock.AnythingOfType("time.Time"), mock.AnythingOfType("bool"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Warningf", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
				f.Logger.On("Errorf", "invoice failed - could not allocate costs of value 1.7 for account unknown-account1 payer unknown-account2 as no asset/assetSettings doc ref found for customer test-testCustomer-id")
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				mockGetInvoiceMonth := f.parser.On("GetInvoiceMonth", testInvoiceMonth)
				mockGetInvoiceMonth.RunFn = func(args mock.Arguments) {
					dateString := args[0].(string)
					mockGetInvoiceMonth.ReturnArguments = mock.Arguments{dateAsTime(dateString), nil}
				}

				resultMap := map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
					dateAsTime("2022-01-01"): {
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "usage", Label: ""}:                  &pkg.CostAndSavingsAwsLineItem{Costs: 15.0, Savings: 3.0, FlexsaveComputeNegations: 3.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "unknown-account1", PayerAccountID: "unknown-account2", CostType: "usage", Label: ""}: &pkg.CostAndSavingsAwsLineItem{Costs: 1.7, Savings: 0.0, FlexsaveComputeNegations: 0.0},
					},
				}
				resultAccounts := []string{"account1", "unknown-account1"}
				f.billingData.
					On("GetCustomerBillingData", ctx, testCustomer, invoiceMonthAsTime).
					Return(resultMap, resultAccounts, nil).
					Once()
				assetID := "amazon-web-services-account1"
				fsAssetID := "amazon-web-services-unknown-account1"
				assetRef := &firestore.DocumentRef{ID: assetID}
				errAssetRef := &firestore.DocumentRef{ID: fsAssetID}

				f.assetsDAL.
					On("GetRef", ctx, assetID).
					Return(assetRef, nil)
				f.assetsDAL.
					On("GetRef", ctx, fsAssetID).
					Return(errAssetRef, nil)

				entityRef := &firestore.DocumentRef{ID: "entity1"}
				assetSettings := &assetpkg.AWSAssetSettings{}

				assetSettings.Entity = entityRef

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, assetID).
					Return(assetSettings, nil).
					Once()

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, fsAssetID).
					Return(nil, errors.New("any")).
					Once()
				f.flexsaveAPI.
					On("ListFlexsaveAccountsWithCache", ctx, time.Minute*30).Return([]string{}, nil)
				f.monthlyBillingDataDAL.
					On("GetCustomerAWSAssetIDtoMonthlyBillingData", ctx, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("time.Time"), true).
					Return(map[string]*pkg.MonthlyBillingAmazonWebServices{"amazon-web-services-account1": {}}, nil)

				customerRef := &firestore.DocumentRef{ID: testCustomer}
				f.customers.
					On("GetRef", ctx, testCustomer).
					Return(customerRef).
					Once()
				credits := []*aws.CustomerCreditAmazonWebServices{{Name: "credit1"}}
				f.common.
					On("GetAmazonWebServicesCredits", ctx, invoiceMonthAsTime, customerRef, []string{"account1"}).
					Return(credits, nil).
					Once()
				accountSpendMap := map[string]float64{"account1": float64(20)}
				accountToCreditAllocation := map[string]map[string]float64{"account1": {"credit1": float64(5)}}

				calculateSpendAndCreditsDataCall := f.common.
					On("CalculateSpendAndCreditsData",
						"2022-01",
						"account1",
						invoiceMonthAsTime,
						float64(15),
						assetSettings.Entity,
						assetRef,
						credits,
						mock.AnythingOfType("map[string]float64"),
						mock.AnythingOfType("map[string]map[string]float64"))
				calculateSpendAndCreditsDataCall.RunFn = func(args mock.Arguments) {
					accountID := args[1].(string)

					accountSpendArg := args[7].(map[string]float64)
					accountSpendArg[accountID] = accountSpendMap[accountID]

					accountToCreditAllocationArg := args[8].(map[string]map[string]float64)
					accountToCreditAllocationArg[accountID] = accountToCreditAllocation[accountID]
				}

				f.parser.On("GetInvoicingDaySwitchOver").Return(10)
				f.billingData.On("GetCustomerInvoicingReadiness", ctx, testCustomer, invoiceMonthAsTime, f.parser.GetInvoicingDaySwitchOver()).Return(true, nil)
				f.billingData.On("HasCustomerInvoiceBeenIssued", ctx, testCustomer, invoiceMonthAsTime).Return(false, nil)
				f.billingData.On("GetCustomerBillingSessionID", ctx, testCustomer, invoiceMonthAsTime).Return("test-session-id")

			},
			outErr: errors.New("invoice failed - could not allocate costs of value 1.7 for account unknown-account1 as no asset/assetSettings doc ref found for customer test-testCustomer-id"),
		},
		{
			name: "ProcessPayerStatusTransition marketplace and credit rows",
			args: &args{
				ginCtx:            ctx,
				customerID:        testCustomer,
				invoiceMonthInput: testInvoiceMonth,
			},
			on: func(f *fields) {
				invoiceMonthAsTime := dateAsTime(testInvoiceMonth)
				f.Logger.On("Infof", "fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonthAsTime)
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer, mock.AnythingOfType("time.Time"), mock.AnythingOfType("bool"))
				f.Logger.On("Infof", mock.AnythingOfType("string"), testCustomer, mock.Anything, mock.Anything, mock.Anything)

				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
				f.Logger.On("Debugf", mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("bool"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"))
				f.Logger.On("Warningf", mock.AnythingOfType("string"), mock.AnythingOfType("string"))
				f.Logger.On("Debugf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)

				mockGetInvoiceMonth := f.parser.On("GetInvoiceMonth", testInvoiceMonth)
				mockGetInvoiceMonth.RunFn = func(args mock.Arguments) {
					dateString := args[0].(string)
					mockGetInvoiceMonth.ReturnArguments = mock.Arguments{dateAsTime(dateString), nil}
				}

				resultMap := map[time.Time]map[pkg.CostAndSavingsAwsLineItemKey]*pkg.CostAndSavingsAwsLineItem{
					dateAsTime("2022-01-01"): {
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "Usage", Label: ""}:                                                &pkg.CostAndSavingsAwsLineItem{Costs: 15.0, Savings: 3.5, FlexsaveComputeNegations: 3.5},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "Credit", Label: ""}:                                               &pkg.CostAndSavingsAwsLineItem{Costs: -3.0, Savings: 0.0, FlexsaveComputeNegations: 0.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "Usage", Label: "", IsMarketplace: true, MarketplaceSD: "mangoDB"}: &pkg.CostAndSavingsAwsLineItem{Costs: 25.2, Savings: 0.0, FlexsaveComputeNegations: 0.0},
						pkg.CostAndSavingsAwsLineItemKey{AccountID: "account1", PayerAccountID: "0011111", CostType: "Usage", Label: "", IsMarketplace: true, MarketplaceSD: "couchDB"}: &pkg.CostAndSavingsAwsLineItem{Costs: 30.3, Savings: 0.0, FlexsaveComputeNegations: 0.0},
					},
				}
				resultAccounts := []string{"account1"}
				f.billingData.
					On("GetCustomerBillingData", ctx, testCustomer, invoiceMonthAsTime).
					Return(resultMap, resultAccounts, nil).
					Once()
				assetID := "amazon-web-services-account1"
				assetRef := &firestore.DocumentRef{ID: assetID}

				f.assetsDAL.
					On("GetRef", ctx, assetID).
					Return(assetRef, nil)

				entityRef := &firestore.DocumentRef{ID: "entity1"}
				assetSettings := &assetpkg.AWSAssetSettings{}

				assetSettings.Entity = entityRef

				f.assetSettingsDAL.
					On("GetAWSAssetSettings", ctx, assetID).
					Return(assetSettings, nil).
					Once()

				f.flexsaveAPI.
					On("ListFlexsaveAccountsWithCache", ctx, time.Minute*30).Return([]string{}, nil)
				f.monthlyBillingDataDAL.
					On("GetCustomerAWSAssetIDtoMonthlyBillingData", ctx, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("time.Time"), true).
					Return(map[string]*pkg.MonthlyBillingAmazonWebServices{"amazon-web-services-account1": {}}, nil)

				customerRef := &firestore.DocumentRef{ID: testCustomer}
				f.customers.
					On("GetRef", ctx, testCustomer).
					Return(customerRef).
					Once()
				creditRef := &firestore.DocumentRef{ID: "credit1"}
				credits := []*aws.CustomerCreditAmazonWebServices{{
					Name:                   "dummy credit1",
					Type:                   "amazon-web-services",
					Customer:               customerRef,
					Entity:                 entityRef,
					Assets:                 []*firestore.DocumentRef{assetRef},
					Currency:               "USD",
					Amount:                 35,
					StartDate:              invoiceMonthAsTime.AddDate(0, -2, 0),
					EndDate:                invoiceMonthAsTime.AddDate(0, 4, 0),
					DepletionDate:          nil,
					Utilization:            make(map[string]map[string]float64),
					Metadata:               nil,
					Alerts:                 nil,
					UpdatedBy:              nil,
					Timestamp:              time.Time{},
					Remaining:              35,
					RemainingPreviousMonth: 0,
					Touched:                false,
					Snapshot:               &firestore.DocumentSnapshot{Ref: creditRef},
				}}

				credits[0].Utilization[testInvoiceMonthString] = make(map[string]float64)
				f.common.
					On("GetAmazonWebServicesCredits", ctx, invoiceMonthAsTime, customerRef, []string{"account1"}).
					Return(credits, nil).
					Once()

				awsInvoicingService := commonAWSInvoicingService{logger.FromContext}

				calculateSpendAndCreditsDataCall := f.common.
					On("CalculateSpendAndCreditsData",
						"2022-01",
						mock.AnythingOfType("string"),
						invoiceMonthAsTime,
						mock.AnythingOfType("float64"),
						assetSettings.Entity,
						assetRef,
						credits,
						mock.AnythingOfType("map[string]float64"),
						mock.AnythingOfType("map[string]map[string]float64"))
				calculateSpendAndCreditsDataCall.RunFn = func(args mock.Arguments) {
					awsInvoicingService.CalculateSpendAndCreditsData(args[0].(string), args[1].(string),
						args[2].(time.Time), args[3].(float64), args[4].(*firestore.DocumentRef),
						args[5].(*firestore.DocumentRef), args[6].([]*aws.CustomerCreditAmazonWebServices), args[7].(map[string]float64), args[8].(map[string]map[string]float64))
				}
				//calculateSpendAndCreditsDataCall.RunFn = func(args mock.Arguments) {
				//	accountID := args[1].(string)
				//	spend := args[3].(float64)
				//	accountSpendArg := args[7].(map[string]float64)
				//	accountSpendArg[accountID] += spend
				//}

				f.parser.On("GetInvoicingDaySwitchOver").Return(10)
				f.billingData.On("GetCustomerBillingSessionID", ctx, testCustomer, invoiceMonthAsTime).Return("test-session-id")
				f.billingData.On("GetCustomerInvoicingReadiness", ctx, testCustomer, invoiceMonthAsTime, f.parser.GetInvoicingDaySwitchOver()).Return(false, nil)
				f.billingData.On("HasCustomerInvoiceBeenIssued", ctx, testCustomer, invoiceMonthAsTime).Return(false, nil)
				f.billingData.On("SnapshotCustomerBillingTable", ctx, testCustomer, invoiceMonthAsTime).Return(nil)
				f.billingData.On("SaveCreditUtilizationToFS", ctx, invoiceMonthAsTime, mock.AnythingOfType("[]*aws.CustomerCreditAmazonWebServices")).Return(nil)

				f.monthlyBillingDataDAL.
					On("BatchUpdateMonthlyBillingData",
						ctx, "2022-01",
						mock.MatchedBy(func(assetIDToBillingDataMap map[*firestore.DocumentRef]interface{}) bool {

							assetBillingData := assetIDToBillingDataMap[assetRef].(pkg.MonthlyBillingAmazonWebServices)
							returnValue := true
							returnValue = returnValue && len(assetIDToBillingDataMap) == 1 &&
								assetBillingData.Customer == customerRef &&
								reflect.DeepEqual(assetBillingData.Spend, float64(67.5)) &&
								reflect.DeepEqual(assetBillingData.Credits, map[string]float64{"credit1": 35.0}) &&
								assetBillingData.Type == common.Assets.AmazonWebServices
							returnValue = returnValue &&
								reflect.DeepEqual(assetBillingData.MarketplaceConstituents["marketplace_none"].Spend, 12.0) &&
								reflect.DeepEqual(assetBillingData.MarketplaceConstituents["marketplace_4075390099"].Spend, 25.2) &&
								reflect.DeepEqual(assetBillingData.MarketplaceConstituents["marketplace_701542243"].Spend, 30.3) &&
								reflect.DeepEqual(assetBillingData.MarketplaceConstituentsRef["marketplace_701542243"], "couchDB")
							returnValue = returnValue && assetBillingData.Verified == false

							totalCredit := assetBillingData.MarketplaceConstituents["marketplace_none"].Credits["credit1"] +
								assetBillingData.MarketplaceConstituents["marketplace_4075390099"].Credits["credit1"] +
								assetBillingData.MarketplaceConstituents["marketplace_701542243"].Credits["credit1"]

							returnValue = returnValue && totalCredit == assetBillingData.Credits["credit1"]
							return returnValue
						}), true).
					Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{}
			s := &AnalyticsAWSInvoicingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				common:                &f.common,
				invoiceMonthParser:    &f.parser,
				billingData:           &f.billingData,
				assetsDAL:             &f.assetsDAL,
				assetSettingsDAL:      &f.assetSettingsDAL,
				monthlyBillingDataDAL: &f.monthlyBillingDataDAL,
				customers:             &f.customers,
				cloudTaskClient:       nil,
				flexsaveAPI:           &f.flexsaveAPI,
			}

			if tt.on != nil {
				tt.on(f)
			}
			// act
			err := s.AmazonWebServicesInvoicingDataWorker(tt.args.ginCtx, tt.args.customerID, tt.args.invoiceMonthInput, false)

			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func Test_findDifference(t *testing.T) {
	assert.ElementsMatch(t, findDifference([]string{"aa", "bb", "cc"}, []string{"aa", "cc"}), []string{"bb"}, "failed basic difference test")
	assert.ElementsMatch(t, findDifference([]string{}, []string{"aa", "cc"}), []string{}, "failed empty difference test")
	assert.ElementsMatch(t, findDifference([]string{"aa", "bb", "cc"}, []string{}), []string{"aa", "bb", "cc"}, "failed nil difference test")
	assert.ElementsMatch(t, findDifference(nil, []string{"aa", "cc"}), []string{}, "failed empty difference test")
	assert.ElementsMatch(t, findDifference([]string{"aa", "bb", "cc"}, nil), []string{"aa", "bb", "cc"}, "failed nil difference test")
}

func Test_mergeSlices(t *testing.T) {
	assert.ElementsMatch(t, mergeSlices([]string{"aa", "bb", "cc"}, []string{"ab", "bc"}), []string{"aa", "ab", "bb", "bc", "cc"}, "failed basic merge test")
	assert.ElementsMatch(t, mergeSlices([]string{"Bc", "", "Aa", "CC"}, []string{"bc", "ab", ""}), []string{"", "Aa", "Bc", "CC", "ab", "bc"}, "failed basic merge test")
	assert.ElementsMatch(t, mergeSlices([]string{"ab", "Bb", "Aa", "CC"}, []string{"bc", "CC", "ab"}), []string{"Aa", "Bb", "CC", "ab", "bc"}, "failed basic merge test")
	assert.ElementsMatch(t, mergeSlices([]string{"ab", "Bb", "Aa", "CC"}, nil), []string{"Aa", "Bb", "CC", "ab"}, "failed basic merge test")
	assert.ElementsMatch(t, mergeSlices(nil, []string{"ab", "Bb", "Aa", "CC"}), []string{"Aa", "Bb", "CC", "ab"}, "failed basic merge test")
	assert.ElementsMatch(t, mergeSlices(nil, nil), nil, "failed basic merge test")

}

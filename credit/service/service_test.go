package service

import (
	"context"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schema"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/credit"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func mockReportField(cost float64) []schema.Report {
	return []schema.Report{
		{
			Cost: bigquery.NullFloat64{
				Float64: cost,
				Valid:   true,
			},
			Usage: bigquery.NullFloat64{
				Float64: 0,
				Valid:   true,
			},
			Savings: bigquery.NullFloat64{
				Float64: 0,
				Valid:   true,
			},
			Credit: bigquery.NullString{
				StringVal: "test",
				Valid:     true,
			},
			ExtMetric: nil,
		},
	}
}

const (
	month             = "2009-10"
	gcpBillingAccount = "ABC123-DEF456-123456"
	awsBillingAccount = "123456789101"
)

func getBillingRowMock() schema.BillingRow {
	usageStartTime, _ := time.ParseInLocation(times.YearMonthLayout, month, time.UTC)
	usageDateTime := usageStartTime
	location, _ := time.LoadLocation(domainQuery.TimeZonePST)
	y, m, d := usageDateTime.Date()
	usageDateTime = time.Date(y, m, d, 0, 0, 0, 0, location)

	return schema.BillingRow{
		BillingAccountID: gcpBillingAccount,
		CloudProvider:    common.Assets.GoogleCloud,
		Customer:         "test_customer",
		UsageDateTime: bigquery.NullDateTime{
			DateTime: civil.DateTimeOf(usageDateTime),
			Valid:    true,
		},
		UsageStartTime:         usageStartTime,
		UsageEndTime:           usageStartTime.AddDate(0, 1, -1),
		ExportTime:             usageStartTime,
		Currency:               string(fixer.USD),
		CurrencyConversionRate: 1,
		Invoice: &schema.Invoice{
			Month: strings.ReplaceAll(month, "-", ""),
		},
		Report:   mockReportField(-100),
		CostType: CostTypeCredit,
	}
}

func getBillingRowMockAdjustment() schema.BillingRow {
	baseRow := getBillingRowMock()
	baseRow.Report = mockReportField(50)
	baseRow.CostType = CostTypeCreditAdjustment

	return baseRow
}

func getBillingRowMockAWS() schema.BillingRow {
	baseRow := getBillingRowMock()
	baseRow.UsageDateTime = bigquery.NullDateTime{
		DateTime: civil.DateTimeOf(baseRow.UsageStartTime),
		Valid:    true,
	}
	baseRow.Customer = "test_aws_customer"
	baseRow.BillingAccountID = baseRow.Customer
	baseRow.CloudProvider = common.Assets.AmazonWebServices
	baseRow.ProjectID = bigquery.NullString{
		StringVal: awsBillingAccount,
		Valid:     true,
	}
	baseRow.Report = mockReportField(-50)

	return baseRow
}

func TestCreditsService_createCreditRows(t *testing.T) {
	type fields struct {
		Logger loggerMocks.ILogger
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	type args struct {
		ctx        context.Context
		creditData map[*firestore.DocumentRef]credit.BaseCredit
	}

	const (
		keyValidateWarning = "validate utilization key [%s][%s] error: path [%s] %s"
		creditDocPath      = "customers/test_customer/customerCredits/xyz"
	)

	tests := []struct {
		name   string
		on     func(*fields)
		assert func(*testing.T, *fields)
		args   args
		want   []schema.BillingRow
		outErr error
	}{
		{
			name: "valid path",
			args: args{
				ctx: ctx,
				creditData: map[*firestore.DocumentRef]credit.BaseCredit{
					{Path: creditDocPath}: {
						Name: "test",
						Customer: &firestore.DocumentRef{
							ID: "test_customer",
						},
						Utilization: map[string]map[string]float64{
							month: {
								gcpBillingAccount:                    50,
								gcpBillingAccount + adjustmentSuffix: 50,
							},
						},
						Type: common.Assets.GoogleCloud,
					},
				},
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Warningf", 0)
			},
			outErr: nil,
			want:   []schema.BillingRow{getBillingRowMock(), getBillingRowMockAdjustment()},
		},
		{
			name: "invalid utilization key",
			args: args{
				ctx: ctx,
				creditData: map[*firestore.DocumentRef]credit.BaseCredit{
					{Path: creditDocPath}: {
						Name: "test",
						Customer: &firestore.DocumentRef{
							ID: "test_customer",
						},
						Utilization: map[string]map[string]float64{
							month: {
								"this_is_invalid": 100,
							},
						},
						Type: common.Assets.GoogleCloud,
					},
				},
			},
			on: func(f *fields) {
				f.Logger.On("Warningf", keyValidateWarning, month, "this_is_invalid", creditDocPath, ErrInvalidCreditUtilizationKey)
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Warningf", 1)
			},
			outErr: nil,
		},
		{
			name: "valid aws path",
			args: args{
				ctx: ctx,
				creditData: map[*firestore.DocumentRef]credit.BaseCredit{
					{Path: creditDocPath}: {
						Name: "test",
						Customer: &firestore.DocumentRef{
							ID: "test_aws_customer",
						},
						Utilization: map[string]map[string]float64{
							month: {
								awsBillingAccount: 50,
							},
						},
						Type: common.Assets.AmazonWebServices,
					},
				},
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Warningf", 0)
			},
			outErr: nil,
			want:   []schema.BillingRow{getBillingRowMockAWS()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}
			s := &CreditsService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
			}

			if tt.on != nil {
				tt.on(&f)
			}

			got, err := s.createCreditRows(tt.args.ctx, tt.args.creditData)
			// assert
			if err != tt.outErr {
				assert.EqualErrorf(t, err, tt.outErr.Error(), "Error should be: %v, got: %v", tt.outErr.Error(), err)
			}

			if tt.assert != nil {
				tt.assert(t, &f)
			}

			if !reflect.DeepEqual(got, tt.want) {
				// since this depends on unordered map iteration
				oppositeOrder := []schema.BillingRow{got[1], got[0]}
				if !reflect.DeepEqual(oppositeOrder, tt.want) {
					t.Errorf("got: %#v \n want: %#v", got, tt.want)
				}
			}
		})
	}
}

func TestCreditsService_validateUtilizationKey(t *testing.T) {
	type args struct {
		cloudProvider    string
		billingAccountID string
	}

	tests := []struct {
		name                          string
		args                          args
		wantBillingAccountID          string
		wantIsCreditDiscoutAdjustment bool
		wantErr                       error
	}{
		{
			name: "valid gcp billing account id",
			args: args{
				cloudProvider:    common.Assets.GoogleCloud,
				billingAccountID: gcpBillingAccount,
			},
			wantBillingAccountID:          gcpBillingAccount,
			wantIsCreditDiscoutAdjustment: false,
			wantErr:                       nil,
		},
		{
			name: "valid aws billing account id",
			args: args{
				cloudProvider:    common.Assets.AmazonWebServices,
				billingAccountID: awsBillingAccount,
			},
			wantBillingAccountID:          awsBillingAccount,
			wantIsCreditDiscoutAdjustment: false,
			wantErr:                       nil,
		},
		{
			name: "valid gcp billing account id with credit discout adjustment",
			args: args{
				cloudProvider:    common.Assets.GoogleCloud,
				billingAccountID: gcpBillingAccount + adjustmentSuffix,
			},
			wantBillingAccountID:          gcpBillingAccount,
			wantIsCreditDiscoutAdjustment: true,
			wantErr:                       nil,
		},
		{
			name: "invalid billing account id",
			args: args{
				cloudProvider:    common.Assets.AmazonWebServices,
				billingAccountID: "_custom",
			},
			wantBillingAccountID:          "",
			wantIsCreditDiscoutAdjustment: false,
			wantErr:                       ErrInvalidCreditUtilizationKey,
		},
		{
			name: "microsoft azure billing account id",
			args: args{
				cloudProvider:    common.Assets.MicrosoftAzure,
				billingAccountID: "123456",
			},
			wantBillingAccountID:          "",
			wantIsCreditDiscoutAdjustment: false,
			wantErr:                       fmt.Errorf("invalid credit cloud provider %s", common.Assets.MicrosoftAzure),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CreditsService{}

			got, got1, err := s.validateUtilizationKey(tt.args.cloudProvider, tt.args.billingAccountID)
			if err != tt.wantErr {
				if err.Error() != tt.wantErr.Error() {
					t.Errorf("CreditsService.validateBillingAccountID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}

			if got != tt.wantBillingAccountID {
				t.Errorf("CreditsService.validateBillingAccountID() got = %v, want %v", got, tt.wantBillingAccountID)
			}

			if got1 != tt.wantIsCreditDiscoutAdjustment {
				t.Errorf("CreditsService.validateBillingAccountID() got1 = %v, want %v", got1, tt.wantIsCreditDiscoutAdjustment)
			}
		})
	}
}

func TestCreditsService_getReportValues(t *testing.T) {
	type fields struct {
	}

	type args struct {
		creditName  string
		creditValue float64
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   []schema.Report
	}{
		{
			name: "get report values",
			args: args{
				creditName:  "test",
				creditValue: -100,
			},
			want: mockReportField(-100),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CreditsService{}
			if got := s.getReportValues(tt.args.creditName, tt.args.creditValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreditsService.getReportValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreditsService_getTableLoaderPayload(t *testing.T) {
	type fields struct {
		Logger loggerMocks.ILogger
	}

	type args struct {
		ctx         context.Context
		billingRows []schema.BillingRow
	}

	var requestData = bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   consts.CustomBillingDev,
		DestinationDatasetID:   consts.CustomBillingDataset,
		DestinationTableName:   consts.CreditsTable,
		ObjectDir:              consts.CreditsTable,
		ConfigJobID:            consts.CreditsTable,
		WriteDisposition:       bigquery.WriteTruncate,
		RequirePartitionFilter: false,
		PartitionField:         domainQuery.FieldExportTime,
		Clustering:             &[]string{domainQuery.FieldCustomer, domainQuery.FieldCloudProvider},
	}

	rawRows := []schema.BillingRow{getBillingRowMock()}

	rows := make([]interface{}, len(rawRows))
	for i, v := range rawRows {
		rows[i] = v
	}

	ctx := context.Background()

	client, err := bigquery.NewClient(ctx,
		"",
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		panic(err)
	}

	bigQueryClientFunc := func(ctx context.Context) *bigquery.Client {
		return client
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *bqutils.BigQueryTableLoaderParams
		wantErr bool
	}{
		{
			name: "get table loader payload",
			args: args{
				ctx:         ctx,
				billingRows: []schema.BillingRow{getBillingRowMock()},
			},
			want: &bqutils.BigQueryTableLoaderParams{
				Client: client,
				Schema: &schema.CreditsSchema,
				Rows:   rows,
				Data:   &requestData,
			},
		},
	}

	for _, tt := range tests {
		f := fields{}

		t.Run(tt.name, func(t *testing.T) {
			s := &CreditsService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &f.Logger
				},
				bigQueryClientFunc: bigQueryClientFunc,
			}

			got, err := s.getTableLoaderPayload(tt.args.ctx, tt.args.billingRows)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreditsService.getTableLoaderPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got: %#v \n want: %#v", got, tt.want)
			}
		})
	}
}

func Test_updateDateFields(t *testing.T) {
	type args struct {
		billingRow *schema.BillingRow
		creditType string
		month      string
	}

	billingRow := &schema.BillingRow{
		Customer:      "test_customer",
		CloudProvider: common.Assets.GoogleCloud,
	}
	withDates := schema.BillingRow{
		Customer:      "test_customer",
		CloudProvider: common.Assets.GoogleCloud,
	}

	usageStartTime, _ := time.ParseInLocation(times.YearMonthLayout, month, time.UTC)
	usageDateTime := usageStartTime
	location, _ := time.LoadLocation(domainQuery.TimeZonePST)
	y, m, d := usageDateTime.Date()
	usageDateTime = time.Date(y, m, d, 0, 0, 0, 0, location)

	withDates.UsageStartTime = usageStartTime
	withDates.ExportTime = usageStartTime
	withDates.UsageEndTime = usageStartTime.AddDate(0, 1, -1)
	withDates.UsageDateTime = bigquery.NullDateTime{
		DateTime: civil.DateTimeOf(usageDateTime),
		Valid:    true,
	}
	withDates.Invoice = &schema.Invoice{
		Month: strings.ReplaceAll(month, "-", ""),
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{

			name: "get date fields",
			args: args{
				billingRow: billingRow,
				creditType: common.Assets.GoogleCloud,
				month:      month,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := updateDateFields(tt.args.billingRow, tt.args.creditType, tt.args.month); err != tt.wantErr {
				t.Errorf("getDateFields() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.args.billingRow, &withDates) {
				t.Errorf("got: %#v \n want: %#v", tt.args.billingRow, withDates)
			}
		})
	}
}

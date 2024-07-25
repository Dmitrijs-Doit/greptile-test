package service

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/bqutils"
	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/schema"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/looker/domain"
)

var (
	emptyContextMock = mock.AnythingOfType("*gin.emptyCtx")
)

func TestAssetsService_CreateLookerRows(t *testing.T) {
	type fields struct {
		contractDal mocks.Contracts
	}

	type args struct {
		ctx *gin.Context
	}

	now := time.Now()

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	contracts := getDummyContracts()
	interval := []time.Time{time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)}
	billingRows := getDummyBillingRows(*contracts[0], *contracts[1], interval)

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		on             func(*fields)
		expectedResult map[time.Time][]schema.BillingRow
	}{
		{
			name: "Create rows for looker contracts",
			fields: fields{
				contractDal: mocks.Contracts{},
			},
			args: args{
				ctx: ctx,
			},
			wantErr:        false,
			expectedResult: billingRows,

			on: func(f *fields) {
				f.contractDal.
					On("GetActiveCustomerContractsForProductType", emptyContextMock, "looker").
					Return(contracts, nil).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				contractDal: mocks.Contracts{},
			}

			s := &AssetsService{
				contractsDAL: &tt.fields.contractDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			result, err := s.CreateLookerRows(tt.args.ctx, contracts, interval)

			for key, val := range result {
				for i, res := range val {
					assert.Equal(t, tt.expectedResult[key][i].Usage.Amount, res.Usage.Amount)
					assert.Equal(t, tt.expectedResult[key][i].Usage.PricingUnit, res.Usage.PricingUnit)
					assert.Equal(t, tt.expectedResult[key][i].Usage.Unit, res.Usage.Unit)
					assert.Equal(t, tt.expectedResult[key][i].Usage.AmountInPricingUnits, res.Usage.AmountInPricingUnits)
					assert.Equal(t, tt.expectedResult[key][i].Invoice.Month, res.Invoice.Month)
					tt.expectedResult[key][i].Invoice = nil
					val[i].Invoice = nil
					tt.expectedResult[key][i].Usage = nil
					val[i].Usage = nil
				}
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("AssetService.CreateLookerRows() error = %v, wantErr %v", err, tt.wantErr)
			}

			for key, val := range result {
				assert.Equal(t, tt.expectedResult[key], val)
			}
		})
	}
}

func TestAssetsService_GetTableLoaderPayload(t *testing.T) {
	type fields struct {
		contractDal mocks.Contracts
	}

	type args struct {
		ctx context.Context
	}

	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	conn, _ := connection.NewConnection(ctx, log)
	now := time.Now()
	contracts := getDummyContracts()
	interval := []time.Time{time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)}
	billingRows := getDummyBillingRows(*contracts[0], *contracts[1], interval)
	rows := make([]interface{}, len(billingRows[interval[0]]))

	for i, v := range billingRows[interval[0]] {
		rows[i] = v
	}

	requestData := bqutils.BigQueryTableLoaderRequest{
		DestinationProjectID:   getProjectID(),
		DestinationDatasetID:   consts.CustomBillingDataset,
		DestinationTableName:   consts.LookerTable,
		ObjectDir:              consts.LookerTable,
		ConfigJobID:            consts.LookerTable,
		WriteDisposition:       bigquery.WriteTruncate,
		RequirePartitionFilter: true,
		PartitionField:         domainQuery.FieldExportTime,
		Clustering:             &[]string{domainQuery.FieldCustomer, domainQuery.FieldCloudProvider},
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		on             func(*fields)
		expectedResult *bqutils.BigQueryTableLoaderParams
	}{
		{
			name: "Get table loader payload for looker contract 1",
			fields: fields{
				contractDal: mocks.Contracts{},
			},
			args: args{
				ctx: ctx,
			},
			wantErr: false,
			expectedResult: &bqutils.BigQueryTableLoaderParams{

				Client: conn.Bigquery(ctx),
				Schema: &schema.CreditsSchema,
				Rows:   rows,
				Data:   &requestData,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{}

			s := &AssetsService{
				contractsDAL:           &tt.fields.contractDal,
				bigQueryFromContextFun: conn.Bigquery,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			for _, v := range billingRows {
				result, err := s.GetTableLoaderPayload(tt.args.ctx, v)

				if (err != nil) != tt.wantErr {
					t.Errorf("AssetService.GetTableLoaderPayload() error = %v, wantErr %v", err, tt.wantErr)
				}

				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
func getDummyContracts() []*pkg.Contract {
	now := time.Now()
	lastMonth := now.AddDate(0, -1, 0)

	assetsRef1 := &firestore.DocumentRef{
		ID: "assets/google-cloud-000000-000000-000001",
	}
	assetsRef2 := &firestore.DocumentRef{
		ID: "assets/google-cloud-000000-000000-000002",
	}

	customerRef1 := &firestore.DocumentRef{
		ID: "customer1",
	}
	customerRef2 := &firestore.DocumentRef{
		ID: "customer2",
	}

	lookerContract1 := pkg.Contract{
		ID: "contract1",
		Assets: []*firestore.DocumentRef{
			assetsRef1,
		},
		Type:      "looker",
		Customer:  customerRef1,
		Active:    true,
		StartDate: &lastMonth,
		Properties: map[string]interface{}{
			"contractDuration": 12,
			"invoiceFrequency": 1,
			"salesProcess":     "Existing Renewal",
			"skus": []interface{}{
				map[string]interface{}{
					"monthlySalesPrice": 100,
					"months":            12,
					"quantity":          1,
					"skuName": map[string]interface{}{
						"googleSku":        "14EB-3C03-C96B",
						"label":            "Add-On Instance - Customer Hosted",
						"monthlyListPrice": 2000,
					},
					"startDate": &lastMonth,
				},
			},
		},
		TimeCreated: lastMonth,
		Timestamp:   lastMonth,
	}
	lookerContract2 := pkg.Contract{
		ID:       "contract2",
		Type:     "looker",
		Customer: customerRef2,
		Assets: []*firestore.DocumentRef{
			assetsRef2,
		},
		Discount:  25,
		Active:    true,
		StartDate: &lastMonth,
		Properties: map[string]interface{}{
			"contractDuration": 12,
			"invoiceFrequency": 1,
			"salesProcess":     "Existing Renewal",
			"skus": []interface{}{
				map[string]interface{}{
					"monthlySalesPrice": 100,
					"months":            12,
					"quantity":          1,
					"skuName": map[string]interface{}{
						"googleSku":        "14EB-3C03-C96B",
						"label":            "Add-On Instance - Customer Hosted",
						"monthlyListPrice": 2000,
					},
					"startDate": &lastMonth,
				},
			},
		},
		TimeCreated: lastMonth,
		Timestamp:   lastMonth,
	}

	return []*pkg.Contract{
		&lookerContract1,
		&lookerContract2,
	}
}

func getDummyBillingRows(lookerContract1 pkg.Contract, lookerContract2 pkg.Contract, interval []time.Time) map[time.Time][]schema.BillingRow {
	monthlyListPrice := 2000
	monthlySalesPrice := 100
	billingRowsByPartition := make(map[time.Time][]schema.BillingRow)

	var properties domain.LookerContractProperties

	properties1, _ := properties.DecodePropertiesMapIntoStruct(lookerContract1.Properties)
	properties2, _ := properties.DecodePropertiesMapIntoStruct(lookerContract2.Properties)

	for _, day := range interval {
		for i := 0; i < 24; i++ {
			billingRow1 := schema.BillingRow{
				Customer:               lookerContract1.Customer.ID,
				BillingAccountID:       "000000-000000-000001",
				Cost:                   calculateHourlyCost(properties1.Skus[0], float64(monthlyListPrice), properties1, day),
				Currency:               "USD",
				CurrencyConversionRate: 1,
				CostType:               "regular",
				SkuID:                  bigquery.NullString{StringVal: "14EB-3C03-C96B", Valid: true},
				SkuDescription:         bigquery.NullString{StringVal: "Add-On Instance - Customer Hosted", Valid: true},
				CloudProvider:          common.Assets.GoogleCloud,
				ServiceDescription:     bigquery.NullString{StringVal: "Looker", Valid: true},
				Usage: &schema.Usage{
					Amount:               float64(1),
					AmountInPricingUnits: float64(1),
				},
				ServiceID: bigquery.NullString{StringVal: gcpTableMgmtDomain.LookerServiceID, Valid: true},
				Report: []schema.Report{
					{
						Cost: bigquery.NullFloat64{
							Float64: calculateHourlyCost(properties1.Skus[0], float64(monthlySalesPrice), properties1, day),
							Valid:   true,
						},
						Usage: bigquery.NullFloat64{
							Float64: 1,
							Valid:   true,
						},
					},
				},
				SystemLabels: []schema.Label{{Key: "cmp/source", Value: "looker"}},
			}
			location, err := time.LoadLocation(domainQuery.TimeZonePST)

			if err != nil {
				break
			}

			y, m, d := day.Date()

			usageDateTime := time.Date(y, m, d, i, 0, 0, 0, location)

			if err := updateDateFields(&billingRow1, usageDateTime); err != nil {
				return nil
			}

			billingRowsByPartition[billingRow1.ExportTime] = append(billingRowsByPartition[day], billingRow1)
		}
	}

	for _, day := range interval {
		for i := 0; i < 24; i++ {
			billingRow2 := schema.BillingRow{
				Customer:               lookerContract2.Customer.ID,
				BillingAccountID:       "000000-000000-000002",
				Cost:                   calculateHourlyCost(properties2.Skus[0], float64(monthlyListPrice), properties2, day),
				Currency:               "USD",
				CurrencyConversionRate: 1,
				CostType:               "regular",
				SkuID:                  bigquery.NullString{StringVal: "14EB-3C03-C96B", Valid: true},
				SkuDescription:         bigquery.NullString{StringVal: "Add-On Instance - Customer Hosted", Valid: true},
				CloudProvider:          common.Assets.GoogleCloud,
				ServiceDescription:     bigquery.NullString{StringVal: "Looker", Valid: true},
				Usage: &schema.Usage{
					Amount:               float64(1),
					AmountInPricingUnits: float64(1),
				},
				ServiceID: bigquery.NullString{StringVal: gcpTableMgmtDomain.LookerServiceID, Valid: true},
				Report: []schema.Report{
					{
						Cost: bigquery.NullFloat64{
							Float64: calculateHourlyCost(properties2.Skus[0], float64(monthlySalesPrice), properties2, day) * utils.ToProportion(25),
							Valid:   true,
						},
						Usage: bigquery.NullFloat64{
							Float64: 1,
							Valid:   true,
						},
					},
				},
				SystemLabels: []schema.Label{{Key: "cmp/source", Value: "looker"}},
			}
			location, err := time.LoadLocation(domainQuery.TimeZonePST)

			if err != nil {
				break
			}

			y, m, d := day.Date()

			usageDateTime := time.Date(y, m, d, i, 0, 0, 0, location)

			if err := updateDateFields(&billingRow2, usageDateTime); err != nil {
				return nil
			}

			billingRowsByPartition[billingRow2.ExportTime] = append(billingRowsByPartition[day], billingRow2)
		}
	}

	return billingRowsByPartition
}

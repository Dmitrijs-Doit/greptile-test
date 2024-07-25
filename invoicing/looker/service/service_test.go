package service

import (
	"net/http/httptest"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	lookerDomain "github.com/doitintl/hello/scheduled-tasks/looker/domain"
	lookerUtils "github.com/doitintl/hello/scheduled-tasks/looker/utils"
)

var (
	emptyContextMock = mock.AnythingOfType("*gin.emptyCtx")
)

func getTwoYearsAgo() time.Time {
	return time.Date(2021, 6, 30, 0, 0, 0, 0, time.UTC)
}
func getInvoiceMonth() time.Time {
	base := getTwoYearsAgo()
	return base.AddDate(2, 0, 0)
}
func getLastMonth() time.Time {
	base := getTwoYearsAgo()
	return base.AddDate(2, -1, 0)
}

func getTwoMonthsAgo() time.Time {
	base := getTwoYearsAgo()
	return base.AddDate(2, -2, 0)
}
func getOneAndHalfYearsAgo() time.Time {
	base := getTwoYearsAgo()
	return base.AddDate(0, 6, 0)
}

func getThreeMonthsAgo() time.Time {
	base := getTwoYearsAgo()
	return base.AddDate(2, -3, 0)
}

func TestInvoicingService_GetInvoiceRows(t *testing.T) {
	type fields struct {
		contractDal mocks.Contracts
	}

	type args struct {
		ctx *gin.Context
	}

	customerRef := &firestore.DocumentRef{
		ID: "customer1",
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	contracts := getDummyContracts()
	invoiceRows := getDummyInvoiceRows()

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        bool
		on             func(*fields)
		expectedResult domain.ProductInvoiceRows
	}{
		{
			name: "Create invoice rows for looker contracts to be billed",
			fields: fields{
				contractDal: mocks.Contracts{},
			},
			args: args{
				ctx: ctx,
			},
			wantErr:        false,
			expectedResult: invoiceRows,

			on: func(f *fields) {
				f.contractDal.
					On("GetActiveCustomerContractsForProductTypeAndMonth", emptyContextMock, customerRef, getInvoiceMonth(), "looker").
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

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			res := domain.ProductInvoiceRows{
				Type:  lookerUtils.LookerProductType,
				Rows:  make([]*domain.InvoiceRow, 0),
				Error: nil,
			}

			for _, contract := range contracts {
				var properties lookerDomain.LookerContractProperties

				properties, _ = properties.DecodePropertiesMapIntoStruct(contract.Properties)

				rows := extractRowsFromLookerContract(properties, getInvoiceMonth(), *contract)

				res.Rows = append(res.Rows, rows...)
			}

			for i := 0; i < len(res.Rows); i++ {
				row := res.Rows[i]
				assert.Equal(t, tt.expectedResult.Rows[i].Entity.ID, row.Entity.ID)

				res.Rows[i].Entity = nil
				tt.expectedResult.Rows[i].Entity = nil
				assert.Equal(t, tt.expectedResult.Rows[i].Bucket.ID, row.Bucket.ID)

				res.Rows[i].Bucket = nil

				tt.expectedResult.Rows[i].Bucket = nil
				if row.DeferredRevenuePeriod != nil {
					assert.Equal(t, tt.expectedResult.Rows[i].DeferredRevenuePeriod.StartDate, row.DeferredRevenuePeriod.StartDate)
					assert.Equal(t, tt.expectedResult.Rows[i].DeferredRevenuePeriod.EndDate, row.DeferredRevenuePeriod.EndDate)

					res.Rows[i].DeferredRevenuePeriod = nil
					tt.expectedResult.Rows[i].DeferredRevenuePeriod = nil
				}
			}

			for i, row := range res.Rows {
				assert.Equal(t, tt.expectedResult.Rows[i], row)
			}
		})
	}
}

func getDummyContracts() []*pkg.Contract {
	// invoice date will be hardcoded to 2021-07-01
	lastMonth := getLastMonth()
	twoMonthsAgo := getTwoMonthsAgo()
	threeMonthsAgo := getThreeMonthsAgo()
	oneAndHalfYearsAgo := getOneAndHalfYearsAgo()
	twoYearsAgo := getTwoYearsAgo()

	entity := &firestore.DocumentRef{
		Parent: &firestore.CollectionRef{},
		ID:     "entities/0000",
	}
	assetsRef := &firestore.DocumentRef{
		ID: "assets/google-cloud-000000-000000-000000",
	}
	customerRef := &firestore.DocumentRef{
		ID: "customer1",
	}

	//Contracts which should be billed:

	// to be billed every three months, started three months before invoice date...
	lookerContract1 := pkg.Contract{
		ID: "contract1",
		Assets: []*firestore.DocumentRef{
			assetsRef,
		},
		Entity:    entity,
		Discount:  0,
		Type:      "looker",
		Customer:  customerRef,
		Active:    true,
		StartDate: &threeMonthsAgo,
		Properties: map[string]interface{}{
			"contractDuration": 12,
			"invoiceFrequency": 3,
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
					"startDate": &threeMonthsAgo,
				},
			},
		},
		TimeCreated: threeMonthsAgo,
		Timestamp:   threeMonthsAgo,
	}
	// to be billed every month, started 1 month before invoice date...
	lookerContract2 := pkg.Contract{
		ID:       "contract2",
		Type:     "looker",
		Customer: customerRef,
		Assets: []*firestore.DocumentRef{
			assetsRef,
		},
		Entity:    entity,
		Discount:  0,
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

	//to be billed every year, started two years before invoice date...
	lookerContract3 := pkg.Contract{
		ID:       "contract3",
		Type:     "looker",
		Customer: customerRef,
		Assets: []*firestore.DocumentRef{
			assetsRef,
		},
		Entity:    entity,
		Discount:  0,
		Active:    true,
		StartDate: &twoYearsAgo,
		Properties: map[string]interface{}{
			"contractDuration": 36,
			"invoiceFrequency": 12,
			"salesProcess":     "Existing Renewal",
			"skus": []interface{}{
				map[string]interface{}{
					"monthlySalesPrice": 100,
					"months":            36,
					"quantity":          1,
					"skuName": map[string]interface{}{
						"googleSku":        "14EB-3C03-C96B",
						"label":            "Add-On Instance - Customer Hosted",
						"monthlyListPrice": 2000,
					},
					"startDate": &twoYearsAgo,
				},
			},
		},
		TimeCreated: twoYearsAgo,
		Timestamp:   twoYearsAgo,
	}

	//Contracts which should not be billed:

	//to be billed every 3 months, started two months before invoice date...
	lookerContract4 := pkg.Contract{
		ID:       "contract4",
		Type:     "looker",
		Customer: customerRef,
		Assets: []*firestore.DocumentRef{
			assetsRef,
		},
		Entity:    entity,
		Discount:  0,
		Active:    true,
		StartDate: &twoMonthsAgo,
		Properties: map[string]interface{}{
			"contractDuration": 12,
			"invoiceFrequency": 3,
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
					"startDate": &twoMonthsAgo,
				},
			},
		},
		TimeCreated: twoMonthsAgo,
		Timestamp:   twoMonthsAgo,
	}
	// contract to be billed every year, started 1.5 years before invoice date...
	lookerContract5 := pkg.Contract{
		ID:       "contract4",
		Type:     "looker",
		Customer: customerRef,
		Assets: []*firestore.DocumentRef{
			assetsRef,
		},
		Entity:    entity,
		Discount:  0,
		Active:    true,
		StartDate: &oneAndHalfYearsAgo,
		Properties: map[string]interface{}{
			"contractDuration": 36,
			"invoiceFrequency": 12,
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
					"startDate": &oneAndHalfYearsAgo,
				},
			},
		},
		TimeCreated: oneAndHalfYearsAgo,
		Timestamp:   oneAndHalfYearsAgo,
	}

	return []*pkg.Contract{
		&lookerContract1,
		&lookerContract2,
		&lookerContract3,
		&lookerContract4,
		&lookerContract5,
	}
}

func getDummyInvoiceRows() domain.ProductInvoiceRows {
	invoiceMonth := getInvoiceMonth()

	entity := &firestore.DocumentRef{
		Parent: &firestore.CollectionRef{
			ID:   "buckets",
			Path: "buckets/0000",
		},
		ID: "entities/0000",
	}
	bucket := &firestore.DocumentRef{
		ID: "looker-bucket",
	}

	resForContracts := domain.ProductInvoiceRows{
		Type:  lookerUtils.LookerProductType,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}
	// row for contract 1: a 3-month period:
	var deferredRevPeriod1 = &domain.DeferredRevenuePeriod{
		StartDate: invoiceMonth,
		EndDate:   invoiceMonth.AddDate(0, 3, 0),
	}

	threeMonthsAgo := getThreeMonthsAgo()
	if threeMonthsAgo.AddDate(0, 3, 0).Month() == invoiceMonth.Month() {
		resForContracts.Rows = append(resForContracts.Rows, &domain.InvoiceRow{
			Discount:              0,
			Description:           "Google Looker",
			Details:               "Add-On Instance - Customer Hosted",
			Quantity:              1,
			PPU:                   300,
			Currency:              string(fixer.USD),
			Total:                 300,
			SKU:                   "14EB-3C03-C96B",
			Rank:                  1,
			Type:                  lookerUtils.LookerProductType,
			Final:                 true,
			Entity:                entity,
			Bucket:                bucket,
			DeferredRevenuePeriod: deferredRevPeriod1,
		})
	}

	// row for contract 2: a 1-month period:

	resForContracts.Rows = append(resForContracts.Rows, &domain.InvoiceRow{
		Discount:    0,
		Description: "Google Looker",
		Details:     "Add-On Instance - Customer Hosted",
		Quantity:    1,
		PPU:         100,
		Currency:    string(fixer.USD),
		Total:       100,
		SKU:         "14EB-3C03-C96B",
		Rank:        1,
		Type:        lookerUtils.LookerProductType,
		Final:       true,
		Entity:      entity,
		Bucket:      bucket,
	})

	var deferredRevPeriod2 = &domain.DeferredRevenuePeriod{
		StartDate: invoiceMonth,
		EndDate:   invoiceMonth.AddDate(0, 12, 0),
	}
	// row for contract 3: a 1-year period:
	resForContracts.Rows = append(resForContracts.Rows, &domain.InvoiceRow{
		Discount:              0,
		Description:           "Google Looker",
		Details:               "Add-On Instance - Customer Hosted",
		Quantity:              1,
		PPU:                   1200,
		Currency:              string(fixer.USD),
		Total:                 1200,
		SKU:                   "14EB-3C03-C96B",
		Rank:                  1,
		Type:                  lookerUtils.LookerProductType,
		Final:                 true,
		Entity:                entity,
		Bucket:                bucket,
		DeferredRevenuePeriod: deferredRevPeriod2,
	})

	return resForContracts
}

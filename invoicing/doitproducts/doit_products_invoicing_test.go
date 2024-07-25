package doitproducts

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestDoITPackageInvoicingService_getDoITPackageInvoice(t *testing.T) {
	ctx := context.Background()
	time1 := time.Date(2024, 01, 10, 14, 5, 5, 0, time.UTC)
	time2 := time.Date(2024, 01, 15, 14, 5, 5, 0, time.UTC)
	USD := "USD"

	contractIDMap10 := map[string]*pkg.Contract{
		"solve1": &pkg.Contract{
			ID:              "solve1",
			Type:            "solve",
			Customer:        &firestore.DocumentRef{ID: "C1"},
			Entity:          &firestore.DocumentRef{ID: "E1"},
			Discount:        0,
			StartDate:       &time1,
			EndDate:         &time2,
			Timestamp:       time.Time{},
			TimeCreated:     time.Time{},
			Tier:            &firestore.DocumentRef{ID: "T1"},
			PointOfSale:     "doit",
			PaymentTerm:     "monthly",
			MonthlyFlatRate: 1.55,
		},
	}

	contractIDMap11 := map[string]*pkg.Contract{
		"solve1": &pkg.Contract{
			ID:              "solve1",
			Type:            "solve",
			Customer:        &firestore.DocumentRef{ID: "C1"},
			Entity:          &firestore.DocumentRef{ID: "E1"},
			Discount:        3,
			StartDate:       &time1,
			EndDate:         &time2,
			Timestamp:       time.Time{},
			TimeCreated:     time.Time{},
			Tier:            &firestore.DocumentRef{ID: "T1"},
			PointOfSale:     "doit",
			PaymentTerm:     "monthly",
			MonthlyFlatRate: 1.55,
		},
	}

	contractIDMap20 := map[string]*pkg.Contract{
		"solve2": &pkg.Contract{
			ID:              "solve2",
			Type:            "solve",
			Customer:        &firestore.DocumentRef{ID: "C2"},
			Entity:          &firestore.DocumentRef{ID: "E2"},
			Discount:        0,
			StartDate:       &time1,
			Timestamp:       time.Time{},
			TimeCreated:     time.Time{},
			Tier:            &firestore.DocumentRef{ID: "T1"},
			PointOfSale:     "doit",
			PaymentTerm:     "annual",
			MonthlyFlatRate: 1.55,
		},
	}

	contractIDMap21 := map[string]*pkg.Contract{
		"solve2": &pkg.Contract{
			ID:              "solve2",
			Type:            "solve",
			Customer:        &firestore.DocumentRef{ID: "C2"},
			Entity:          &firestore.DocumentRef{ID: "E2"},
			Discount:        5.5,
			StartDate:       &time1,
			Timestamp:       time.Time{},
			TimeCreated:     time.Time{},
			Tier:            &firestore.DocumentRef{ID: "T1"},
			PointOfSale:     "doit",
			PaymentTerm:     "annual",
			MonthlyFlatRate: 1.55,
		},
	}

	tier := &Tier{
		Description: "Solve Std",
		DisplayName: "solve std",
		Name:        "standard",
		PackageType: "solve",
		Type:        "preset",
	}

	type fields struct {
		private mockDoitPackagePrivate
	}
	type args struct {
		task         *domain.CustomerTaskData
		customerRef  *firestore.DocumentRef
		entities     map[string]*common.Entity
		contractType string
	}
	tests := []struct {
		name string
		args args
		on   func(f *fields)
		want func(rows *domain.ProductInvoiceRows) bool
	}{
		{
			name: "basic test monthly contract",
			args: args{
				task: &domain.CustomerTaskData{
					CustomerID:   "C1",
					Now:          time.Date(2024, 01, 20, 1, 0, 0, 0, time.UTC),
					InvoiceMonth: time.Date(2024, 01, 31, 11, 0, 0, 0, time.UTC),
					Rates:        map[string]float64{"USD": 1},
					TimeIndex:    -2,
				},
				customerRef: &firestore.DocumentRef{
					ID: "C1",
				},
				entities: map[string]*common.Entity{"E1": &common.Entity{
					Name:     "E1",
					Currency: &USD,
				}},
				contractType: "solve",
			},
			on: func(f *fields) {
				f.private.On("customerContractsData", ctx, mock.Anything, mock.Anything, "solve").
					Return(contractIDMap10, nil)
				f.private.On("contractBillingData", ctx, mock.Anything).Return(
					map[string]interface{}{
						"2024-01-10": DoiTPackageServicesRow{
							BaseFee: 12.0,
							Consumption: []DoiTPackageConsumption{DoiTPackageConsumption{
								Cloud:       "amazon-web-services",
								VariableFee: 101.20,
								Currency:    "USD",
								Final:       true,
							},
							},
						},
						"2024-01-11": DoiTPackageServicesRow{
							BaseFee:     0,
							Consumption: nil,
						},
						"final":          true,
						"lastUpdateDate": "2024-01-10",
					}, nil)
				f.private.On("contractTierData", ctx, mock.Anything).Return(tier, nil)
				f.private.On("contractContext", ctx, mock.Anything).Return(nil, fmt.Errorf("bla blah"))

			},
			want: func(rows *domain.ProductInvoiceRows) bool {
				firstRow := rows.Rows[0]
				secondRow := rows.Rows[1]

				if !strings.Contains(rows.Rows[0].Description, "Subscription") {
					firstRow = rows.Rows[1]
					secondRow = rows.Rows[0]
				}

				returnValue := true
				returnValue = returnValue && (firstRow.Description == "DoiT Cloud Solve Std - Subscription" &&
					secondRow.Description == "DoiT Cloud Solve Std - 1.55% cloud spend (AWS)" &&
					firstRow.Total == 12.0 && secondRow.Total == 101.20)

				returnValue = returnValue && firstRow.Discount == 0

				returnValue = returnValue && firstRow.Final == true && secondRow.Final == true

				return returnValue
			},
		},
		{
			name: "basic test monthly contract with discount",
			args: args{
				task: &domain.CustomerTaskData{
					CustomerID:   "C1",
					Now:          time.Date(2024, 01, 20, 1, 0, 0, 0, time.UTC),
					InvoiceMonth: time.Date(2024, 01, 31, 11, 0, 0, 0, time.UTC),
					Rates:        map[string]float64{"USD": 1},
					TimeIndex:    -2,
				},
				customerRef: &firestore.DocumentRef{
					ID: "C1",
				},
				entities: map[string]*common.Entity{"E1": &common.Entity{
					Name:     "E1",
					Currency: &USD,
				}},
				contractType: "solve",
			},
			on: func(f *fields) {
				f.private.On("customerContractsData", ctx, mock.Anything, mock.Anything, "solve").
					Return(contractIDMap11, nil)
				f.private.On("contractBillingData", ctx, mock.Anything).Return(
					map[string]interface{}{
						"2024-01-10": DoiTPackageServicesRow{
							BaseFee: 12.0,
							Consumption: []DoiTPackageConsumption{DoiTPackageConsumption{
								Cloud:       "amazon-web-services",
								VariableFee: 101.20,
								Currency:    "USD",
								Final:       false,
							},
							},
						},
						"2024-01-11": DoiTPackageServicesRow{
							BaseFee:     0,
							Consumption: nil,
						},
						"final":          false,
						"lastUpdateDate": "2024-01-10",
					}, nil)
				f.private.On("contractTierData", ctx, mock.Anything).Return(tier, nil)
				f.private.On("contractContext", ctx, mock.Anything).Return(nil, fmt.Errorf("bla blah"))
			},
			want: func(rows *domain.ProductInvoiceRows) bool {
				firstRow := rows.Rows[0]
				secondRow := rows.Rows[1]

				if !strings.Contains(rows.Rows[0].Description, "Subscription") {
					firstRow = rows.Rows[1]
					secondRow = rows.Rows[0]
				}

				returnValue := true
				returnValue = returnValue && (firstRow.Description == "DoiT Cloud Solve Std - Subscription with 3.00% discount" &&
					secondRow.Description == "DoiT Cloud Solve Std - 1.55% cloud spend (AWS)" &&
					firstRow.Total == 12.0 && secondRow.Total == 101.20)

				returnValue = returnValue && firstRow.Discount == 0

				return returnValue
			},
		},
		{
			name: "basic test annual contract",
			args: args{
				task: &domain.CustomerTaskData{
					CustomerID:   "C2",
					Now:          time.Date(2024, 01, 20, 1, 0, 0, 0, time.UTC),
					InvoiceMonth: time.Date(2024, 01, 31, 11, 0, 0, 0, time.UTC),
					Rates:        map[string]float64{"USD": 1},
					TimeIndex:    -2,
				},
				customerRef: &firestore.DocumentRef{
					ID: "C2",
				},
				entities: map[string]*common.Entity{"E2": &common.Entity{
					Name:     "E2",
					Currency: &USD,
				}},
				contractType: "solve",
			},
			on: func(f *fields) {
				f.private.On("customerContractsData", ctx, mock.Anything, mock.Anything, "solve").
					Return(contractIDMap20, nil)
				f.private.On("contractBillingData", ctx, mock.Anything).Return(
					map[string]interface{}{
						"2024-01-10": DoiTPackageServicesRow{
							BaseFee: 12.0,
							Consumption: []DoiTPackageConsumption{DoiTPackageConsumption{
								Cloud:       "amazon-web-services",
								VariableFee: 101.20,
								Currency:    "USD",
								Final:       false,
							},
							},
						},
						"2024-01-11": DoiTPackageServicesRow{
							BaseFee:     0,
							Consumption: nil,
						},
						"final":          false,
						"lastUpdateDate": "2024-01-10",
					}, nil)
				f.private.On("contractTierData", ctx, mock.Anything).Return(tier, nil)
				f.private.On("contractContext", ctx, mock.Anything).Return(nil, fmt.Errorf("bla blah"))
			},
			want: func(rows *domain.ProductInvoiceRows) bool {
				firstRow := rows.Rows[0]
				secondRow := rows.Rows[1]

				if !strings.Contains(rows.Rows[0].Description, "Subscription") {
					firstRow = rows.Rows[1]
					secondRow = rows.Rows[0]
				}

				returnValue := true
				returnValue = returnValue && (firstRow.Description == "DoiT Cloud Solve Std - Subscription - Annual" &&
					secondRow.Description == "DoiT Cloud Solve Std - 1.55% cloud spend (AWS) - Annual" &&
					firstRow.Total == 12.0 && secondRow.Total == 101.20)

				expectedStartDay := time.Date(firstRow.DeferredRevenuePeriod.StartDate.Year(),
					firstRow.DeferredRevenuePeriod.StartDate.Month(),
					firstRow.DeferredRevenuePeriod.StartDate.Day(), 0, 0, 0, 0, time.UTC)

				expectedEndDay := time.Date(firstRow.DeferredRevenuePeriod.StartDate.Year()+1,
					firstRow.DeferredRevenuePeriod.StartDate.Month(),
					firstRow.DeferredRevenuePeriod.StartDate.Day()-1, 0, 0, 0, 0, time.UTC)

				returnValue = returnValue && firstRow.DeferredRevenuePeriod.StartDate == expectedStartDay &&
					firstRow.DeferredRevenuePeriod.EndDate == expectedEndDay

				returnValue = returnValue && firstRow.Discount == 0

				returnValue = returnValue && firstRow.Details == "Period of 2024/01/10 to 2025/01/09"
				returnValue = returnValue && secondRow.Details == "Period of 2024/01/10 to 2024/01/31"

				return returnValue
			},
		},
		{
			name: "basic test annual contract with discount",
			args: args{
				task: &domain.CustomerTaskData{
					CustomerID:   "C2",
					Now:          time.Date(2024, 01, 20, 1, 0, 0, 0, time.UTC),
					InvoiceMonth: time.Date(2024, 01, 31, 11, 0, 0, 0, time.UTC),
					Rates:        map[string]float64{"USD": 1},
					TimeIndex:    -2,
				},
				customerRef: &firestore.DocumentRef{
					ID: "C2",
				},
				entities: map[string]*common.Entity{"E2": &common.Entity{
					Name:     "E2",
					Currency: &USD,
				}},
				contractType: "solve",
			},
			on: func(f *fields) {
				f.private.On("customerContractsData", ctx, mock.Anything, mock.Anything, "solve").
					Return(contractIDMap21, nil)
				f.private.On("contractBillingData", ctx, mock.Anything).Return(
					map[string]interface{}{
						"2024-01-10": DoiTPackageServicesRow{
							BaseFee: 12.0,
							Consumption: []DoiTPackageConsumption{DoiTPackageConsumption{
								Cloud:       "amazon-web-services",
								VariableFee: 101.20,
								Currency:    "USD",
								Final:       false,
							},
							},
						},
						"2024-01-11": DoiTPackageServicesRow{
							BaseFee:     0,
							Consumption: nil,
						},
						"final":          false,
						"lastUpdateDate": "2024-01-10",
					}, nil)
				f.private.On("contractTierData", ctx, mock.Anything).Return(tier, nil)
				f.private.On("contractContext", ctx, mock.Anything).Return(nil, nil)
			},
			want: func(rows *domain.ProductInvoiceRows) bool {
				firstRow := rows.Rows[0]
				secondRow := rows.Rows[1]

				if !strings.Contains(rows.Rows[0].Description, "Subscription") {
					firstRow = rows.Rows[1]
					secondRow = rows.Rows[0]
				}

				returnValue := true
				returnValue = returnValue && (firstRow.Description == "DoiT Cloud Solve Std - Subscription - Annual with 5.50% discount" &&
					secondRow.Description == "DoiT Cloud Solve Std - 1.55% cloud spend (AWS) - Annual" &&
					firstRow.Total == 12.0 && secondRow.Total == 101.20)

				expectedStartDay := time.Date(firstRow.DeferredRevenuePeriod.StartDate.Year(),
					firstRow.DeferredRevenuePeriod.StartDate.Month(),
					firstRow.DeferredRevenuePeriod.StartDate.Day(), 0, 0, 0, 0, time.UTC)

				expectedEndDay := time.Date(firstRow.DeferredRevenuePeriod.StartDate.Year()+1,
					firstRow.DeferredRevenuePeriod.StartDate.Month(),
					firstRow.DeferredRevenuePeriod.StartDate.Day()-1, 0, 0, 0, 0, time.UTC)

				returnValue = returnValue && firstRow.DeferredRevenuePeriod.StartDate == expectedStartDay &&
					firstRow.DeferredRevenuePeriod.EndDate == expectedEndDay

				returnValue = returnValue && firstRow.Discount == 0

				returnValue = returnValue && firstRow.Details == "Period of 2024/01/10 to 2025/01/09"
				returnValue = returnValue && secondRow.Details == "Period of 2024/01/10 to 2024/01/31"

				return returnValue
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{}
			tt.on(&f)

			s := &DoITPackageInvoicingService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return logger.FromContext(ctx)
				},
				private: &f.private,
			}

			if got := s.getDoITPackageInvoice(ctx, tt.args.task, tt.args.customerRef, tt.args.entities, tt.args.contractType); !tt.want(got) {
				t.Errorf("getDoITPackageInvoice() = %v", got)
			}
		})
	}
}

func TestIntegrationWithContractInDevEnvironment(t *testing.T) {
	t.Skip("integration Test, depends on dev firestore state, only enable when testing locally")

	ctx := context.Background()
	loggerProvider := func(ctx context.Context) logger.ILogger {
		return logger.FromContext(ctx)
	}

	cli, err := firestore.NewClient(ctx, "doitintl-cmp-dev")
	if err != nil {
		assert.Fail(t, "failed to connect fs")
	}

	conn := connection.Connection{}
	ctx = context.WithValue(ctx, connection.CtxFirestoreKey, cli)
	testService, _ := NewDoITPackageService(loggerProvider, &conn)

	testParameters := []struct {
		customerID   string
		contractType string
		contractID   string
		rowType      string
		invoiceMonth time.Time
		description  string
		sku          string
		rank         int
	}{
		{customerID: "fSjxgVOn6vkMJgwbdPGt", contractType: "navigator", contractID: "2VSkQizSXcYDF1iKXE7L",
			invoiceMonth: time.Date(2024, 3, 31, 23, 59, 0, 0, time.UTC),
			rowType:      "base", description: "DoiT Cloud Navigator Premium - Subscription", sku: "P-PT-M-D-001", rank: 1},
		{customerID: "01Vrnon4EywhUtdIOr4j", contractType: "solve", contractID: "31LhboU1FW2BRZbQt5LQ",
			invoiceMonth: time.Date(2024, 3, 31, 23, 59, 0, 0, time.UTC),
			rowType:      "base", description: "DoiT Cloud Solve Standard - Subscription", sku: "S-ST-M-D-001", rank: 1},
		{customerID: "01Vrnon4EywhUtdIOr4j", contractType: "solve", contractID: "31LhboU1FW2BRZbQt5LQ",
			invoiceMonth: time.Date(2024, 3, 31, 23, 59, 0, 0, time.UTC),
			rowType:      "google-cloud", description: "DoiT Cloud Solve Standard - 5.00% cloud spend (GCP)", sku: "S-ST-M-D-002", rank: 3},
	}

	for _, testCase := range testParameters {
		contractData, err := testService.private.
			customerContractsData(ctx, cli.Collection("customers").
				Doc(testCase.customerID), testCase.invoiceMonth, testCase.contractType)
		if err != nil {
			assert.Fail(t, "failed to read customerContractdsData for %v", testCase.customerID)
		}
		assert.True(t, len(contractData) > 0)
		contractsFound := 0

		entities := map[string]*common.Entity{}
		for contractID, contract := range contractData {
			if contractID != testCase.contractID {
				continue
			}

			contractsFound++

			entitySnap, err := cli.Collection("entities").Doc(contract.Entity.ID).Get(ctx)
			if err != nil {
				assert.Fail(t, "contract has no entity")
			}
			var entity common.Entity

			err = entitySnap.DataTo(&entity)
			if err != nil {
				assert.Fail(t, "contract incorrect data")
			}

			entities[contract.Entity.ID] = &entity

			rows, err := testService.customerDOITPackageCostBillingData(ctx, contract, entities, testCase.invoiceMonth, map[string]float64{})

			assert.Nilf(t, err, "error when fetching billingData %v", err)
			assert.True(t, rows[contractID][testCase.rowType].Description == testCase.description, "description does not match, wanted %v , found %v", testCase.description, rows[contractID][testCase.rowType].Description)
			assert.True(t, rows[contractID][testCase.rowType].SKU == testCase.sku, "sku does not match, wanted %v , found %v", testCase.sku, rows[contractID][testCase.rowType].SKU)
			assert.True(t, rows[contractID][testCase.rowType].Rank == testCase.rank, "rank does not match, wanted %v , found %v", testCase.rank, rows[contractID][testCase.rowType].Rank)

			break
		}

		assert.True(t, contractsFound == 1, "no contracts found")
	}
}

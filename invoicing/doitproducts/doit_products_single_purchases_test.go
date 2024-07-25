package doitproducts

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

func TestDoITPackageInvoicingService_getDoITPackageSinglePurchaseInvoice(t *testing.T) {
	ctx := context.Background()
	time1 := time.Date(2024, 01, 10, 14, 5, 5, 0, time.UTC)
	time2 := time.Date(2024, 01, 15, 14, 5, 5, 0, time.UTC)
	USD := "USD"

	contractIDMap10 := map[string]*pkg.Contract{
		"solve1": &pkg.Contract{
			ID:          "solve-accelerator1",
			Type:        "solve-accelerator",
			Customer:    &firestore.DocumentRef{ID: "C1"},
			Entity:      &firestore.DocumentRef{ID: "E1"},
			Discount:    0,
			StartDate:   &time1,
			EndDate:     &time2,
			Timestamp:   time.Time{},
			TimeCreated: time.Time{},
			PointOfSale: "doit",
			PaymentTerm: "monthly",
			Properties:  map[string]interface{}{"typeContext": &firestore.DocumentRef{ID: "typeContextRef1"}},
		},
	}

	contractIDMap11 := map[string]*pkg.Contract{
		"solve1": &pkg.Contract{
			ID:          "solve-accelerator2",
			Type:        "solve-accelerator",
			Customer:    &firestore.DocumentRef{ID: "C1"},
			Entity:      &firestore.DocumentRef{ID: "E1"},
			Discount:    3,
			StartDate:   &time1,
			EndDate:     &time2,
			Timestamp:   time.Time{},
			TimeCreated: time.Time{},
			PointOfSale: "doit",
			PaymentTerm: "monthly",
			Properties:  map[string]interface{}{"typeContext": &firestore.DocumentRef{ID: "typeContextRef2"}},
		},
	}

	typeContext1 := &ContractContext{
		Cloud:     "cloud1",
		Label:     "bla bla1",
		SkuNumber: "087",
	}

	typeContext2 := &ContractContext{
		Cloud:     "cloud2",
		Label:     "bla bla2",
		SkuNumber: "097",
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
				contractType: "solve-accelerator",
			},
			on: func(f *fields) {
				f.private.On("customerSingleSaleContracts", ctx, mock.Anything, mock.Anything, "solve-accelerator").
					Return(contractIDMap10, nil)
				f.private.On("contractBillingData", ctx, mock.Anything).Return(
					map[string]interface{}{
						"2024-01-10": DoiTPackageServicesRow{
							Consumption: nil,
							BaseFee:     12.0,
						},
						"2024-01-11": DoiTPackageServicesRow{
							BaseFee:     0,
							Consumption: nil,
						},
						"final":          true,
						"lastUpdateDate": "2024-01-10",
					}, nil)
				f.private.On("contractTierData", ctx, mock.Anything).Return(nil, nil)
				f.private.On("contractContext", ctx, mock.Anything).Return(typeContext1, nil)

			},
			want: func(rows *domain.ProductInvoiceRows) bool {
				firstRow := rows.Rows[0]

				returnValue := true
				returnValue = returnValue && (firstRow.Description == "Accelerator: bla bla1" &&
					firstRow.Total == 12.0)

				returnValue = returnValue && firstRow.Discount == 0

				returnValue = returnValue && firstRow.Final == true

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
				contractType: "solve-accelerator",
			},
			on: func(f *fields) {
				f.private.On("customerSingleSaleContracts", ctx, mock.Anything, mock.Anything, "solve-accelerator").
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
				f.private.On("contractTierData", ctx, mock.Anything).Return(nil, fmt.Errorf("bla blah"))
				f.private.On("contractContext", ctx, mock.Anything).Return(typeContext2, nil)
			},
			want: func(rows *domain.ProductInvoiceRows) bool {
				firstRow := rows.Rows[0]

				returnValue := true
				returnValue = returnValue && (firstRow.Description == "Accelerator: bla bla2 with 3.00% discount" &&
					firstRow.Total == 12.0)

				returnValue = returnValue && firstRow.Discount == 0

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

			if got := s.getDoITPackageSinglePurchaseInvoice(ctx, tt.args.task, tt.args.customerRef, tt.args.entities, tt.args.contractType); !tt.want(got) {
				t.Errorf("getDoITPackageSinglePurchaseInvoice() = %v", got)
			}
		})
	}
}

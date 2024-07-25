package invoicing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/gsuite"
)

func TestGSuite_getGSuitePrice(t *testing.T) {
	type args struct {
		invItem     gsuite.InventoryItem
		catalogItem CatalogItem
		currency    string
	}

	testPrevPriceEndDate20230416 := time.Date(2023, 4, 16, 0, 0, 0, 0, time.UTC)

	usd := 60.0
	eur := 5.75
	gbp := 5.0
	aud := 8.4
	brl := 35.0
	nok := 59.0
	dkk := 44.6

	prevPrice := &CatalogItemPrice{
		USD: &usd,
		EUR: &eur,
		GBP: &gbp,
		AUD: &aud,
		BRL: &brl,
		NOK: &nok,
		DKK: &dkk,
	}

	price := CatalogItemPrice{
		USD: common.Float(1.2 * usd),
		EUR: common.Float(1.2 * eur),
		GBP: common.Float(1.2 * gbp),
		AUD: common.Float(1.2 * aud),
		BRL: common.Float(1.2 * brl),
		NOK: common.Float(1.2 * nok),
		DKK: common.Float(1.2 * dkk),
	}

	tests := []struct {
		name         string
		args         args
		want         float64
		wantPriceKey string
		wantErr      error
	}{
		{
			name: "1. catalog item is 'gsuite enterprise', creation date is before hardcoded 'newPricesStartDate', plan: 'annual', payment: 'yearly', currency: 'GBP'",
			args: args{
				invItem: gsuite.InventoryItem{
					Subscription: &gsuite.Subscription{
						CreationTime: newPricesStartDate20210116.AddDate(0, 0, -1).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:   gsuite.GSuiteEnterprise,
					Plan:    PlanAnnual,
					Payment: PaymentYearly,
				},
				currency: "GBP",
			},
			want:         GSuiteEnterpriseYearlyGBP,
			wantPriceKey: priceKey20210116,
			wantErr:      nil,
		},
		{
			name: "2. catalog item is 'gsuite enterprise', creation date is before hardcoded 'newPricesStartDate', plan: 'annual', payment: 'monthly', currency: 'USD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Subscription: &gsuite.Subscription{
						CreationTime: newPricesStartDate20210116.AddDate(0, 0, -1).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:   gsuite.GSuiteEnterprise,
					Plan:    PlanAnnual,
					Payment: PaymentMonthly,
				},
				currency: "USD",
			},
			want:         GSuiteEnterpriseMonthlyUSD,
			wantPriceKey: priceKey20210116,
			wantErr:      nil,
		},
		{
			name: "3. catalog item is 'gsuite enterprise', creation date is after hardcoded 'newPricesStartDate' and before 'catalog.PrevPriceEndDate', plan: 'annual', payment: 'monthly', currency: 'USD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Subscription: &gsuite.Subscription{
						CreationTime: newPricesStartDate20210116.AddDate(0, 0, 2).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GSuiteEnterprise,
					Plan:             PlanAnnual,
					Payment:          PaymentMonthly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        prevPrice,
					Price:            price,
				},
				currency: "USD",
			},
			want:         *prevPrice.USD,
			wantPriceKey: priceKeyPrev,
			wantErr:      nil,
		},
		{
			name: "4. catalog item is 'gsuite enterprise', creation date is after hardcoded 'newPricesStartDate' and after 'catalog.PrevPriceEndDate', plan: 'annual', payment: 'monthly', currency: 'EUR'",
			args: args{
				invItem: gsuite.InventoryItem{
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, 2).UnixMilli(),
					},
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 1),
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GSuiteEnterprise,
					Plan:             PlanAnnual,
					Payment:          PaymentMonthly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        prevPrice,
					Price:            price,
				},
				currency: "USD",
			},
			want:         *price.USD,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
		{
			name: "5. catalog item is 'gsuite cloud identity premium', creation date is before hardcoded 'newPricesStartDate', plan: 'annual', payment: 'yearly', currency: 'GBP'",
			args: args{
				invItem: gsuite.InventoryItem{
					Subscription: &gsuite.Subscription{
						CreationTime: newPricesStartDate20210116.AddDate(0, 0, -1).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:     gsuite.CloudIdentityPremium,
					Plan:      PlanAnnual,
					Payment:   PaymentYearly,
					PrevPrice: prevPrice,
					Price:     price,
				},
				currency: "GBP",
			},
			want:         *price.GBP * 0.665,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
		{
			name: "6. catalog item is a regular item (no extra conditions), creation date is before 'catalog.PrevPriceEndDate', plan: 'annual', payment: 'yearly', currency: 'AUD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 5),
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, -5).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GoogleVault,
					Plan:             PlanAnnual,
					Payment:          PaymentYearly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        prevPrice,
					Price:            price,
				},
				currency: "AUD",
			},
			want:         *prevPrice.AUD,
			wantPriceKey: priceKeyPrev,
			wantErr:      nil,
		},
		{
			name: "7. catalog item is a regular item (no extra conditions), creation date is after 'catalog.PrevPriceEndDate', plan: 'annual', payment: 'yearly', currency: 'AUD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 5),
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, 5).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GoogleVault,
					Plan:             PlanAnnual,
					Payment:          PaymentYearly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        prevPrice,
					Price:            price,
				},
				currency: "AUD",
			},
			want:         *price.AUD,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
		{
			name: "8. catalog item is a regular item (no extra conditions), inventory date is after 'catalog.PrevPriceEndDate', plan: 'flexible', payment: 'monthly', currency: 'AUD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 10),
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, -5).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GoogleVault,
					Plan:             PlanFlexible,
					Payment:          PaymentMonthly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        prevPrice,
					Price:            price,
				},
				currency: "AUD",
			},
			want:         *price.AUD,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
		{
			name: "9. catalog item is a regular item (no extra conditions), catalog.PrevPriceEndDate is nil, plan: 'annual', payment: 'yearly', currency: 'AUD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 10),
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, -1).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:     gsuite.GoogleVault,
					Plan:      PlanAnnual,
					Payment:   PaymentYearly,
					PrevPrice: prevPrice,
					Price:     price,
				},
				currency: "AUD",
			},
			want:         *price.AUD,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
		{
			name: "10. catalog item is a regular item (no extra conditions), catalog.PrevPrice is nil, plan: 'annual', payment: 'yearly', currency: 'AUD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 10),
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, -1).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GoogleVault,
					Plan:             PlanAnnual,
					Payment:          PaymentYearly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        nil,
					Price:            price,
				},
				currency: "AUD",
			},
			want:         *price.AUD,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
		{
			name: "11. catalog item is a regular item (no extra conditions), creation time after catalog.PrevPriceEndDate, plan: 'annual', payment: 'yearly', currency: 'AUD'",
			args: args{
				invItem: gsuite.InventoryItem{
					Date: testPrevPriceEndDate20230416.AddDate(0, 0, 10),
					Subscription: &gsuite.Subscription{
						CreationTime: testPrevPriceEndDate20230416.AddDate(0, 0, 12).UnixMilli(),
					},
				},
				catalogItem: CatalogItem{
					SkuID:            gsuite.GoogleVault,
					Plan:             PlanAnnual,
					Payment:          PaymentYearly,
					PrevPriceEndDate: &testPrevPriceEndDate20230416,
					PrevPrice:        prevPrice,
					Price:            price,
				},
				currency: "AUD",
			},
			want:         *price.AUD,
			wantPriceKey: priceKeyCurr,
			wantErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, priceKey, err := getGSuitePrice(
				&tt.args.invItem,
				&tt.args.catalogItem,
				tt.args.currency,
			)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, price)
			assert.Equal(t, tt.wantPriceKey, priceKey)
		})
	}
}

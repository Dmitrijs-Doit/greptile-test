package dal

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	customers "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/times"
	testPackage "github.com/doitintl/tests"
)

const (
	testInvoiceMonth = "2022-01-01"
)

func TestNewStandaloneFirestoreDAL(t *testing.T) {
	_, err := NewFlexsaveStandaloneFirestore(context.Background(), common.TestProjectID)
	assert.NoError(t, err)

	d := NewFlexsaveStandaloneFirestoreWithClient(nil)
	assert.NotNil(t, d)
}

func TestFlexsaveStandaloneFirestore_BatchSetFlexsaveBillingData(t *testing.T) {
	type args struct {
		ctx      context.Context
		assetMap map[string]pkg.MonthlyBillingFlexsaveStandalone
	}

	ctx := context.Background()

	if err := testPackage.LoadTestData("Assets"); err != nil {
		t.Error(err)
	}

	d, err := NewFlexsaveStandaloneFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	customerDAL, err := customers.NewCustomersFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	var assetMap = map[string]pkg.MonthlyBillingFlexsaveStandalone{
		"amazon-web-services-standalone-023946476650": {
			Customer: customerDAL.GetRef(ctx, "bWBwhhI3gIRZre40dhz8"),
			Spend: map[string]float64{
				"023946476650": 2.0,
			},
			Type:         common.Assets.AmazonWebServicesStandalone,
			InvoiceMonth: "2022-01",
		},
		"amazon-web-services-standalone-538692396177": {
			Customer: customerDAL.GetRef(ctx, "Lv86gBf5roruvKGB5poN"),
			Spend: map[string]float64{
				"538692396177": 242.1,
			},
			Type:         common.Assets.AmazonWebServicesStandalone,
			InvoiceMonth: "2022-02",
		},
	}

	var invalidAssetMap = map[string]pkg.MonthlyBillingFlexsaveStandalone{
		"amazon-web-services-standalone-023946476650": {
			Customer: &firestore.DocumentRef{ID: "test_customer"},
			Spend: map[string]float64{
				"023946476650": 2.0,
			},
			Type:         common.Assets.AmazonWebServicesStandalone,
			InvoiceMonth: "2022-01",
		},
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:      ctx,
				assetMap: assetMap,
			},
		},
		{
			name: "invalid reference commit error",
			args: args{
				ctx:      ctx,
				assetMap: invalidAssetMap,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := d.BatchSetFlexsaveBillingData(tt.args.ctx, tt.args.assetMap); err != nil {
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					t.Errorf("FlexsaveStandaloneFirestor error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestFlexsaveStandaloneFirestore_GetCustomerStandaloneAWSAssetIDtoMonthlyBillingData(t *testing.T) {
	type args struct {
		ctx          context.Context
		customerRef  *firestore.DocumentRef
		invoiceMonth time.Time
		assetType    string
	}

	if err := testPackage.LoadTestData("BillingData"); err != nil {
		t.Error(err)
	}

	if err := testPackage.LoadTestData("Customers"); err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	parsedMonth, _ := time.Parse(times.YearMonthDayLayout, testInvoiceMonth)

	d, err := NewFlexsaveStandaloneFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	customerDAL, err := customers.NewCustomersFirestore(ctx, "doitintl-cmp-dev")
	if err != nil {
		t.Error(err)
	}

	customerRef := customerDAL.GetRef(ctx, "Lv86gBf5roruvKGB5poN")

	tests := []struct {
		name    string
		args    args
		want    map[string]*pkg.MonthlyBillingFlexsaveStandalone
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:          ctx,
				customerRef:  customerRef,
				invoiceMonth: parsedMonth,
				assetType:    common.Assets.AmazonWebServicesStandalone,
			},
			want: map[string]*pkg.MonthlyBillingFlexsaveStandalone{
				"amazon-web-services-standalone-281727049056": {
					Customer: customerRef,
					Spend: map[string]float64{
						"023946476650": 1948,
					},
					Type:         common.Assets.AmazonWebServicesStandalone,
					InvoiceMonth: testInvoiceMonth[:7],
				},
			},
		},
		{
			name: "invalid customer ref",
			args: args{
				ctx:          ctx,
				customerRef:  &firestore.DocumentRef{ID: "test_customer"},
				invoiceMonth: parsedMonth,
				assetType:    common.Assets.AmazonWebServicesStandalone,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetMap, err := d.GetCustomerStandaloneAssetIDtoMonthlyBillingData(tt.args.ctx, tt.args.customerRef, tt.args.invoiceMonth, common.Assets.AmazonWebServicesStandalone)
			if err != nil {
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					t.Errorf("FlexsaveStandaloneFirestor error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			for assetDocID, billingData := range assetMap {
				expected, ok := tt.want[assetDocID]
				// key exists
				assert.Equal(t, true, ok)
				// properties are equal
				assert.Equal(t, expected.Customer.ID, billingData.Customer.ID)
				assert.Equal(t, expected.Spend, billingData.Spend)
				assert.Equal(t, expected.Type, billingData.Type)
				assert.Equal(t, expected.InvoiceMonth, billingData.InvoiceMonth)
			}
		})
	}
}

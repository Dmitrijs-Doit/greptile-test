package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
)

type FlexsaveStandalone interface {
	BatchSetFlexsaveBillingData(ctx context.Context, assetRefMap map[string]pkg.MonthlyBillingFlexsaveStandalone) error
	GetCustomerStandaloneAssetIDtoMonthlyBillingData(ctx context.Context, customerRef *firestore.DocumentRef, invoiceMonth time.Time, assetType string) (map[string]*pkg.MonthlyBillingFlexsaveStandalone, error)
}

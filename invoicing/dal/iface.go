package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
)

type MonthlyBillingData interface {
	BatchUpdateMonthlyBillingData(ctx context.Context, invoiceMonthString string, assetIDToBillingDataMap map[*firestore.DocumentRef]interface{}, useAnalyticsCollection bool) error
	GetCustomerAWSAssetIDtoMonthlyBillingData(ctx context.Context, customerRef *firestore.DocumentRef, invoiceMonth time.Time, useAnalyticsCollection bool) (map[string]*pkg.MonthlyBillingAmazonWebServices, error)
}

package invoicing

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
)

func (s *CustomerAssetInvoiceWorker) GetStandaloneInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows, provider string) {
	res := &domain.ProductInvoiceRows{
		Type:  provider,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	var (
		sku             string
		details         func(docID string) string
		finalBillingDay int
	)

	switch provider {
	case common.Assets.AmazonWebServicesStandalone:
		sku = FlexsaveAwsStandadaloneSKU
		details = func(account string) string { return fmt.Sprintf("Account #%s", account) }
		finalBillingDay = common.FinalBillingDayAWS
	case common.Assets.GoogleCloudStandalone:
		sku = FlexsaveGcpStandadaloneSKU
		details = func(project string) string { return fmt.Sprintf("Project '%s'", project) }
		finalBillingDay = common.FinalBillingDayGCP
	default:
		res.Error = errors.New("invalid provider")
		respChan <- res

		return
	}

	final := task.TimeIndex == -2 && task.Now.Day() >= finalBillingDay

	monthlyBillingData, err := s.flexsaveDAL.GetCustomerStandaloneAssetIDtoMonthlyBillingData(ctx, customerRef, task.InvoiceMonth, provider)
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	for docID, data := range monthlyBillingData {
		asset, err := s.assetsDAL.Get(ctx, docID)
		if err != nil {
			res.Error = err
			respChan <- res

			return
		}

		for k, total := range data.Spend {
			qty, value := utils.GetQuantityAndValue(1, total)

			res.Rows = append(res.Rows, &domain.InvoiceRow{
				Description: utils.BillingFlexsaveCostType,
				Details:     details(k),
				Tags:        []string{},
				Quantity:    qty,
				PPU:         value,
				Currency:    "USD",
				Total:       float64(qty) * value,
				SKU:         sku,
				Rank:        1,
				Type:        provider,
				Final:       final,
				Entity:      asset.Entity,
				Bucket:      nil,
			})
		}
	}

	respChan <- res
}

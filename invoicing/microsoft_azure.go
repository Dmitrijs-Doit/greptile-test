package invoicing

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
)

func (s *InvoicingService) customerMicrosoftAzureHandler(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	fs := s.Firestore(ctx)
	l := s.Logger(ctx)

	res := &domain.ProductInvoiceRows{
		Type:  common.Assets.MicrosoftAzure,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	final := task.TimeIndex == -2 && task.Now.Day() >= 6

	monthlyBillingData, err := customerMonthlyBillingMicrosoftAzure(ctx, fs, customerRef, task.InvoiceMonth)
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	l.Info(monthlyBillingData)

	invoiceAdjustments, err := getCustomerInvoiceAdjustments(ctx, customerRef, common.Assets.MicrosoftAzure, task.InvoiceMonth)
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	for docID, data := range monthlyBillingData {
		if task.Now.Sub(data.Timestamp) >= time.Hour*72 {
			l.Warningf("microsoft azure: stale billing data for asset %s", docID)
		}

		assetSettings, err := getMicrosoftAzureAssetSettings(ctx, fs, docID)
		if err != nil {
			res.Error = err
			respChan <- res

			return
		}

		entityRef := assetSettings.Entity
		if entityRef == nil {
			res.Error = fmt.Errorf("%s: unassigned asset on customer %s", docID, task.CustomerID)
			respChan <- res

			return
		}

		entity, prs := entities[entityRef.ID]
		if !prs {
			res.Error = fmt.Errorf("%s: entity %s not found on customer %s", docID, entityRef.ID, task.CustomerID)
			respChan <- res

			return
		}

		if entity.PriorityID == "999999" {
			respChan <- res
			return
		}

		if data.Spend > 0 {
			bucket := entity.Invoicing.Default
			if assetSettings.Bucket != nil {
				bucket = assetSettings.Bucket
			}

			displayName := data.SubscriptionName
			if displayName == "" {
				displayName = data.SubscriptionID
			}

			res.Rows = append(res.Rows, &domain.InvoiceRow{
				Description: "Microsoft Azure",
				Details:     fmt.Sprintf("Subscription %s", displayName),
				Tags:        assetSettings.Tags,
				Quantity:    1,
				PPU:         data.Spend,
				Currency:    "USD",
				Total:       data.Spend,
				SKU:         MicrosoftAzureSKU,
				Rank:        1,
				Type:        common.Assets.MicrosoftAzure,
				Final:       final && data.Verified,
				Entity:      entityRef,
				Bucket:      bucket,
			})
		}
	}

	for _, invoiceAdjustment := range invoiceAdjustments {
		entity, prs := entities[invoiceAdjustment.Entity.ID]
		if !prs {
			err := fmt.Errorf("invalid entity for invoiceAdjustment %s", invoiceAdjustment.Snapshot.Ref.Path)
			res.Error = err
			respChan <- res

			return
		}

		var quantity int64 = 1

		ppu := invoiceAdjustment.Amount
		if ppu < 0 {
			ppu *= -1
			quantity = -1
		}

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: invoiceAdjustment.Description,
			Details:     invoiceAdjustment.Details,
			Quantity:    quantity,
			PPU:         ppu,
			Currency:    invoiceAdjustment.Currency,
			Total:       invoiceAdjustment.Amount,
			SKU:         MicrosoftAzureSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.MicrosoftAzure,
			Final:       true,
			Entity:      invoiceAdjustment.Entity,
			Bucket:      entity.Invoicing.Default,
		})
	}

	respChan <- res
}

func getMicrosoftAzureAssetSettings(ctx context.Context, fs *firestore.Client, docID string) (*common.AssetSettings, error) {
	assetSettingsRef := fs.Collection("assetSettings").Doc(docID)

	assetSettingsDocSnap, err := assetSettingsRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var assetSettings common.AssetSettings
	if err := assetSettingsDocSnap.DataTo(&assetSettings); err != nil {
		return nil, err
	}

	return &assetSettings, nil
}

func customerMonthlyBillingMicrosoftAzure(ctx context.Context, fs *firestore.Client, customerRef *firestore.DocumentRef, invoiceMonth time.Time) (map[string]*MonthlyBillingMicrosoftAzure, error) {
	result := make(map[string]*MonthlyBillingMicrosoftAzure)
	docs, err := fs.CollectionGroup("monthlyBillingData").
		Where("customer", "==", customerRef).
		Where("type", "==", common.Assets.MicrosoftAzure).
		Where("invoiceMonth", "==", invoiceMonth.Format("2006-01")).
		Documents(ctx).GetAll()

	if err != nil {
		return nil, err
	}

	for _, docSnap := range docs {
		var d MonthlyBillingMicrosoftAzure
		if err := docSnap.DataTo(&d); err != nil {
			return nil, err
		}

		result[docSnap.Ref.Parent.Parent.ID] = &d
	}

	return result, nil
}

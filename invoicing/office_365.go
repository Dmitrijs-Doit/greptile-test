package invoicing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

func (s *InvoicingService) customerOffice365Handler(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	l := s.Logger(ctx)
	fs := s.Firestore(ctx)

	res := &domain.ProductInvoiceRows{
		Type:  common.Assets.Office365,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	discountsCache := make(map[string][2]float64)
	assetSettingsCache := make(map[string]*common.AssetSettings)
	ranks := make(map[string]int)
	final := task.TimeIndex == -2
	dailyRate := 1 / float64(task.InvoiceMonth.Day())
	items := make(map[string][]*microsoft.InventoryItem)

	invoiceAdjustments, err := getCustomerInvoiceAdjustments(ctx, customerRef, common.Assets.Office365, task.InvoiceMonth)
	if err != nil {
		l.Errorf("failed to get customer invoice adjustments with error: %s", err)
		res.Error = err
		respChan <- res

		return
	}

	parentCollection := fmt.Sprintf("%s-%s", microsoft.Inventory, common.Assets.Office365)
	docID := fmt.Sprintf("%s-%s", task.InvoiceMonth.Format("2006-01"), common.Assets.Office365)

	docs, err := fs.Collection(microsoft.Inventory).Doc(docID).Collection(parentCollection).
		Where("customer", "==", customerRef).
		Where("subscription.status", "==", "active").
		Where("subscription.isTrial", "==", false).
		Documents(ctx).GetAll()
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	for _, docSnap := range docs {
		var item microsoft.InventoryItem
		if err := docSnap.DataTo(&item); err != nil {
			res.Error = err
			respChan <- res

			return
		}

		assetID := fmt.Sprintf("%s-%s", common.Assets.Office365, item.Subscription.ID)
		items[assetID] = append(items[assetID], &item)

		var as = &common.AssetSettings{}
		if v, prs := assetSettingsCache[assetID]; prs {
			as = v
		} else {
			docSnap, err := fs.Collection("assetSettings").Doc(assetID).Get(ctx)
			if err != nil && status.Code(err) != codes.NotFound {
				res.Error = err
				respChan <- res

				return
			}

			if err := docSnap.DataTo(as); err != nil {
				res.Error = err
				respChan <- res

				return
			}

			assetSettingsCache[assetID] = as
		}
	}

	for assetID, days := range items {
		var prevQuantity int64

		var prevContractID string

		var prevFlexPeriod time.Time

		var rowData = make(map[string]*domain.InvoiceRow)

		as := assetSettingsCache[assetID]

		for i, day := range days {
			if len(days) <= 0 {
				continue
			}

			if day.Subscription.OfferID == common.MicrosoftAzureOfferID ||
				strings.HasPrefix(day.Subscription.OfferID, common.MicrosoftAzurePlanOfferIDPrefix) ||
				day.Subscription.OfferName == "Azure plan" {
				continue
			}

			plan, payment, startDate, endDate, currency, err := parseOffice365InventoryItem(day)
			if err != nil {
				res.Error = err
				respChan <- res

				return
			}

			var quantity = getOffice365Quantity(day)
			if quantity <= 0 {
				continue
			}

			var catalogItem *CatalogItem

			for _, c := range office365Catalog {
				if c.Plan != plan || c.Payment != payment {
					continue
				}

				if c.SkuID == day.Subscription.OfferID {
					catalogItem = c
					break
				}

				// For non-NCE subscriptions, the product type is null
				if day.Subscription.ProductType != nil {
					// The catalog does not include the last part of the SKU (availability ID)
					if strings.HasPrefix(day.Subscription.OfferID, c.SkuID) {
						catalogItem = c
						break
					}
				}
			}

			if catalogItem == nil {
				res.Error = fmt.Errorf("(%s) could not find %s %s %s in catalog", day.Subscription.ID, day.Subscription.OfferID, plan, payment)
				respChan <- res

				return
			}

			var discounts, contractID = utils.GetDiscount(ctx, day.Contract, discountsCache)

			if payment == "MONTHLY" {
				if i == 0 || prevFlexPeriod.IsZero() || quantity != prevQuantity || contractID != prevContractID {
					prevQuantity = quantity
					prevContractID = contractID
					prevFlexPeriod = day.Date
				}

				ppu := getOffice365Price(catalogItem, currency) * dailyRate
				k := fmt.Sprintf("%s#%s#%s#%s#%s#%s", catalogItem.SkuPriority, day.CustomerID, day.Subscription.ID, prevFlexPeriod.Format(Date), currency, contractID)
				total := float64(quantity) * ppu * utils.ToProportion(discounts[0])
				details := fmt.Sprintf("Period of %s to %s for domain %s", prevFlexPeriod.Format(prettyDate), day.Date.Format(prettyDate), day.CustomerDomain)

				if row, prs := rowData[k]; prs {
					row.PPU += ppu
					row.Total += total
					row.Details = details
				} else {
					rank := getCustomerRank(day.CustomerID, day.Subscription.ID, prevFlexPeriod.Format(Date), ranks)
					newRow := &domain.InvoiceRow{
						Description: catalogItem.SkuName,
						Details:     details,
						Tags:        as.Tags,
						Quantity:    quantity,
						PPU:         ppu,
						Discount:    discounts[0],
						Currency:    currency,
						Total:       total,
						SKU:         catalogItem.SkuPriority,
						Rank:        rank,
						Type:        common.Assets.Office365,
						Final:       final,
						Entity:      as.Entity,
						Bucket:      as.Bucket,
					}
					rowData[k] = newRow
				}
			}

			if payment == "YEARLY" {
				var adjustment bool

				var ppu float64

				var increase int64

				if startDate.Equal(day.Date) {
					ppu = getOffice365Price(catalogItem, currency)
					increase = quantity
				} else {
					if i == 0 {
						prevDay, err := getOffice365PreviousDay(ctx, l, fs, day)
						if err != nil {
							res.Error = err
							respChan <- res

							return
						}

						if prevDay != nil {
							prevQuantity = getOffice365Quantity(prevDay)
						}
					} else {
						prevQuantity = getOffice365Quantity(days[i-1])
					}

					if startDate.After(day.Date) {
						res.Error = fmt.Errorf("start date %v is after date %v for %s (%s)", startDate.Format(Date), day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
						respChan <- res

						return
					}

					if prevQuantity == 0 {
						res.Error = fmt.Errorf("no seats count for previous day on %v for %s (%s)", day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
						respChan <- res

						return
					}

					increase = quantity - prevQuantity
					prevQuantity = quantity

					if increase == 0 {
						// No increase in seats, do nothing.
						continue
					}

					if increase < 0 {
						l.Errorf("invalid seats count on %v for %s (%s)", day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
						continue
					}

					adjustment = true

					totalDays := endDate.Sub(startDate).Hours() / 24
					if totalDays < 0 {
						res.Error = fmt.Errorf("subscription has invalid start/end dates for %s (%s)", day.Subscription.ID, day.CustomerDomain)
						respChan <- res

						return
					}

					remainingDays := endDate.Sub(day.Date).Hours() / 24
					if remainingDays < 0 {
						res.Error = fmt.Errorf("subscription was past its end date on %s for %s (%s)", day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
						respChan <- res

						return
					}

					proration := remainingDays / totalDays
					ppu = getOffice365Price(catalogItem, currency) * proration
				}

				k := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%v", catalogItem.SkuPriority, day.CustomerID, day.Subscription.ID, day.Date.Format(Date), currency, contractID, adjustment)

				var details string
				if adjustment {
					details = fmt.Sprintf("Annual adjustment on %s for domain %s", day.Date.Format(prettyDate), day.CustomerDomain)
				} else {
					details = fmt.Sprintf("Annual commitment from %s to %s for domain %s", startDate.Format(prettyDate), endDate.Format(prettyDate), day.CustomerDomain)
				}

				rank := getCustomerRank(day.CustomerID, day.Subscription.ID, day.Date.Format(Date), ranks)
				total := float64(increase) * ppu * utils.ToProportion(discounts[0])
				newRow := &domain.InvoiceRow{
					Description: catalogItem.SkuName,
					Details:     details,
					Tags:        as.Tags,
					Quantity:    increase,
					PPU:         ppu,
					Discount:    discounts[0],
					Currency:    currency,
					Total:       total,
					SKU:         catalogItem.SkuPriority,
					Rank:        rank,
					Type:        common.Assets.Office365,
					Final:       final,
					Entity:      as.Entity,
					Bucket:      as.Bucket,
				}
				rowData[k] = newRow
			}
		}

		for _, row := range rowData {
			res.Rows = append(res.Rows, row)
		}
	}

	for _, invoiceAdjustment := range invoiceAdjustments {
		entity, prs := entities[invoiceAdjustment.Entity.ID]
		if !prs {
			l.Errorf("invalid entity for invoice adjustment %s", invoiceAdjustment.Snapshot.Ref.Path)
			res.Error = fmt.Errorf("invalid entity for invoiceAdjustment %s", invoiceAdjustment.Snapshot.Ref.Path)
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
			SKU:         Office365SKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.Office365,
			Final:       true,
			Entity:      invoiceAdjustment.Entity,
			Bucket:      entity.Invoicing.Default,
		})
	}

	respChan <- res
}

func getOffice365Price(item *CatalogItem, currency string) float64 {
	p, err := pricesCurrencySwitch(currency, item.Price)
	if err != nil {
		return *item.Price.USD
	}

	return p
}

func parseOffice365InventoryItem(day *microsoft.InventoryItem) (string, string, time.Time, time.Time, string, error) {
	var plan = "FLEXIBLE"

	var payment = "MONTHLY"

	var currency = "USD"

	var startDate, endDate time.Time

	if day.Settings != nil {
		if day.Settings.Plan != nil {
			plan = day.Settings.Plan.PlanName

			if day.Settings.Plan.IsCommitmentPlan {
				payment = "YEARLY"
				startDate = common.EpochMillisecondsToTime(day.Settings.Plan.CommitmentInterval.StartTime).UTC().Truncate(dayDuration)
				endDate = common.EpochMillisecondsToTime(day.Settings.Plan.CommitmentInterval.EndTime).UTC().Truncate(dayDuration)
			} else {
				payment = "MONTHLY"
			}
		}

		if day.Settings.Payment != "" {
			payment = day.Settings.Payment
		}

		if day.Settings.Currency != "" {
			currency = day.Settings.Currency
		}
	}

	if plan == "ANNUAL" {
		if startDate.IsZero() || endDate.IsZero() || endDate.Equal(startDate) || endDate.Before(startDate) {
			err := fmt.Errorf("invalid annual subscription start/end dates for %s (%s)", day.SubscriptionID, day.CustomerDomain)
			return plan, payment, startDate, endDate, currency, err
		}
	}

	return plan, payment, startDate, endDate, currency, nil
}

func getOffice365PreviousDay(ctx context.Context, l logger.ILogger, fs *firestore.Client, day *microsoft.InventoryItem) (*microsoft.InventoryItem, error) {
	prevDate := day.Date.Add(-dayDuration)
	fullDateString := prevDate.Format("2006-01-02")
	monthDateString := prevDate.Format("2006-01")

	parentDocID := fmt.Sprintf("%s-%s", monthDateString, common.Assets.Office365)
	parentCollection := fmt.Sprintf("%s-%s", microsoft.Inventory, common.Assets.Office365)
	docID := fmt.Sprintf("%s-%s-%s", fullDateString, common.Assets.Office365, day.Subscription.ID)

	docRef := fs.Collection(microsoft.Inventory).Doc(parentDocID).Collection(parentCollection).Doc(docID)

	docSnap, err := docRef.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		l.Errorf("failed to get previous day %s with error: %s", docID, err)
		return nil, err
	}

	if docSnap != nil && docSnap.Exists() {
		var prevDay microsoft.InventoryItem
		if err := docSnap.DataTo(&prevDay); err != nil {
			l.Errorf("failed to populate previous day %s to struct with error: %s", docID, err)
			return nil, err
		}

		return &prevDay, nil
	}

	return nil, nil
}

func getOffice365Quantity(day *microsoft.InventoryItem) int64 {
	return day.Subscription.Quantity
}

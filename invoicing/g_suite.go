package invoicing

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/gsuite"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Plan = string

const (
	PlanAnnual   Plan = "ANNUAL"
	PlanFlexible Plan = "FLEXIBLE"
)

type Payment = string

const (
	PaymentMonthly Payment = "MONTHLY"
	PaymentYearly  Payment = "YEARLY"
)

const (
	Date        = "2006-01-02"
	prettyDate  = "2006/01/02"
	dayDuration = 24 * time.Hour
)

const (
	priceKey20210116 = "20210116"
	priceKeyPrev     = "prev"
	priceKeyCurr     = "curr"
)

// Google workspace enterprise plus (G Suite Ent.) new pricing start date
var newPricesStartDate20210116 = time.Date(2021, 1, 16, 0, 0, 0, 0, time.UTC)

// Google workspace enterprise plus (G Suite Ent.) old pricing (2021-01-16)
var (
	GSuiteEnterpriseYearlyUSD  = 300.0
	GSuiteEnterpriseYearlyEUR  = 276.0
	GSuiteEnterpriseYearlyGBP  = 240.0
	GSuiteEnterpriseYearlyAUD  = 408.0
	GSuiteEnterpriseMonthlyUSD = GSuiteEnterpriseYearlyUSD / 12
	GSuiteEnterpriseMonthlyEUR = GSuiteEnterpriseYearlyEUR / 12
	GSuiteEnterpriseMonthlyGBP = GSuiteEnterpriseYearlyGBP / 12
	GSuiteEnterpriseMonthlyAUD = GSuiteEnterpriseYearlyAUD / 12

	gsuiteEnterpriseYearlyPrices = CatalogItemPrice{
		USD: &GSuiteEnterpriseYearlyUSD,
		EUR: &GSuiteEnterpriseYearlyEUR,
		GBP: &GSuiteEnterpriseYearlyGBP,
		AUD: &GSuiteEnterpriseYearlyAUD,
	}

	gsuiteEnterpriseMonthlyPrices = CatalogItemPrice{
		USD: &GSuiteEnterpriseMonthlyUSD,
		EUR: &GSuiteEnterpriseMonthlyEUR,
		GBP: &GSuiteEnterpriseMonthlyGBP,
		AUD: &GSuiteEnterpriseMonthlyAUD,
	}
)

func pricesCurrencySwitch(currency string, prices CatalogItemPrice) (float64, error) {
	var p *float64

	switch currency {
	case string(fixer.USD):
		p = prices.USD
	case string(fixer.EUR):
		p = prices.EUR
	case string(fixer.GBP):
		p = prices.GBP
	case string(fixer.AUD):
		p = prices.AUD
	case string(fixer.BRL):
		p = prices.BRL
	case string(fixer.NOK):
		p = prices.NOK
	case string(fixer.DKK):
		p = prices.DKK
	default:
		return 0, fmt.Errorf("currency %s is not supported", currency)
	}

	if p == nil {
		return 0, fmt.Errorf("price for currency %s is not defined", currency)
	}

	return *p, nil
}

func getAnnualSubscriptionStartDateOrFallback(invItem *gsuite.InventoryItem, fallback time.Time) time.Time {
	planCheck := func(plan *assets.SubscriptionPlan) *time.Time {
		if plan != nil && plan.IsCommitmentPlan && plan.CommitmentInterval != nil && plan.CommitmentInterval.StartTime != 0 {
			d := common.EpochMillisecondsToTime(plan.CommitmentInterval.StartTime).Truncate(dayDuration)
			return &d
		}

		return nil
	}

	// Get start date from custom asset settings if exists
	if invItem.Settings != nil {
		if d := planCheck(invItem.Settings.Plan); d != nil {
			return *d
		}
	}

	// Get start date from real the subscription plan if exists
	if d := planCheck(invItem.Subscription.Plan); d != nil {
		return *d
	}

	return fallback
}

func getGSuitePrice(invItem *gsuite.InventoryItem, catalogItem *CatalogItem, currency string) (float64, string, error) {
	creationDate := common.EpochMillisecondsToTime(invItem.Subscription.CreationTime).Truncate(dayDuration)

	// TODO (dror): Check if we can remove this old price logic - April 2023: Not yet
	if catalogItem.SkuID == gsuite.GSuiteEnterprise && !creationDate.IsZero() && creationDate.Before(newPricesStartDate20210116) {
		if catalogItem.Payment == PaymentYearly {
			price, err := pricesCurrencySwitch(currency, gsuiteEnterpriseYearlyPrices)
			return price, priceKey20210116, err
		} else if catalogItem.Payment == PaymentMonthly {
			price, err := pricesCurrencySwitch(currency, gsuiteEnterpriseMonthlyPrices)
			return price, priceKey20210116, err
		} else {
			return 0, "", fmt.Errorf("invalid payment mode %s", catalogItem.Payment)
		}
	}

	// Cloud Identity Premium lower price for first year
	if catalogItem.SkuID == gsuite.CloudIdentityPremium && invItem.Date.Before(creationDate.AddDate(1, 0, 0)) {
		price, err := pricesCurrencySwitch(currency, catalogItem.Price)
		if err != nil {
			return 0, "", err
		}

		return price * 0.665, priceKeyCurr, nil
	}

	catalogPrices := catalogItem.Price
	priceKey := priceKeyCurr

	if catalogItem.PrevPrice != nil && catalogItem.PrevPriceEndDate != nil && !catalogItem.PrevPriceEndDate.IsZero() {
		switch catalogItem.Plan {
		case PlanAnnual:
			prevDateValue := getAnnualSubscriptionStartDateOrFallback(invItem, creationDate)

			if !prevDateValue.IsZero() && prevDateValue.Before(*catalogItem.PrevPriceEndDate) {
				catalogPrices = *catalogItem.PrevPrice
				priceKey = priceKeyPrev
			}

		case PlanFlexible:
			if invItem.Date.Before(*catalogItem.PrevPriceEndDate) {
				catalogPrices = *catalogItem.PrevPrice
				priceKey = priceKeyPrev
			}
		}
	}

	price, err := pricesCurrencySwitch(currency, catalogPrices)
	if err != nil {
		return 0, "", err
	}

	return price, priceKey, nil
}

func (s *InvoicingService) customerGSuiteHandler(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	l := s.Logger(ctx)
	fs := s.Firestore(ctx)

	res := &domain.ProductInvoiceRows{
		Type:  common.Assets.GSuite,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	discountsCache := make(map[string][2]float64)
	assetSettingsCache := make(map[string]*common.AssetSettings)
	annuals := make(map[string]*gsuite.InventoryItem)
	ranks := make(map[string]int)
	final := task.TimeIndex == -2
	dailyRate := 1 / float64(task.InvoiceMonth.Day())
	items := make(map[string][]*gsuite.InventoryItem)

	invoiceAdjustments, err := getCustomerInvoiceAdjustments(ctx, customerRef, common.Assets.GSuite, task.InvoiceMonth)
	if err != nil {
		l.Errorf("[g-suite] failed to get invoice adjustments for %s with error: %s", customerRef.ID, err)
		res.Error = err
		respChan <- res

		return
	}

	parentCollection := fmt.Sprintf("%s-%s", gsuite.Inventory, common.Assets.GSuite)
	docID := fmt.Sprintf("%s-%s", task.InvoiceMonth.Format("2006-01"), common.Assets.GSuite)

	docs, err := fs.Collection(gsuite.Inventory).Doc(docID).Collection(parentCollection).
		Where("customer", "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	// Annual credits calculcation for subscriptions that ended before their planned end date.
	for _, docSnap := range docs {
		var item gsuite.InventoryItem
		if err := docSnap.DataTo(&item); err != nil {
			res.Error = err
			respChan <- res

			return
		}

		if item.Subscription.Plan.PlanName == "FREE" || item.Subscription.Plan.PlanName == "TRIAL" {
			continue
		}

		if gsuite.SubscriptionSuspended(item.Subscription.Status) {
			continue
		}

		if item.Subscription.TrialSettings != nil && item.Subscription.TrialSettings.IsInTrial {
			continue
		}

		assetID := fmt.Sprintf("%s-%s", common.Assets.GSuite, item.Subscription.ID)
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

		_, payment, startDate, endDate, _, err := ParseGSuiteInventoryItem(&item)
		if err != nil {
			res.Error = err
			respChan <- res

			return
		}

		if v, prs := annuals[item.Subscription.ID]; prs {
			vPlan, vPayment, vStartDate, vEndDate, vCurrency, err := ParseGSuiteInventoryItem(v)
			if err != nil {
				res.Error = err
				respChan <- res

				return
			}

			if payment == PaymentYearly && startDate.Equal(vStartDate) && endDate.Equal(vEndDate) {
				annuals[item.Subscription.ID] = &item
			} else {
				// if YEARLY payment ended more than a few days before then the actual end date then we should to credit
				if vEndDate.Sub(v.Date) > time.Hour*24*7 {
					var catalogItem *CatalogItem

					for _, c := range gsuiteCatalog {
						if c.SkuID == v.Subscription.SkuID && c.Plan == vPlan && c.Payment == vPayment {
							catalogItem = c
							break
						}
					}

					quantity := getGSuiteQuantity(v)
					discounts, _ := utils.GetDiscount(ctx, v.Contract, discountsCache)
					rank := getCustomerRank(v.CustomerID, v.Subscription.ID, v.Date.Format(Date), ranks)
					remainingDays := vEndDate.Sub(v.Date).Hours() / 24
					totalDays := vEndDate.Sub(vStartDate).Hours() / 24
					proration := remainingDays / totalDays

					ppu, _, err := getGSuitePrice(v, catalogItem, vCurrency)
					if err != nil {
						res.Error = err
						respChan <- res

						return
					}

					ppu *= proration

					if discounts[1] != 0 {
						yearDays := vStartDate.AddDate(1, 0, 0).Sub(vStartDate).Hours() / 24
						priceMultiplier := utils.ToProportion(discounts[1]) * (totalDays / yearDays)
						ppu *= priceMultiplier
					}

					total := float64(quantity) * ppu * utils.ToProportion(discounts[0])
					row := &domain.InvoiceRow{
						Description: catalogItem.SkuName,
						Details:     fmt.Sprintf("Annual credit from %s to %s for domain %s", v.Date.Format(prettyDate), vEndDate.Format(prettyDate), v.CustomerDomain),
						Quantity:    -quantity,
						PPU:         ppu,
						Discount:    discounts[0],
						Currency:    vCurrency,
						Total:       -total,
						SKU:         catalogItem.SkuPriority,
						Rank:        rank,
						Type:        common.Assets.GSuite,
						Final:       false,
						Entity:      as.Entity,
						Bucket:      as.Bucket,
					}
					res.Rows = append(res.Rows, row)
				}

				delete(annuals, item.Subscription.ID)
			}
		} else {
			if payment == PaymentYearly && (item.Date.Equal(startDate) || item.Date.After(startDate)) {
				annuals[item.Subscription.ID] = &item
			}
		}
	}

	yesterday := task.Now.Truncate(dayDuration).Add(-dayDuration)
	for _, v := range annuals {
		if (task.TimeIndex == -1 && v.Date.Before(yesterday)) || (task.TimeIndex == -2 && v.Date.Before(task.InvoiceMonth)) {
			vPlan, vPayment, vStartDate, vEndDate, vCurrency, err := ParseGSuiteInventoryItem(v)
			if err != nil {
				res.Error = err
				respChan <- res

				return
			}

			if vEndDate.Sub(v.Date) > time.Hour*24*7 {
				assetID := fmt.Sprintf("g-suite-%s", v.Subscription.ID)
				as := assetSettingsCache[assetID]

				var catalogItem *CatalogItem

				for _, c := range gsuiteCatalog {
					if c.SkuID == v.Subscription.SkuID && c.Plan == vPlan && c.Payment == vPayment {
						catalogItem = c
						break
					}
				}

				quantity := getGSuiteQuantity(v)
				discounts, _ := utils.GetDiscount(ctx, v.Contract, discountsCache)
				rank := getCustomerRank(v.CustomerID, v.Subscription.ID, v.Date.Format(Date), ranks)
				remainingDays := vEndDate.Sub(v.Date).Hours() / 24
				totalDays := vEndDate.Sub(vStartDate).Hours() / 24
				proration := remainingDays / float64(totalDays)

				ppu, _, err := getGSuitePrice(v, catalogItem, vCurrency)
				if err != nil {
					res.Error = err
					respChan <- res

					return
				}

				ppu *= proration

				if discounts[1] != 0 {
					yearDays := vStartDate.AddDate(1, 0, 0).Sub(vStartDate).Hours() / 24
					priceMultiplier := utils.ToProportion(discounts[1]) * (totalDays / yearDays)
					ppu *= priceMultiplier
				}

				total := float64(quantity) * ppu * utils.ToProportion(discounts[0])
				row := &domain.InvoiceRow{
					Description: catalogItem.SkuName,
					Details:     fmt.Sprintf("Annual credit from %s to %s for domain %s", v.Date.Format(prettyDate), vEndDate.Format(prettyDate), v.CustomerDomain),
					Quantity:    -quantity,
					PPU:         ppu,
					Discount:    discounts[0],
					Currency:    vCurrency,
					Total:       -total,
					SKU:         catalogItem.SkuPriority,
					Rank:        rank,
					Type:        common.Assets.GSuite,
					Final:       false,
					Entity:      as.Entity,
					Bucket:      as.Bucket,
				}
				res.Rows = append(res.Rows, row)
			}
		}
	}

	for assetID, days := range items {
		var (
			prevQuantity   int64
			prevContractID string
			prevPriceKey   string
			prevFlexPeriod time.Time
		)

		rowData := make(map[string]*domain.InvoiceRow)
		as := assetSettingsCache[assetID]

		for i, day := range days {
			if day.Subscription == nil || day.Subscription.Plan == nil {
				res.Error = fmt.Errorf("invalid subscription %s", assetID)
				respChan <- res

				return
			}

			if !gsuite.SubscriptionActive(day.Subscription.Status) {
				l.Warningf("[g-suite] subscription %s status was '%s' on %s", day.Subscription.ID, day.Subscription.Status, day.Date.Format(Date))
			}

			// Do not charge for days where subscription had a FREE/TRIAL plan names
			if day.Subscription.Plan.PlanName == "FREE" || day.Subscription.Plan.PlanName == "TRIAL" {
				continue
			}

			// Do not charge for days where subscription was suspended
			if gsuite.SubscriptionSuspended(day.Subscription.Status) {
				continue
			}

			// Do not charge for days where subscription was in trial
			if day.Subscription.TrialSettings != nil && day.Subscription.TrialSettings.IsInTrial {
				continue
			}

			if day.Subscription.BillingMethod == "OFFLINE" {
				// Skip OFFLINE Cloud Identity Free and G Suite Lite subscriptions
				if day.Subscription.SkuID == gsuite.CloudIdentityFree || day.Subscription.SkuID == gsuite.GSuiteLite {
					continue
				}

				// Skip OFFLINE annual subscription with no commitment interval on subscription or settings
				if day.Subscription.Plan.PlanName == "ANNUAL" &&
					day.Subscription.Plan.CommitmentInterval == nil &&
					day.Settings == nil {
					continue
				}
			}

			plan, payment, startDate, endDate, currency, err := ParseGSuiteInventoryItem(day)
			if err != nil {
				res.Error = err
				respChan <- res

				return
			}

			var quantity = getGSuiteQuantity(day)
			if quantity <= 0 {
				continue
			}

			var catalogItem *CatalogItem

			for _, c := range gsuiteCatalog {
				if c.SkuID == day.Subscription.SkuID && c.Plan == plan && c.Payment == payment {
					catalogItem = c
					break
				}
			}

			if catalogItem == nil {
				res.Error = fmt.Errorf("(%s) could not find %s %s %s in catalog", day.Subscription.ID, day.Subscription.SkuID, plan, payment)
				respChan <- res

				return
			}

			var discounts, contractID = utils.GetDiscount(ctx, day.Contract, discountsCache)

			if payment == PaymentMonthly {
				// Get the daily price
				ppu, priceKey, err := getGSuitePrice(day, catalogItem, currency)
				if err != nil {
					res.Error = err
					respChan <- res

					return
				}

				ppu *= dailyRate

				if i == 0 || prevFlexPeriod.IsZero() || quantity != prevQuantity || contractID != prevContractID || priceKey != prevPriceKey {
					prevQuantity = quantity
					prevContractID = contractID
					prevFlexPeriod = day.Date
					prevPriceKey = priceKey
				}

				// If there's a special promo for this contract
				if plan == "ANNUAL" {
					if discounts[1] != 0 {
						ppu *= utils.ToProportion(discounts[1])
					}

					// Validate end date is not in the past
					remainingDays := endDate.Sub(day.Date).Hours() / 24
					if remainingDays < 0 {
						res.Error = fmt.Errorf("subscription was past its end date on %s for %s (%s)", day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
						respChan <- res

						return
					}
				}

				k := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%s", catalogItem.SkuPriority, day.CustomerID, day.Subscription.ID, prevFlexPeriod.Format(Date), currency, priceKey, contractID)
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
						Type:        common.Assets.GSuite,
						Final:       final,
						Entity:      as.Entity,
						Bucket:      as.Bucket,
					}
					rowData[k] = newRow
				}
			}

			if payment == PaymentYearly {
				var (
					adjustment bool
					ppu        float64
					increase   int64
					priceKey   string
				)

				if startDate.Equal(day.Date) {
					// Amount of seats
					increase = quantity

					// Get the yearly price
					ppu, priceKey, err = getGSuitePrice(day, catalogItem, currency)
					if err != nil {
						res.Error = err
						respChan <- res

						return
					}

					// If subscription is not a 1 year plan
					if endDate.After(startDate.AddDate(1, 0, 0)) {
						totalDays := endDate.Sub(startDate).Hours() / 24
						if totalDays < 0 {
							res.Error = fmt.Errorf("subscription has invalid start/end dates for %s (%s)", day.Subscription.ID, day.CustomerDomain)
							respChan <- res

							return
						}

						yearDays := startDate.AddDate(1, 0, 0).Sub(startDate).Hours() / 24
						ppu *= (totalDays / yearDays)
					}

					// If there's a special promo for this contract
					if discounts[1] != 0 {
						ppu *= utils.ToProportion(discounts[1])
					}
				} else {
					if i == 0 {
						prevDay, err := getGSuitePreviousDay(ctx, l, fs, day)
						if err != nil {
							res.Error = err
							respChan <- res

							return
						}

						if prevDay != nil {
							prevQuantity = getGSuiteQuantity(prevDay)
						}
					} else {
						prevQuantity = getGSuiteQuantity(days[i-1])
					}

					if startDate.After(day.Date) {
						var creationDate = common.EpochMillisecondsToTime(day.Subscription.CreationTime).UTC().Truncate(dayDuration)
						if creationDate.After(day.Date) {
							res.Error = fmt.Errorf("start date %v is after date %v for %s (%s)", startDate.Format(Date), day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
							respChan <- res

							return
						}

						l.Warningf("[g-suite] start date %v is after date %v for %s (%s) - SKIPPING", startDate.Format(Date), day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)

						continue
					}

					if prevQuantity == 0 {
						// If there is no previous day for a commitment; it's possible it's a new flexible subscription that has annual override
						// It should be treated as an increase of subscription, otherwise this might be an error...
						if day.Subscription.Plan.IsCommitmentPlan && day.Settings == nil {
							res.Error = fmt.Errorf("no seats count for previous day on %v for %s (%s)", day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
							respChan <- res

							return
						}
					}

					increase = quantity - prevQuantity
					prevQuantity = quantity

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

					if increase == 0 {
						// No increase in seats, do nothing.
						continue
					}

					if increase < 0 {
						res.Error = fmt.Errorf("invalid seats count on %v for %s (%s)", day.Date.Format(Date), day.Subscription.ID, day.CustomerDomain)
						respChan <- res

						return
					}

					// Prorated pricing
					proration := remainingDays / totalDays

					// Get the yearly price
					ppu, priceKey, err = getGSuitePrice(day, catalogItem, currency)
					if err != nil {
						res.Error = err
						respChan <- res

						return
					}

					ppu *= proration

					// If subscription is not a 1 year plan
					if endDate.After(startDate.AddDate(1, 0, 0)) {
						yearDays := startDate.AddDate(1, 0, 0).Sub(startDate).Hours() / 24
						ppu *= (totalDays / yearDays)
					}

					// If there's a special promo for this contract
					if discounts[1] != 0 {
						ppu *= utils.ToProportion(discounts[1])
					}
				}

				k := fmt.Sprintf("%s#%s#%s#%s#%s#%s#%s#%v", catalogItem.SkuPriority, day.CustomerID, day.Subscription.ID, day.Date.Format(Date), currency, priceKey, contractID, adjustment)

				var details string

				var deferredRevenuePeriod *domain.DeferredRevenuePeriod

				if adjustment {
					details = fmt.Sprintf("Annual adjustment on %s for domain %s", day.Date.Format(prettyDate), day.CustomerDomain)
					deferredRevenuePeriod = &domain.DeferredRevenuePeriod{
						StartDate: day.Date,
						EndDate:   endDate,
					}
				} else {
					details = fmt.Sprintf("Annual commitment from %s to %s for domain %s", startDate.Format(prettyDate), endDate.Format(prettyDate), day.CustomerDomain)
					deferredRevenuePeriod = &domain.DeferredRevenuePeriod{
						StartDate: startDate,
						EndDate:   endDate,
					}
				}

				rank := getCustomerRank(day.CustomerID, day.Subscription.ID, day.Date.Format(Date), ranks)
				total := float64(increase) * ppu * utils.ToProportion(discounts[0])
				newRow := &domain.InvoiceRow{
					Description:           catalogItem.SkuName,
					Details:               details,
					Tags:                  as.Tags,
					Quantity:              increase,
					PPU:                   ppu,
					Discount:              discounts[0],
					Currency:              currency,
					Total:                 total,
					SKU:                   catalogItem.SkuPriority,
					Rank:                  rank,
					Type:                  common.Assets.GSuite,
					Final:                 final,
					DeferredRevenuePeriod: deferredRevenuePeriod,
					Entity:                as.Entity,
					Bucket:                as.Bucket,
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
			l.Errorf("[g-suite] invalid entity for invoice adjustment %s", invoiceAdjustment.Snapshot.Ref.Path)
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
			SKU:         GSuiteSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.GSuite,
			Final:       true,
			Entity:      invoiceAdjustment.Entity,
			Bucket:      entity.Invoicing.Default,
		})
	}

	respChan <- res
}

func getGSuitePreviousDay(ctx context.Context, l logger.ILogger, fs *firestore.Client, day *gsuite.InventoryItem) (*gsuite.InventoryItem, error) {
	prevDate := day.Date.Add(-dayDuration)
	fullDateString := prevDate.Format("2006-01-02")
	monthDateString := prevDate.Format("2006-01")

	parentDocID := fmt.Sprintf("%s-%s", monthDateString, common.Assets.GSuite)
	parentCollection := fmt.Sprintf("%s-%s", gsuite.Inventory, common.Assets.GSuite)
	docID := fmt.Sprintf("%s-%s-%s", fullDateString, common.Assets.GSuite, day.Subscription.ID)

	docRef := fs.Collection(gsuite.Inventory).Doc(parentDocID).Collection(parentCollection).Doc(docID)

	docSnap, err := docRef.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		l.Errorf("[g-suite] failed to get previous day for %s with error: %s", docID, err)
		return nil, err
	}

	if docSnap != nil && docSnap.Exists() {
		var prevDay gsuite.InventoryItem
		if err := docSnap.DataTo(&prevDay); err != nil {
			l.Errorf("[g-suite] failed to populate previous day for %s with error: %s", docID, err)
			return nil, err
		}

		return &prevDay, nil
	}

	return nil, nil
}

func getGSuiteQuantity(day *gsuite.InventoryItem) int64 {
	var quantity int64

	if day == nil || day.Subscription == nil || day.Subscription.Seats == nil {
		return quantity
	}

	if day.Subscription.Plan.IsCommitmentPlan {
		quantity = day.Subscription.Seats.NumberOfSeats

		// Handle Reseller API List subs bug during March 2022, that caused amount of licenses to be
		// lower than amount of users with licenses for some subscriptions.
		if day.Subscription.Seats.LicensedNumberOfSeats > day.Subscription.Seats.NumberOfSeats {
			quantity = day.Subscription.Seats.LicensedNumberOfSeats
		}
	} else {
		// Non commitment plan should be charged according to the licenses the are currently used,
		// and not by the maximum number of licenses.
		quantity = day.Subscription.Seats.LicensedNumberOfSeats

		// When Flex plan is overridden as a commitment plan, the quantity should be the maximum
		// number of licenses.
		if day.Settings != nil && day.Settings.Plan != nil && day.Settings.Plan.IsCommitmentPlan {
			quantity = day.Subscription.Seats.MaximumNumberOfSeats
		}
	}

	if quantity >= 25000 {
		quantity = day.Subscription.Seats.LicensedNumberOfSeats
	}

	return quantity
}

func ParseGSuiteInventoryItem(day *gsuite.InventoryItem) (string, string, time.Time, time.Time, string, error) {
	var (
		plan      string
		payment   string
		startDate time.Time
		endDate   time.Time
		currency  = "USD"
	)

	if day.Subscription.Plan.IsCommitmentPlan {
		plan = "ANNUAL"
		payment = PaymentYearly

		if day.Subscription.Plan.CommitmentInterval != nil {
			startDate = common.EpochMillisecondsToTime(day.Subscription.Plan.CommitmentInterval.StartTime).UTC().Truncate(dayDuration)
			endDate = common.EpochMillisecondsToTime(day.Subscription.Plan.CommitmentInterval.EndTime).UTC().Truncate(dayDuration)
		}
	} else {
		plan = "FLEXIBLE"
		payment = PaymentMonthly
	}

	if day.Settings != nil {
		if day.Settings.Plan != nil {
			plan = day.Settings.Plan.PlanName

			if day.Settings.Plan.IsCommitmentPlan {
				payment = PaymentYearly
				startDate = common.EpochMillisecondsToTime(day.Settings.Plan.CommitmentInterval.StartTime).UTC().Truncate(dayDuration)
				endDate = common.EpochMillisecondsToTime(day.Settings.Plan.CommitmentInterval.EndTime).UTC().Truncate(dayDuration)
			} else {
				payment = PaymentMonthly
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

func getCustomerRank(customerID, subscriptionID, id string, ranks map[string]int) int {
	k := fmt.Sprintf("%s-%s-%s", customerID, subscriptionID, id)

	v, prs := ranks[k]
	if prs {
		return v
	}

	ranks[k] = len(ranks) + 1

	return ranks[k]
}

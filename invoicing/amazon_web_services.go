package invoicing

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
)

type flexsaveAwsAuxilaryCosts struct {
	amount float64
	entity *firestore.DocumentRef
	bucket *firestore.DocumentRef
}

func (s *CustomerAssetInvoiceWorker) GetAWSInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	logger := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	res := &domain.ProductInvoiceRows{
		Type:  common.Assets.AmazonWebServices,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	final := task.TimeIndex == -2 && task.Now.Day() >= 3
	credits := make(map[string]map[string]map[string]float64)

	monthlyBillingData, err := s.customerMonthlyBillingAmazonWebServices(ctx, customerRef, task.InvoiceMonth)
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	invoiceAdjustments, err := getCustomerInvoiceAdjustments(ctx, customerRef, common.Assets.AmazonWebServices, task.InvoiceMonth)
	if err != nil {
		res.Error = err
		respChan <- res

		return
	}

	assetTypeLength := len(common.Assets.AmazonWebServices)
	flexsaveComputeSavings := make(map[string]*flexsaveAwsAuxilaryCosts)
	flexsaveSagemakerSavings := make(map[string]*flexsaveAwsAuxilaryCosts)
	flexsaveRDSSavings := make(map[string]*flexsaveAwsAuxilaryCosts)
	flexsaveManagementCosts := make(map[string]*flexsaveAwsAuxilaryCosts)
	flexsaveRDSCharges := make(map[string]*flexsaveAwsAuxilaryCosts)
	flexsaveCredits := make(map[string]*flexsaveAwsAuxilaryCosts)
	flexsaveAdjustments := make(map[string]*flexsaveAwsAuxilaryCosts)

	for docID, data := range monthlyBillingData {
		accountID := docID[assetTypeLength+1:]

		assetSettings, err := getAmazonWebServicesAssetSettings(ctx, fs, docID)
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
			logger.Info("skipping billing profile 999999")
			respChan <- res

			return
		}

		// Handle the Flexsave savings and management cost line items
		if data.Flexsave != nil {
			// Adjust the account spend with flexsave savings
			// we add the saving to the account total and instead add a separate line with the
			// total Flexsave savings
			data.Spend -= data.Flexsave.FlexsaveComputeNegations
			data.Spend -= data.Flexsave.FlexsaveSagemakerNegations
			data.Spend -= data.Flexsave.FlexsaveRDSNegations
			data.Spend -= data.Flexsave.FlexsaveAdjustments

			auxCostID := entityRef.ID
			if assetSettings.Bucket != nil {
				auxCostID = entityRef.ID + "_" + assetSettings.Bucket.ID
			}

			if _, ok := flexsaveComputeSavings[auxCostID]; !ok {
				flexsaveComputeSavings[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			if _, ok := flexsaveSagemakerSavings[auxCostID]; !ok {
				flexsaveSagemakerSavings[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			if _, ok := flexsaveRDSSavings[auxCostID]; !ok {
				flexsaveRDSSavings[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			if _, ok := flexsaveRDSCharges[auxCostID]; !ok {
				flexsaveRDSCharges[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			if _, ok := flexsaveManagementCosts[auxCostID]; !ok {
				flexsaveManagementCosts[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			if _, ok := flexsaveCredits[auxCostID]; !ok {
				flexsaveCredits[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			if _, ok := flexsaveAdjustments[auxCostID]; !ok {
				flexsaveAdjustments[auxCostID] = &flexsaveAwsAuxilaryCosts{
					entity: entityRef,
					bucket: assetSettings.Bucket,
				}
			}

			flexsaveComputeSavings[auxCostID].amount += data.Flexsave.FlexsaveComputeNegations
			flexsaveSagemakerSavings[auxCostID].amount += data.Flexsave.FlexsaveSagemakerNegations
			flexsaveRDSSavings[auxCostID].amount += data.Flexsave.FlexsaveRDSNegations
			flexsaveRDSCharges[auxCostID].amount += data.Flexsave.FlexsaveRDSCharges
			flexsaveManagementCosts[auxCostID].amount += data.Flexsave.ManagementCosts
			flexsaveCredits[auxCostID].amount += data.Flexsave.FlexsaveSpCredits
			flexsaveAdjustments[auxCostID].amount += data.Flexsave.FlexsaveAdjustments
		}

		marketplaceInvMode := marketplaceInvoicingMode(data.MarketplaceConstituents, entity.Invoicing)

		if data.Spend != 0 {
			if marketplaceInvMode == "none" {
				qty, value := utils.GetQuantityAndValue(1, data.Spend)

				res.Rows = append(res.Rows, &domain.InvoiceRow{
					Description:             "Amazon Web Services",
					Details:                 fmt.Sprintf("Account #%s", accountID),
					Tags:                    assetSettings.Tags,
					Quantity:                qty,
					PPU:                     value,
					Currency:                "USD",
					Total:                   float64(qty) * value,
					SKU:                     AmazonWebServicesSKU,
					Rank:                    1,
					Type:                    common.Assets.AmazonWebServices,
					Final:                   final && data.Verified,
					Entity:                  entityRef,
					Bucket:                  assetSettings.Bucket,
					CustBillingTblSessionID: data.CustBillingTblSessionID,
				})
			} else {
				for marketplaceSDKey, marketplaceValue := range data.MarketplaceConstituents {
					adjustedSpend := marketplaceValue.Spend
					if marketplaceSDKey == "marketplace_none" { // marketplace_none doesn't have this adjustment
						adjustedSpend = marketplaceValue.Spend - (data.Flexsave.FlexsaveComputeNegations +
							data.Flexsave.FlexsaveRDSNegations + data.Flexsave.FlexsaveSagemakerNegations + data.Flexsave.FlexsaveAdjustments)
					}

					qty, value := utils.GetQuantityAndValue(1, adjustedSpend)
					sd := data.MarketplaceConstituentsRef[marketplaceSDKey]

					if sd == "" {
						sd = marketplaceSDKey
					}

					currentRow := domain.InvoiceRow{
						Description:             "Amazon Web Services",
						Details:                 fmt.Sprintf("Account #%s : %s", accountID, sd),
						Tags:                    assetSettings.Tags,
						Quantity:                qty,
						PPU:                     value,
						Currency:                "USD",
						Total:                   float64(qty) * value,
						SKU:                     AmazonWebServicesSKU,
						Rank:                    1,
						Type:                    common.Assets.AmazonWebServices,
						Final:                   final && data.Verified,
						Entity:                  entityRef,
						Bucket:                  assetSettings.Bucket,
						CustBillingTblSessionID: data.CustBillingTblSessionID,
					}

					if marketplaceSDKey != "marketplace_none" {
						// row.category can Never be 'marketplace_none'
						// only valid values for row.category are - "" OR "marketplace_aggregate" OR "marketplace_<<someNumericHash>>"
						if marketplaceInvMode == "marketplace_aggregate" {
							currentRow.Category = "marketplace_aggregate"
						}

						if marketplaceInvMode == "marketplace_individual" {
							currentRow.Category = marketplaceSDKey
						}
					}

					res.Rows = append(res.Rows, &currentRow)
				}
			}
		}

		if marketplaceInvMode == "none" {
			// Sum used credits from different accounts
			for creditID, value := range data.Credits {
				if _, prs := credits[entityRef.ID]; !prs {
					credits[entityRef.ID] = make(map[string]map[string]float64)
				}

				if _, prs := credits[entityRef.ID][creditID]; !prs {
					credits[entityRef.ID][creditID] = make(map[string]float64)
				}

				if assetSettings.Bucket != nil {
					credits[entityRef.ID][creditID][assetSettings.Bucket.ID] += value
				} else {
					credits[entityRef.ID][creditID][""] += value
				}
			}
		} else {
			for marketplaceSDKey, marketplaceValue := range data.MarketplaceConstituents {
				invoiceBatchKey := ""
				if assetSettings.Bucket != nil {
					invoiceBatchKey = assetSettings.Bucket.ID
				}

				if marketplaceSDKey != "marketplace_none" {
					// row.category can Never be 'marketplace_none'
					// only valid values for invoiceBatchKey are - "" OR "marketplace_aggregate" OR "marketplace_<<someNumericHash>>"

					if marketplaceInvMode == "marketplace_aggregate" {
						invoiceBatchKey = "marketplace_aggregate"
					}

					if marketplaceInvMode == "marketplace_individual" {
						invoiceBatchKey = marketplaceSDKey
					}

					if assetSettings.Bucket != nil {
						invoiceBatchKey = invoiceBatchKey + "__" + assetSettings.Bucket.ID
					}
				}

				// Sum used credits from different accounts
				for creditID, value := range marketplaceValue.Credits {
					if _, prs := credits[entityRef.ID]; !prs {
						credits[entityRef.ID] = make(map[string]map[string]float64)
					}

					if _, prs := credits[entityRef.ID][creditID]; !prs {
						credits[entityRef.ID][creditID] = make(map[string]float64)
					}

					credits[entityRef.ID][creditID][invoiceBatchKey] += value
				}
			}
		}
	}

	for entityID, entityCredits := range credits {
		entityRef := fs.Collection("entities").Doc(entityID)

		for creditID, creditValues := range entityCredits {
			docSnap, err := fs.Collection("customers").Doc(task.CustomerID).Collection("customerCredits").Doc(creditID).Get(ctx)
			if err != nil {
				res.Error = err
				respChan <- res

				return
			}

			name, err := docSnap.DataAt("name")
			if err != nil {
				res.Error = err
				respChan <- res

				return
			}

			for bucketID, value := range creditValues {
				var bucketRef *firestore.DocumentRef

				category := ""

				if bucketID != "" && !strings.HasPrefix(bucketID, "marketplace") {
					bucketRef = entityRef.Collection("buckets").Doc(bucketID)
				} else if strings.HasPrefix(bucketID, "marketplace") && strings.Contains(bucketID, "__") {
					b := strings.Split(bucketID, "__")
					category = b[0]
					bucketRef = entityRef.Collection("buckets").Doc(b[1])
				}

				res.Rows = append(res.Rows, &domain.InvoiceRow{
					Description: "Amazon Web Services Credit",
					Details:     name.(string),
					Quantity:    -1,
					PPU:         value,
					Currency:    "USD",
					Total:       -value,
					SKU:         AmazonWebServicesSKU,
					Rank:        CreditRank,
					Type:        common.Assets.AmazonWebServices,
					Final:       final,
					Entity:      entityRef,
					Bucket:      bucketRef,
					Category:    category, // category has higher priority over bucket configs
				})
			}
		}
	}

	for _, invoiceAdjustment := range invoiceAdjustments {
		if _, prs := entities[invoiceAdjustment.Entity.ID]; !prs {
			err := fmt.Errorf("invalid entity for invoiceAdjustment %s", invoiceAdjustment.Snapshot.Ref.Path)
			res.Error = err
			respChan <- res

			return
		}

		var (
			runningTotal float64
			hasCredits   bool
		)

		if common.ComparableFloat(invoiceAdjustment.Amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, invoiceAdjustment.Amount)

		var invoiceAdjustmentAmount float64 = invoiceAdjustment.Amount

		for _, row := range res.Rows {
			runningTotal += row.Total

			if row.SKU == AmazonWebServicesSKU && row.Description == "Amazon Web Services Credit" {
				hasCredits = true
			}
		}

		if hasCredits && invoiceAdjustment.Description == "Flexsave Savings" {
			// If the spend is fully covered by credits then we don't add the invoice adjustment
			if common.ComparableFloat(runningTotal).IsZero() {
				logger.Info("Skipping Flexsave Savings adjustment due to credits.")
				continue
			}
			// If spend after applying credits is nonzero, adjust invoiceAdjustment up to the remaining spend
			if runningTotal > 0 {
				invoiceAdjustmentAmount = math.Max(-1*runningTotal, invoiceAdjustment.Amount)
				qty, value = utils.GetQuantityAndValue(1, invoiceAdjustmentAmount)
			}
		}

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: invoiceAdjustment.Description,
			Details:     invoiceAdjustment.Details,
			Quantity:    qty,
			PPU:         value,
			Currency:    invoiceAdjustment.Currency,
			Total:       invoiceAdjustmentAmount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       true,
			Entity:      invoiceAdjustment.Entity,
			Bucket:      nil,
		})
	}

	for _, flexsaveComputeSavings := range flexsaveComputeSavings {
		if common.ComparableFloat(flexsaveComputeSavings.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, flexsaveComputeSavings.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     "DoiT Flexsave Compute Savings",
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       flexsaveComputeSavings.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      flexsaveComputeSavings.entity,
			Bucket:      flexsaveComputeSavings.bucket,
		})
	}

	for _, flexsaveSagemakerSavings := range flexsaveSagemakerSavings {
		if common.ComparableFloat(flexsaveSagemakerSavings.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, flexsaveSagemakerSavings.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     "DoiT Flexsave SageMaker Savings",
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       flexsaveSagemakerSavings.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      flexsaveSagemakerSavings.entity,
			Bucket:      flexsaveSagemakerSavings.bucket,
		})
	}

	for _, flexsaveRDSSavings := range flexsaveRDSSavings {
		if common.ComparableFloat(flexsaveRDSSavings.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, flexsaveRDSSavings.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     "DoiT Flexsave RDS Savings",
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       flexsaveRDSSavings.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      flexsaveRDSSavings.entity,
			Bucket:      flexsaveRDSSavings.bucket,
		})
	}

	for _, managementCost := range flexsaveManagementCosts {
		if common.ComparableFloat(managementCost.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, managementCost.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     flexsaveManagementCost,
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       managementCost.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      managementCost.entity,
			Bucket:      managementCost.bucket,
		})
	}

	for _, rdsCharges := range flexsaveRDSCharges {
		if common.ComparableFloat(rdsCharges.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, rdsCharges.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     flexsaveRDSCharge,
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       rdsCharges.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      rdsCharges.entity,
			Bucket:      rdsCharges.bucket,
		})
	}

	for _, flexsaveCredit := range flexsaveCredits {
		if common.ComparableFloat(flexsaveCredit.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, flexsaveCredit.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     "AWS Credits for eligible Flexsave charges",
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       flexsaveCredit.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      flexsaveCredit.entity,
			Bucket:      flexsaveCredit.bucket,
		})
	}

	for _, flexsaveAdjustment := range flexsaveAdjustments {
		if common.ComparableFloat(flexsaveAdjustment.amount).IsZero() {
			continue
		}

		qty, value := utils.GetQuantityAndValue(1, flexsaveAdjustment.amount)

		res.Rows = append(res.Rows, &domain.InvoiceRow{
			Description: "Flexsave",
			Details:     "Flexsave Adjustments for Credits",
			Quantity:    qty,
			PPU:         value,
			Currency:    "USD",
			Total:       flexsaveAdjustment.amount,
			SKU:         AmazonWebServicesSKU,
			Rank:        InvoiceAdjustmentRank,
			Type:        common.Assets.AmazonWebServices,
			Final:       final,
			Entity:      flexsaveAdjustment.entity,
			Bucket:      flexsaveAdjustment.bucket,
		})
	}

	respChan <- res
}

// marketplaceInvoicingMode returns oneOf "none", "marketplace_individual", "marketplace_aggregate"
func marketplaceInvoicingMode(marketplaceConstituents map[string]pkg.MarketplaceConstituent, entityInvoicing common.Invoicing) interface{} {
	if len(marketplaceConstituents) == 0 {
		return "none"
	}

	separateInvoice := false
	invoicePerService := false

	if len(entityInvoicing.Marketplace) > 0 {
		separateInvoice, _ = entityInvoicing.Marketplace["separateInvoice"].(bool)
		invoicePerService, _ = entityInvoicing.Marketplace["invoicePerService"].(bool)

		if separateInvoice && invoicePerService {
			return "marketplace_individual"
		} else if separateInvoice {
			return "marketplace_aggregate"
		}
	}

	return "none"
}

func getAmazonWebServicesAssetSettings(ctx context.Context, fs *firestore.Client, docID string) (*common.AssetSettings, error) {
	doc, err := fs.Collection("assetSettings").Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var assetSettings common.AssetSettings
	if err := doc.DataTo(&assetSettings); err != nil {
		return nil, err
	}

	return &assetSettings, nil
}

func (s *CustomerAssetInvoiceWorker) customerMonthlyBillingAmazonWebServices(ctx context.Context, customerRef *firestore.DocumentRef, invoiceMonth time.Time) (map[string]*pkg.MonthlyBillingAmazonWebServices, error) {
	accountConfig, err := s.customersDAL.GetCustomerAWSAccountConfiguration(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	return s.monthlyBillingDataDAL.GetCustomerAWSAssetIDtoMonthlyBillingData(ctx, customerRef, invoiceMonth, accountConfig.UseAnalyticsDataForInvoice)
}

func (s *CustomerAssetInvoiceWorker) GetAWSStandaloneInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	s.GetStandaloneInvoiceRows(ctx, task, customerRef, entities, respChan, common.Assets.AmazonWebServicesStandalone)
}

func (s *CustomerAssetInvoiceWorker) GetGCPStandaloneInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, entities map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	s.GetStandaloneInvoiceRows(ctx, task, customerRef, entities, respChan, common.Assets.GoogleCloudStandalone)
}

func (s *CustomerAssetInvoiceWorker) IsUseAnalyticsDataForInvoice(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	accountConfig, err := s.customersDAL.GetCustomerAWSAccountConfiguration(ctx, customerRef)

	if err != nil {
		return false, err
	}

	return accountConfig.UseAnalyticsDataForInvoice, nil
}

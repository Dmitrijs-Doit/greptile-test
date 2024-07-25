package invoicing

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/maps"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	assetpkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/pkg"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/times"
	"github.com/gin-gonic/gin"
)

const (
	SkuComputeSp3yrNoUpfront          = "ComputeSP:3yrNoUpfront"
	SkuComputeSp3yrNoUpfrontFmt       = "ComputeSP_3yrNoUpfront"
	CostTypeCredit                    = "Credit"
	CostTypeFlexsaveNegation          = "FlexsaveNegation"
	CostTypeFlexsaveManagementFee     = "FlexsaveManagementFee"
	CostTypeFlexsaveRDSCharges        = "FlexsaveRDSCharges"
	CostTypeFlexsaveRIFee             = "FlexsaveRIFee"
	CostTypeFlexsaveRefund            = "FlexsaveRefund"
	CostTypeFlexsaveComputeNegation   = "FlexsaveComputeNegation"
	CostTypeFlexsaveAdjustment        = "FlexsaveAdjustment"
	CostTypeFlexsaveSagemakerNegation = "FlexsaveSagemakerNegation"
	CostTypeFlexsaveRDSManagementFee  = "FlexsaveRDSManagementFee"
	ServiceIdAmazonSageMaker          = "AmazonSageMaker"
	ServiceIdAmazonRDS                = "AmazonRDS"
	MarketplaceServiceDescription     = "MarketplaceServiceDescription"
	CustomCostType                    = "CustomCostType"
	DoitDataSource                    = "doit/data_source"
	CustomerBackfill                  = "customer_backfill"
	v2                                = "v2"
	v1                                = "v1"
	delimiter                         = "__"
)

// UpdateAmazonWebServicesInvoicingData processes all customers accountIDs spend for a given month
func (s *AnalyticsAWSInvoicingService) UpdateAmazonWebServicesInvoicingData(ctx context.Context, invoiceMonthInput, version string, validateWithOldLogic, dry bool) error {
	logger := s.loggerProvider(ctx)
	logBatch := time.Now().UTC().Truncate(time.Hour * 6).Format(time.DateTime)

	invoiceMonth, err := s.invoiceMonthParser.GetInvoiceMonth(invoiceMonthInput)
	if err != nil {
		return err
	}

	logger.Infof("fetching customer list for aws-analytics invoicing for invoiceMonth %v", invoiceMonth)

	var customerIDs, auditCustomerIDs, auditZeroCostCustomerIDs, auditStandAloneCostCustomerIDs, unifiedTableCustomerIDs []string

	auditCustomerIDs, auditZeroCostCustomerIDs, auditStandAloneCostCustomerIDs, err = s.billingData.GetBillableCustomerIDs(ctx, invoiceMonth)
	if err != nil {
		logger.Errorf("billable customer list (audit table) error occured, switching to v1, error : %v", err.Error())
		version = "v1" // fallback to v1
	}

	allAuditCustomers := append(append(auditCustomerIDs, auditZeroCostCustomerIDs...), auditStandAloneCostCustomerIDs...)

	if len(allAuditCustomers) <= 0 {
		logger.Errorf("no customers were found in the billable customers list, switching to V1 (unified query)")
		version = "v1"
	}

	if version == "" || version == v1 || validateWithOldLogic {
		assetIDs, err := s.billingData.GetBillableAssetIDs(ctx, invoiceMonth)
		if err != nil {
			return err
		}

		unifiedTableCustomerIDs, err = s.assetSettingsDAL.GetCustomersForAssets(ctx, assetIDs)
		if err != nil {
			// we get an error if we only found some assets/customers, we still want to continue with what we found
			if unifiedTableCustomerIDs == nil {
				return err
			}

			logger.Warningf("only found some customers for assets, continuing,  err: %v ", err)
		}

		logger.Infof("unified table query : billable customer list : %v", unifiedTableCustomerIDs)

		if len(unifiedTableCustomerIDs) <= 0 {
			return errors.New("no customers with AWS assets were found")
		}

		logger.Infof("unified customers minus audit customer : %v", findDifference(unifiedTableCustomerIDs, allAuditCustomers))
		logger.Infof("audit customers minus unified customer : %v", findDifference(auditCustomerIDs, unifiedTableCustomerIDs))
	}

	if version == v2 || version == "" {
		logger.Infof("setting auditCustomerIDs for invoicing run")
		customerIDs = mergeSlices(allAuditCustomers, unifiedTableCustomerIDs)
		logger.Infof("unified+audit customers(with deduplication) : %v", customerIDs)
	} else {
		logger.Infof("setting unifiedTableCustomerIDs for invoicing run")
		customerIDs = unifiedTableCustomerIDs
	}

	yearMonthDay := invoiceMonth.Format(times.YearMonthDayLayout)

	for _, customerID := range customerIDs {
		logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", 0, "UpdateAWSInvoicingData-Analytics-AllCustomers", "creating invoicing task run at "+time.Now().Format("2006-01-02 15:04:05"))
		t := BillingTaskAmazonWebServicesAnalytics{
			InvoiceMonth:         yearMonthDay,
			Version:              version,
			ValidateWithOldLogic: validateWithOldLogic,
			Dry:                  dry,
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/invoicing/amazon-web-services/customer/%s", customerID),
			Queue:  common.TaskQueueInvoicingAnalytics,
		}

		if _, err = s.cloudTaskClient.CreateTask(ctx, config.Config(t)); err != nil {
			logger.Errorf("UpdateAmazonWebServicesInvoicingData, failed to create task for customer: %s, error: %v", customerID, err)
			logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", 0, "UpdateAWSInvoicingData-Analytics-AllCustomers", "failed to creating invoicing task run at "+time.Now().Format("2006-01-02 15:04:05"))
			return err
		}
	}

	return nil
}

func findDifference(first, second []string) []string {
	var difference []string
	for _, v := range first {
		if !slices.Contains(second, v) {
			difference = append(difference, v)
		}
	}

	return difference
}

func mergeSlices(a, b []string) []string {
	target := append(a, b...)
	sort.Strings(target)
	return slices.Compact(target)
}

func (s *AnalyticsAWSInvoicingService) AmazonWebServicesInvoicingDataWorker(ginCtx *gin.Context, customerID, invoiceMonthInput string, dry bool) error {
	logger := s.loggerProvider(ginCtx)
	logBatch := time.Now().UTC().Truncate(time.Hour * 6).Format(time.DateTime)

	invoiceMonth, err := s.invoiceMonthParser.GetInvoiceMonth(invoiceMonthInput)
	if err != nil {
		return err
	}

	logger.Infof("Starting Analytics AmazonWebServicesInvoicingDataWorker for customer %s invoiceMonth %s DryMode=%v", customerID, invoiceMonth, dry)
	logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", 0, "UpdateAWSInvoicingData-Analytics-SingleCustomer", fmt.Sprintf("running invoicing task for month %v dry mode %v", invoiceMonth.Format(time.DateTime), dry))

	// fetch a list of all Flexsave accounts
	flexsaveAccounts, err := s.flexsaveAPI.ListFlexsaveAccountsWithCache(ginCtx, time.Minute*30)
	if err != nil {
		return err
	}

	if !sort.StringsAreSorted(flexsaveAccounts) {
		sort.Slice(flexsaveAccounts, func(i, j int) bool { return flexsaveAccounts[i] < flexsaveAccounts[j] })
	}

	costSavingsLineItems, accountIDs, err := s.billingData.GetCustomerBillingData(ginCtx, customerID, invoiceMonth)
	if err != nil {
		return err
	}

	// NB only obtain 'assets' reference, don't query - this should function even if an asset is removed
	accountIDtoAssetRefMap := make(map[string]*firestore.DocumentRef)
	accountIDtoAssetSettingsMap := make(map[string]*assetpkg.AWSAssetSettings)
	erroredAccounts := make([]string, 0)
	validAccountIDs := make([]string, 0)
	customerFlexsaveAccounts := make([]string, 0)
	marketplaceSDRefMap := make(map[string]string)

	for _, accountID := range accountIDs {
		// if this is a Flexsave account, skip asset mapping creation
		fsAccIdx := sort.SearchStrings(flexsaveAccounts, accountID)

		isFlexsaveAccount := fsAccIdx < len(flexsaveAccounts) && flexsaveAccounts[fsAccIdx] == accountID
		if isFlexsaveAccount {
			customerFlexsaveAccounts = append(customerFlexsaveAccounts, accountID)
			continue
		}

		assetID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, accountID)
		assetRef := s.assetsDAL.GetRef(ginCtx, assetID)

		assetSettings, err := s.assetSettingsDAL.GetAWSAssetSettings(ginCtx, assetID)
		if err != nil {
			logger.Debugf(" customer: %v: asset not configured, id: %v", customerID, assetID)

			erroredAccounts = append(erroredAccounts, accountID)
		} else {
			accountIDtoAssetRefMap[accountID] = assetRef
			accountIDtoAssetSettingsMap[accountID] = assetSettings

			validAccountIDs = append(validAccountIDs, accountID)
		}
	}

	// restrict accountIDs to valid accounts with asset settings
	accountIDs = validAccountIDs

	if len(accountIDtoAssetSettingsMap) <= 0 {
		logger.Warningf("customer %s: assetSettings not found (either bq returned no data Or data does not contain valid accounts)", customerID)
		logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", 0, "UpdateAWSInvoicingData-Analytics-SingleCustomer", "completed invoicing task run with warning - query returned no valid accounts")
		return nil
	}

	accountIDToSpendMap := make(map[string]float64)
	accountToCreditAllocation := make(map[string]map[string]float64)

	marketplaceAccountIDs := make(map[string]bool)
	marketplaceSpendMap := make(map[string]float64)
	marketplaceCreditAllocation := make(map[string]map[string]float64) // map["account_serviceDescription"]["creditRef"]

	accountToFlexsaveComputeNegationsMap := make(map[string]float64)
	accountToFlexsaveAdjustmentsMap := make(map[string]float64)
	accountToFlexsaveSagemakerNegationsMap := make(map[string]float64)
	accountToFlexsaveRDSNegationsMap := make(map[string]float64)
	accountToFlexsaveCreditsMap := make(map[string]float64)
	accountToFlexsaveManagementCostsMap := make(map[string]float64)
	accountToFlexsaveRDSChargesMap := make(map[string]float64)

	for _, accountID := range accountIDs {
		accountIDToSpendMap[accountID] = 0
		accountToCreditAllocation[accountID] = make(map[string]float64)
		accountToFlexsaveComputeNegationsMap[accountID] = 0
		accountToFlexsaveAdjustmentsMap[accountID] = 0
		accountToFlexsaveSagemakerNegationsMap[accountID] = 0
		accountToFlexsaveRDSNegationsMap[accountID] = 0
	}

	customerRef := s.customers.GetRef(ginCtx, customerID)

	credits, err := s.common.GetAmazonWebServicesCredits(ginCtx, invoiceMonth, customerRef, accountIDs)
	if err != nil {
		return err
	}

	yearMonth := invoiceMonth.Format(times.YearMonthLayout)

	erroredAccountDetails := map[string]float64{}

	// missing data for some days is valid (customer joining in the middle of the month, their usage going to 0, etc)
	for day, dailyCostSavingsLineItems := range costSavingsLineItems {
		for costAndSavingsLineItemKey, eachLineItem := range dailyCostSavingsLineItems {
			accountID := costAndSavingsLineItemKey.AccountID

			if slice.Contains(customerFlexsaveAccounts, accountID) {
				// check if this is a valid Flexsave account
				if slice.Contains(accountIDs, costAndSavingsLineItemKey.PayerAccountID) {
					if costAndSavingsLineItemKey.CostType == CostTypeCredit && costAndSavingsLineItemKey.Label == SkuComputeSp3yrNoUpfrontFmt {
						accountToFlexsaveCreditsMap[costAndSavingsLineItemKey.PayerAccountID] += eachLineItem.Costs
					} else if costAndSavingsLineItemKey.CostType == CostTypeFlexsaveRDSCharges {
						accountToFlexsaveRDSChargesMap[costAndSavingsLineItemKey.PayerAccountID] += eachLineItem.Costs
					} else {
						accountToFlexsaveManagementCostsMap[costAndSavingsLineItemKey.PayerAccountID] += eachLineItem.Costs
					}
				} else {
					errText := fmt.Sprintf("invoice failed - could not allocate costs of value %v for Flexsave account %v since payer %v does not belong to customer %v", eachLineItem.Costs, accountID, costAndSavingsLineItemKey.PayerAccountID, customerID)
					logger.Errorf(errText)

					erroredAccountDetails[accountID] += eachLineItem.Costs
					erroredAccounts = append(erroredAccounts, accountID)
					//return err
				}
			} else {
				// check if this is an expected accountID
				if assetSettings, ok := accountIDtoAssetSettingsMap[accountID]; ok {
					assetRef := accountIDtoAssetRefMap[accountID]

					if costAndSavingsLineItemKey.IsMarketplace {
						_, ok1 := marketplaceSDRefMap[costAndSavingsLineItemKey.MarketplaceSD]
						if !ok1 {
							marketplaceSDRefMap[costAndSavingsLineItemKey.MarketplaceSD] = "marketplace_" + FNV32a(costAndSavingsLineItemKey.MarketplaceSD)
						}

						decoratedAccountID := accountID + delimiter + marketplaceSDRefMap[costAndSavingsLineItemKey.MarketplaceSD]
						_, ok2 := marketplaceSpendMap[decoratedAccountID]
						if !ok2 {
							marketplaceAccountIDs[accountID] = true
							marketplaceSpendMap[decoratedAccountID] = 0
							marketplaceCreditAllocation[decoratedAccountID] = make(map[string]float64)
						}
						s.common.CalculateSpendAndCreditsData(yearMonth, decoratedAccountID, day, eachLineItem.Costs, assetSettings.Entity, assetRef, credits, marketplaceSpendMap, marketplaceCreditAllocation)
					} else {
						s.common.CalculateSpendAndCreditsData(yearMonth, accountID, day, eachLineItem.Costs, assetSettings.Entity, assetRef, credits, accountIDToSpendMap, accountToCreditAllocation)
						accountToFlexsaveComputeNegationsMap[accountID] += eachLineItem.FlexsaveComputeNegations
						accountToFlexsaveAdjustmentsMap[accountID] += eachLineItem.FlexsaveAdjustments
						accountToFlexsaveSagemakerNegationsMap[accountID] += eachLineItem.FlexsaveSagemakerNegations
						accountToFlexsaveRDSNegationsMap[accountID] += eachLineItem.FlexsaveRDSNegations
					}
				} else {
					errText := fmt.Sprintf("invoice failed - could not allocate costs of value %v for account %v payer %v as no asset/assetSettings doc ref found for customer %v", eachLineItem.Costs, accountID, costAndSavingsLineItemKey.PayerAccountID, customerID)
					logger.Errorf(errText)

					erroredAccountDetails[accountID] += eachLineItem.Costs
					erroredAccounts = append(erroredAccounts, accountID)
					//return err
				}
			}
		}
	}

	if len(marketplaceSpendMap) > 0 {
		logger.Infof("Marketplace Customer %v  \nmarketplaceSDRefMap %v \nspendMap %v \nmarketplaceCreditsMap %v", customerID, marketplaceSDRefMap, marketplaceSpendMap, marketplaceCreditAllocation)
	}

	// check if invoice is ready to be exported (store as 'verified')
	ready, err := s.billingData.GetCustomerInvoicingReadiness(ginCtx, customerID, invoiceMonth, s.invoiceMonthParser.GetInvoicingDaySwitchOver())
	if err != nil {
		logger.Errorf("GetCustomerInvoicingReadiness err: %v", err)

		ready = false
	}

	sessionID := ""
	snapshotCustomerBillingTable := false

	if ready {
		issued, err := s.billingData.HasCustomerInvoiceBeenIssued(ginCtx, customerID, invoiceMonth)
		if err != nil {
			logger.Errorf("HasCustomerInvoiceBeenIssued err: %v", err)
		} else {
			if !issued {
				sessionID = s.billingData.GetCustomerBillingSessionID(ginCtx, customerID, invoiceMonth)
				snapshotCustomerBillingTable = true
			} else {
				logger.Debugf("customer %s invoice already issued - no snapshot created", customerID)
			}
		}
	} else {
		logger.Debugf("customer %s, invoiceMonth %s, billing data not ready - no snapshot created", customerID, invoiceMonth.Format("2006-01"))
	}

	totalSpend := 0.0
	totalFlexsaveComputeNegations := 0.0
	totalFlexsaveSagemakerNegations := 0.0
	totalFlexsaveRDSNegations := 0.0
	totalFlexsaveRDSCharges := 0.0
	totalFlexsaveAdjustments := 0.0
	validAndFoundAssetIDs := make([]string, 0)
	assetBillingData := make(map[*firestore.DocumentRef]interface{})

	totalManagementCosts := 0.0
	totalFlexsaveSpCredits := 0.0

	for accountID, spend := range accountIDToSpendMap {
		// skip errored accounts due to undefined asset settings
		if slice.Contains(erroredAccounts, accountID) {
			continue
		}

		assetRef := accountIDtoAssetRefMap[accountID]
		monthlyBillingAws := pkg.MonthlyBillingAmazonWebServices{
			Customer: customerRef,
			Verified: ready,
			Spend:    spend,
			Flexsave: &pkg.MonthlyBillingAwsFlexsave{
				FlexsaveComputeNegations:   accountToFlexsaveComputeNegationsMap[accountID],
				FlexsaveAdjustments:        accountToFlexsaveAdjustmentsMap[accountID],
				FlexsaveSagemakerNegations: accountToFlexsaveSagemakerNegationsMap[accountID],
				FlexsaveRDSNegations:       accountToFlexsaveRDSNegationsMap[accountID],
				FlexsaveRDSCharges:         accountToFlexsaveRDSChargesMap[accountID],
				FlexsaveSpCredits:          accountToFlexsaveCreditsMap[accountID],
				ManagementCosts:            accountToFlexsaveManagementCostsMap[accountID],
			},
			Credits:                 accountToCreditAllocation[accountID],
			InvoiceMonth:            invoiceMonth.Format(times.YearMonthLayout),
			Type:                    common.Assets.AmazonWebServices,
			CustBillingTblSessionID: sessionID,
		}

		if _, ok := marketplaceAccountIDs[accountID]; ok {
			if monthlyBillingAws.MarketplaceConstituents == nil {
				monthlyBillingAws.MarketplaceConstituentsRef = flip(marketplaceSDRefMap)
				monthlyBillingAws.MarketplaceConstituents = map[string]pkg.MarketplaceConstituent{}
				monthlyBillingAws.MarketplaceConstituents["marketplace_none"] = pkg.MarketplaceConstituent{
					Spend:   spend,
					Credits: map[string]float64{},
				}
				maps.Copy(monthlyBillingAws.MarketplaceConstituents["marketplace_none"].Credits, accountToCreditAllocation[accountID])
				monthlyBillingAws.MarketplaceConstituentsRef["marketplace_none"] = "excluding Marketplace costs"
			}

			for k, v := range marketplaceSpendMap {
				constituent := pkg.MarketplaceConstituent{}
				mktplaceKeyParts := strings.Split(k, delimiter)
				if mktplaceKeyParts[0] == accountID {
					SDName := mktplaceKeyParts[1]
					constituent.Spend = v
					constituent.Credits = marketplaceCreditAllocation[k]
					monthlyBillingAws.MarketplaceConstituents[SDName] = constituent

					monthlyBillingAws.Spend += v
					spend += v

					for creditID, creditAmt := range marketplaceCreditAllocation[k] {
						monthlyBillingAws.Credits[creditID] += creditAmt
					}
				}
			}
		}

		totalManagementCosts += accountToFlexsaveManagementCostsMap[accountID]
		totalFlexsaveSpCredits += accountToFlexsaveCreditsMap[accountID]

		assetBillingData[assetRef] = monthlyBillingAws
		totalSpend += spend
		totalFlexsaveComputeNegations += accountToFlexsaveComputeNegationsMap[accountID]
		totalFlexsaveSagemakerNegations += accountToFlexsaveSagemakerNegationsMap[accountID]
		totalFlexsaveRDSNegations += accountToFlexsaveRDSNegationsMap[accountID]
		totalFlexsaveAdjustments += accountToFlexsaveAdjustmentsMap[accountID]
		totalFlexsaveRDSCharges += accountToFlexsaveRDSChargesMap[accountID]
		validAndFoundAssetIDs = append(validAndFoundAssetIDs, fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, accountID))
		logger.Debugf(
			"customerID %v accountID %s verified: %v, gross spend: %v, Flexsave negations - Compute: %v, SageMaker: %v, RDS: %v, Adjustments: %v",
			customerID,
			accountID,
			ready,
			spend-(accountToFlexsaveComputeNegationsMap[accountID]+accountToFlexsaveSagemakerNegationsMap[accountID]+accountToFlexsaveRDSNegationsMap[accountID]+accountToFlexsaveAdjustmentsMap[accountID]),
			accountToFlexsaveComputeNegationsMap[accountID],
			accountToFlexsaveSagemakerNegationsMap[accountID],
			accountToFlexsaveRDSNegationsMap[accountID],
			accountToFlexsaveAdjustmentsMap[accountID],
		)
	}

	logger.Debugf("customerId %v totalSpend (with managementCosts & flexsaveSpCredits) %v", customerID, totalSpend)
	logger.Debugf("customerId %v totalManagementCosts %v", customerID, totalManagementCosts)
	logger.Debugf("customerId %v totalFlexsaveSpCredits %v", customerID, totalFlexsaveSpCredits)
	logger.Debugf("customerId %v totalFlexsaveComputeNegations %v", customerID, totalFlexsaveComputeNegations)
	logger.Debugf("customerId %v totalFlexsaveSagemakerNegations %v", customerID, totalFlexsaveSagemakerNegations)
	logger.Debugf("customerId %v totalFlexsaveRDSNegations %v", customerID, totalFlexsaveRDSNegations)
	logger.Debugf("customerId %v totalFlexsaveRDSCharges %v", customerID, totalFlexsaveRDSCharges)
	logger.Debugf("customerId %v totalFlexsaveAdjustments %v", customerID, totalFlexsaveAdjustments)
	logger.Debugf("customerId %v processedAssetsList %v", customerID, validAndFoundAssetIDs)
	logger.Debugf("customerId %v customerFlexsaveAccounts %v", customerID, customerFlexsaveAccounts)
	logger.Debugf("customerId %v erroredAccounts %v", customerID, erroredAccounts)
	logger.Debugf("customerId %v flexsaveRDSCharge %v", customerID, accountToFlexsaveRDSChargesMap)
	logger.Debugf("customerId %v flexsaveAwsManagementCosts %v", customerID, accountToFlexsaveManagementCostsMap)
	logger.Debugf("customerId %v flexsaveAwsCredits %v", customerID, accountToFlexsaveCreditsMap)
	logger.Debugf("customerId %v accountToFlexsaveCreditsMap %v", customerID, accountToFlexsaveCreditsMap)

	logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", totalSpend+totalManagementCosts+totalFlexsaveSpCredits, "UpdateAWSInvoicingData-Analytics-SingleCustomer", fmt.Sprintf("invoicing costs for month %v dry mode %v", invoiceMonth.Format(time.DateTime), dry))

	if len(erroredAccountDetails) > 0 {
		var retErrs []error
		for errAcc, amt := range erroredAccountDetails {
			retErrs = append(retErrs, fmt.Errorf("invoice failed - could not allocate costs of value %v for account %v as no asset/assetSettings doc ref found for customer %v", amt, errAcc, customerID))
		}

		logger.Debugf("customerId %v aborting invoicing, Error occurred, unconfigured aws assets found in customer billing data", customerID)

		return errors.Join(retErrs...)
	}

	allCustomerAssets, err := s.monthlyBillingDataDAL.GetCustomerAWSAssetIDtoMonthlyBillingData(ginCtx, customerRef, invoiceMonth, true)
	if err != nil {
		return err
	}

	for customerAssetID := range allCustomerAssets {
		if slice.Contains(validAndFoundAssetIDs, customerAssetID) {
			continue
		}

		assetRef := s.assetsDAL.GetRef(ginCtx, customerAssetID)
		assetBillingData[assetRef] = nil
	}

	if !dry {
		err = s.monthlyBillingDataDAL.BatchUpdateMonthlyBillingData(ginCtx, yearMonth, assetBillingData, true)
		if err != nil {
			return err
		}
		logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", totalSpend+totalManagementCosts+totalFlexsaveSpCredits, "UpdateAWSInvoicingData-Analytics-SingleCustomer", "costs committed to MBDA records")

		err = s.billingData.SaveCreditUtilizationToFS(ginCtx, invoiceMonth, credits)
		if err != nil {
			return err
		}

		if snapshotCustomerBillingTable {
			err := s.billingData.SnapshotCustomerBillingTable(ginCtx, customerID, invoiceMonth)
			if err != nil {
				logger.Warningf("SnapshotCustomerBillingTable error: %v", err)
			}
			logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", totalSpend+totalManagementCosts+totalFlexsaveSpCredits, "UpdateAWSInvoicingData-Analytics-SingleCustomer", "BQ table snapshot completed")
		}
	}

	return nil
}

func flip(refMap map[string]string) map[string]string {
	returnMap := make(map[string]string)
	for k, v := range refMap {
		returnMap[v] = k
	}

	return returnMap
}

func FNV32a(text string) string {
	algorithm := fnv.New32a()
	algorithm.Write([]byte(text))
	return strconv.FormatUint(uint64(algorithm.Sum32()), 10)
}

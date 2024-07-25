package service

import (
	"context"
	"strings"
	"time"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
)

var tierBySku = map[string]domain.OriginalGoogleTier{
	"B076-4B67-AFA3": domain.StandardTier,
	"E277-8566-63EE": domain.StandardTier,

	"B064-0606-E072": domain.Enhanced,
	"7517-EEE3-D1DD": domain.Enhanced,

	"F08D-670F-E528": domain.PremiumTier,
	"3ADC-4232-8F2F": domain.PremiumTier,
	"768B-9B76-8BFA": domain.PremiumTier,
	"92AA-79F0-B1C6": domain.PremiumTier,
	"39DA-470F-1873": domain.PremiumTier,
	"1D0C-C18F-A3E9": domain.PremiumTier,
	"A4ED-26C4-BE0A": domain.PremiumTier,
	"7625-C72D-58B1": domain.PremiumTier,
	"E4F5-0256-E0EE": domain.PremiumTier,
	"5D14-41DF-B7BF": domain.PremiumTier,
	"4E5E-B559-B417": domain.PremiumTier,
	"9C0B-F338-0D7C": domain.PremiumTier,
	"7EFE-705D-1818": domain.PremiumTier,
	"778D-93A5-F155": domain.PremiumTier,
	"5467-9D2D-5B98": domain.PremiumTier,
}

func (s *ContractService) UpdateGoogleCloudContractsSupport(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	l.Infof("UpdateGoogleCloudContractsSupport: starting to update google cloud contracts")

	now := time.Now()
	startDate, endDate := getBillingPeriod(7)

	billingRows, err := s.bigqueryDal.GetBillingAccountsSKU(ctx, startDate, endDate)
	if err != nil {
		return err
	}

	skuByAccountFromBilling := createAccountMapFromBilling(billingRows)

	contractsSnaps, err := s.contractsDAL.GetActiveGoogleCloudContracts(ctx)
	if err != nil {
		return err
	}

	tierFound := 0
	tierNotFound := 0
	contractWithoutAssets := 0

	inputs := make([]domain.UpdateSupportInput, 0)

	for _, docSnap := range contractsSnaps {
		var contract pkg.Contract
		if err := docSnap.DataTo(&contract); err != nil {
			l.Errorf("error reading contract %s", docSnap.Ref.ID)
			continue
		}

		contract.ID = docSnap.Ref.ID
		if len(contract.Assets) == 0 {
			contractWithoutAssets++
			continue
		}

		tier := getTierForContract(contract, skuByAccountFromBilling)
		if tier == domain.NoSupport {
			tierNotFound++
		} else {
			tierFound++

			l.Infof("tier FOUND for customerID: %s, contractID: %s, tier: %s", contract.Customer.ID, contract.ID, tier)
		}

		support := domain.GCPContractSupportInput{
			OriginalSupportTier: tier,
			UpdatedAt:           now,
		}

		inputs = append(inputs, domain.UpdateSupportInput{Ref: docSnap.Ref, Support: support})
	}

	if len(inputs) != 0 {
		if err = s.contractsDAL.UpdateContractSupport(ctx, inputs); err != nil {
			return err
		}
	}

	l.Infof("UpdateGoogleCloudContractsSupport: total active google contracts: %d, found tier: %d,  tier not found: %d, contractWithoutAssets: %d", len(contractsSnaps), tierFound, tierNotFound, contractWithoutAssets)

	return nil
}

func createAccountMapFromBilling(billingRows []domain.SKUBillingRecord) map[string][]string {
	skuByAccountFromBilling := make(map[string][]string)

	for _, row := range billingRows {
		skuByAccountFromBilling[row.BillingAccountID] = append(skuByAccountFromBilling[row.BillingAccountID], row.SKUID)
	}

	return skuByAccountFromBilling
}

func getTierForContract(contract pkg.Contract, skuByAccountFromBilling map[string][]string) domain.OriginalGoogleTier {
	var tier domain.OriginalGoogleTier

	for _, asset := range contract.Assets {
		accountID, _ := strings.CutPrefix(asset.ID, "google-cloud-") // asset.ID looks like google-cloud-00633D-6DBEBF-D9D41F

		skus, ok := skuByAccountFromBilling[accountID]
		if !ok {
			// no billing records for this account(asset)
			continue
		}

		for _, sku := range skus {
			tier, ok = tierBySku[sku]
			if ok {
				// returning the first found tier for the most recent SKU in case there are several
				return tier
			}
		}
	}

	return domain.NoSupport
}

func getBillingPeriod(daysAgo int) (string, string) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -daysAgo).Format(time.DateTime)
	endDate := now.Format(time.DateTime)

	return startDate, endDate
}

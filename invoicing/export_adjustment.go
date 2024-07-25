// This file contains the logic for creating specific Finance needs in the export invoices
package invoicing

import (
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
)

var specificLedgerMap = map[string]map[string]InvoiceDefaultsLedger{
	utils.SolveType: {
		"ISR": {
			TargetAccount:  "429",
			RevenueAccount: "2682",
		},
		"NO_ISR": {
			TargetAccount:  "429-1",
			RevenueAccount: "2684",
		},
	},
	utils.SolveAcceleratorType: {
		"ISR": {
			TargetAccount:  "431",
			RevenueAccount: "2682",
		},
		"NO_ISR": {
			TargetAccount:  "431-1",
			RevenueAccount: "2684",
		},
	},
	utils.NavigatorType: {
		"ISR": {
			TargetAccount:  "430",
			RevenueAccount: "2682",
		},
		"NO_ISR": {
			TargetAccount:  "430-1",
			RevenueAccount: "2684",
		},
	},
	"default": {
		"ISR": {
			TargetAccount:  "6100",
			RevenueAccount: "2682",
		},
		"NO_ISR": {
			TargetAccount:  "6200",
			RevenueAccount: "2684",
		},
	},
}

// The types Solve, Solve Accelerator and Navigator needs to have specific ledger accounts
// getISRRevenueAndTargetAccounts returns ISL revenue and target accounts based on export status and invoice type
func getISRRevenueAndTargetAccounts(isExportCustomer bool, invoiceType string) (string, string) {
	var companyCode string
	// set invoiceType to default if not found in specificLedgerMap
	if _, ok := specificLedgerMap[invoiceType]; !ok {
		invoiceType = "default"
	}

	if isExportCustomer {
		// NO_ISR
		companyCode = "NO_ISR"
	} else {
		// ISR
		companyCode = "ISR"
	}

	targetAccount := specificLedgerMap[invoiceType][companyCode].TargetAccount
	revenueAccount := specificLedgerMap[invoiceType][companyCode].RevenueAccount
	return revenueAccount, targetAccount
}

// BizOps requested to change "navigator" with "doit-navigator" and "solve" with "doit-solve" in the spreadsheet only
func mapAssetTypeToSpreadsheetAssetType(originalType string) string {
	if originalType == utils.NavigatorType {
		return utils.NavigatorInvoiceType
	}

	if originalType == utils.SolveType {
		return utils.SolveInvoiceType
	}

	if originalType == utils.SolveAcceleratorType {
		return utils.SolveAcceleratorInvoiceType
	}

	return originalType
}

// For customers that have 'Chicago' or IL in their BP's address, we need to use a different SKU for the invoice
// Both for AWS and GCP
func getIsChicagoBillingProfile(entity common.Entity) bool {

	if entity.BillingAddress.StateA != nil && strings.ToLower(*entity.BillingAddress.StateA) == "chicago" ||
		entity.BillingAddress.StateCode != nil && strings.ToLower(*entity.BillingAddress.StateCode) == "il" {
		return true
	}
	return false
}

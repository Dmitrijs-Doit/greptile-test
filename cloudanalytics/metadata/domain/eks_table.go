package domain

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
)

func GetMetadataEksTable(isCSP bool, customerID string, isStandalone bool) string {
	// Perform eks data union only for regular customers.
	if isCSP {
		return ""
	}

	selectEksFields := getSelectMetadataEksFields(isCSP, isStandalone)

	var whereClause string

	eksTable := fmt.Sprintf(
		"UNION ALL\n\t\tSELECT %s FROM %s%s",
		selectEksFields,
		querytable.GetFullEksTableName(customerID),
		whereClause,
	)

	return eksTable
}

func getSelectMetadataEksFields(isCSP bool, isStandalone bool) string {
	fields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectName,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldUsageStartTime,
		domainQuery.FieldUsageEndTime,
		domainQuery.FieldLabels,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldCost,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
		domainQuery.FieldUsage,
		domainQuery.FieldInvoice,
		domainQuery.FieldCostType,
	}
	if !isCSP && !isStandalone {
		fields = append(fields,
			domainQuery.FieldIsMarketplace,
			domainQuery.FieldBillingReportGCP,
			domainQuery.FieldProject,
			domainQuery.FieldOperation,
			domainQuery.FieldResourceID,
			cloudProviderParam,
		)
	}
	
	if isStandalone {
		fields = []string{
			domainQuery.FieldNulletl,
			domainQuery.FieldBillingAccountID,
			domainQuery.FieldProjectID,
			domainQuery.FieldServiceDescription,
			domainQuery.FieldServiceID,
			domainQuery.FieldSKUDescription,
			domainQuery.FieldSKUID,
			domainQuery.FieldUsageDateTime,
			domainQuery.FieldUsageStartTime,
			domainQuery.FieldUsageEndTime,
			domainQuery.FieldProject,
			domainQuery.FieldLabels,
			domainQuery.FieldSystemLabels,
			domainQuery.FieldLocation,
			domainQuery.FieldExportTime,
			domainQuery.FieldCost,
			domainQuery.FieldCurrency,
			domainQuery.FieldCurrencyRate,
			domainQuery.FieldUsage,
			domainQuery.FieldInvoice,
			domainQuery.FieldCostType,
			domainQuery.FieldReport,
			domainQuery.FieldResourceID,
			domainQuery.FieldOperation,
			domainQuery.FieldAWSMetric,
			domainQuery.FieldRowID,
			domainQuery.FieldDescription,
			domainQuery.FieldCustomerType,
			domainQuery.FieldIsMarketplace,
			domainQuery.FieldCustomerID,
			cloudProviderParam,
		}
	}

	return strings.Join(fields, consts.Comma)
}

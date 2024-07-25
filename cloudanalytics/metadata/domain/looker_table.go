package domain

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
)

const cloudProviderParam string = "@cloud_provider AS cloud_provider"

func GetMetadataLookerTable(isCSP bool) string {
	// Perform looker data union only for regular customers.
	if isCSP {
		return ""
	}

	selectLookerFields := getSelectMetadataLookerFields(isCSP)

	whereClause := " WHERE billing_account_id = @billing_account_id"
	if isCSP {
		whereClause = ""
	}

	lookerTable := fmt.Sprintf(
		"UNION ALL\n\t\tSELECT %s FROM %s%s",
		selectLookerFields,
		querytable.GetFullLookerTableName(),
		whereClause,
	)

	return lookerTable
}

func getSelectMetadataLookerFields(isCSP bool) string {
	fields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldUsageStartTime,
		domainQuery.FieldUsageEndTime,
		domainQuery.NullProject,
		domainQuery.FieldLabels,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldUsage,
		domainQuery.FieldCredits,
		domainQuery.FieldInvoice,
		domainQuery.FieldCostType,
	}

	if isCSP {
		fields = append(fields,
			domainQuery.NullAdjustmentInfo,
			domainQuery.NullTags,
			domainQuery.NullCostAtList,
			domainQuery.NullCustomerType,
			domainQuery.NullGCPMetrics,
			domainQuery.NullResourceID,
			domainQuery.NullResourceGlobalID,
			domainQuery.FalseIsMarketplace,
			domainQuery.NullIsPreemptible,
			domainQuery.NullIsPremiumImage,
			domainQuery.NullExcludeDiscount,
			domainQuery.NullKubernetesClusterName,
			domainQuery.NullKubernetesNamespace,
			domainQuery.NullMarginCredits,
			domainQuery.NullPriceBook,
			domainQuery.NullDiscount,
			cspreport.FieldBillingReportCSPFullTable,
			domainQuery.NullClassification,
			domainQuery.NullPrimaryDomain,
			domainQuery.NullTerritory,
			domainQuery.NullPayeeCountry,
			domainQuery.NullPayerCountry,
			domainQuery.NullFieldSalesRepresentative,
			domainQuery.NullStrategicAccountManager,
			domainQuery.NullTechnicalAccountManager,
			domainQuery.NullCustomerSuccessManager,
			domainQuery.FieldProjectName,
			domainQuery.FieldProjectNumber,
			cloudProviderParam,
			domainQuery.NullOperation,
		)
	} else {
		fields = append(fields,
			domainQuery.NullTags,
			domainQuery.NullPrice,
			domainQuery.NullCostAtList,
			domainQuery.NullCustomerType,
			domainQuery.NullGCPMetrics,
			domainQuery.NullResourceID,
			domainQuery.NullResourceGlobalID,
			domainQuery.FalseIsMarketplace,
			domainQuery.NullIsPreemptible,
			domainQuery.NullIsPremiumImage,
			domainQuery.NullExcludeDiscount,
			domainQuery.NullKubernetesClusterName,
			domainQuery.NullKubernetesNamespace,
			domainQuery.NullPriceBook,
			domainQuery.NullDiscount,
			domainQuery.FieldReport,
			domainQuery.FieldProjectName,
			domainQuery.FieldProjectNumber,
			cloudProviderParam,
			domainQuery.NullOperation,
		)
	}

	return strings.Join(fields, consts.Comma)
}

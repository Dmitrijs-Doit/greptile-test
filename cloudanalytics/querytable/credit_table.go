package querytable

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

func GetCreditTable(customerID string, isCSP bool, fullTableMode bool) string {
	selectCreditFields := getSelectCreditsFields(isCSP, fullTableMode)
	creditTable := fmt.Sprintf(
		"\n\t\tSELECT %s FROM %s %s",
		selectCreditFields,
		GetFullCreditTableName(),
		CreditsWhereClause(customerID, isCSP),
	)

	return creditTable
}

func GetFullCreditTableName() string {
	return getFullTableName(googleCloudConsts.CreditsTable)
}

func CreditsWhereClause(customerID string, isCSP bool) string {
	// DO NOT use customer filter for CSP in order to see ALL credits
	if isCSP {
		return ""
	}

	return fmt.Sprintf(`WHERE %s = "%s"`, domainQuery.FieldCustomer, customerID)
}

func getSelectCreditsFields(isCSP bool, fullTableMode bool) string {
	fields := []string{
		domainQuery.FieldCloudProvider,
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldProjectName,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.NullOperation,
		domainQuery.NullProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.NullTags,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldPricingUsage,
		domainQuery.FieldCostType,
		domainQuery.NullIsMarketplace,
		domainQuery.FieldInvoice,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
	}

	if isCSP {
		if fullTableMode {
			fields = append(fields, cspreport.FieldBillingReportCSPFullTable)
		} else {
			fields = append(fields, cspreport.FieldBillingReportCSPTableCompatability)
		}
	} else {
		fields = append(fields,
			domainQuery.FieldBillingReportLegacy,
			domainQuery.NullResourceGlobalID,
			domainQuery.NullResourceID,
			domainQuery.NullKubernetesClusterName,
			domainQuery.NullKubernetesNamespace,
		)
	}

	r := strings.Join(fields, consts.Comma)

	if isCSP {
		r += consts.Comma + cspreport.GetCspReportFields(true, fullTableMode, false)
	}

	return r
}

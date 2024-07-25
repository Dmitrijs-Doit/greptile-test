package querytable

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

func GetLookerTable(customerID string, isCSP bool, fullTableMode bool) string {
	selectLookerFields := getSelectLookerFields(isCSP, fullTableMode)
	lookerTable := fmt.Sprintf(
		"\n\t\tSELECT %s FROM %s %s",
		selectLookerFields,
		GetFullLookerTableName(),
		lookerWhereClause(customerID, isCSP),
	)

	return lookerTable
}

func GetFullLookerTableName() string {
	return getFullTableName(googleCloudConsts.LookerTable)
}

func lookerWhereClause(customerID string, isCSP bool) string {
	if isCSP {
		return ""
	}

	return fmt.Sprintf(`WHERE %s = "%s"`, domainQuery.FieldCustomer, customerID)
}

func getSelectLookerFields(isCSP bool, fullTableMode bool) string {
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
		domainQuery.FalseIsMarketplace,
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
		r += consts.Comma + cspreport.GetCspReportFields(
			true,
			fullTableMode,
			true,
		)
	}

	return r
}

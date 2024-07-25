package domain

import (
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func GetAggregatedQuery(allPartitions bool, billingAccountID string, isCSP bool, aggregationInterval string) string {
	var fullTableName string

	additionalFields := []string{
		queryDomain.FieldBillingAccountID,
		queryDomain.FieldProjectID,
		queryDomain.FieldServiceDescription,
		queryDomain.FieldServiceID,
		queryDomain.FieldSKUDescription,
		queryDomain.FieldSKUID,
		queryDomain.FieldCurrency,
		queryDomain.FieldCurrencyRate,
		queryDomain.FieldCostType,
		queryDomain.FieldCustomerType,
		queryDomain.FieldIsMarketplace,
	}

	if isCSP {
		fullTableName = GetFullCSPFullBillingTable()

		additionalFields = append(additionalFields, "discount.is_commitment")
	} else {
		additionalFields = append(additionalFields,
			queryDomain.FieldResourceID,
			queryDomain.FieldResourceGlobalID,
			queryDomain.FieldKubernetesClusterName,
			queryDomain.FieldKubernetesNamespace,
		)
		fullTableName = GetFullCustomerBillingTable(billingAccountID, "")
	}

	data := tableMgmtDomain.AggregatedQueryData{
		Cloud:                        common.Assets.GoogleCloud,
		BillingAccountID:             billingAccountID,
		FullBillingDataTableFullName: fullTableName,
		AdditionalFields:             additionalFields,
		AggregationInterval:          aggregationInterval,
		IsCSP:                        isCSP,
	}

	return service.GetAggregatedTableQuery(allPartitions, &data)
}

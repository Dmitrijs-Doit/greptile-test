package tablemanagement

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func getAggregatedQuery(allPartitions bool, suffix string, isCSP bool, aggregationInterval string) string {
	var fullTableName string

	additionalFields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
		domainQuery.FieldCostType,
		domainQuery.FieldCustomerType,
		domainQuery.FieldOperation,
		domainQuery.FieldIsMarketplace,
	}

	if isCSP {
		fullTableName = microsoftazure.GetFullCSPFullBillingTable()

		additionalFields = append(additionalFields, "discount.is_commitment")
	} else {
		additionalFields = append(additionalFields, domainQuery.FieldResourceID)
		fullTableName = fmt.Sprintf(consts.FullTableTemplate, microsoftazure.GetBillingProject(), microsoftazure.GetCustomerBillingDataset(suffix), microsoftazure.GetCustomerBillingTable(suffix, ""))
	}

	data := domain.AggregatedQueryData{
		Cloud:                        common.Assets.MicrosoftAzure,
		FullBillingDataTableFullName: fullTableName,
		AdditionalFields:             additionalFields,
		AggregationInterval:          aggregationInterval,
		IsCSP:                        isCSP,
	}

	return service.GetAggregatedTableQuery(allPartitions, &data)
}

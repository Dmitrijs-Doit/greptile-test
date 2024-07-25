package querytable

import (
	"fmt"
	"strings"

	awsCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func GetEksTable(customerID string, isCSP bool, fullTableMode bool) string {
	selectEksFields := getSelectEksFields(isCSP, fullTableMode)
	EksTable := fmt.Sprintf(
		"\n\t\tSELECT 'amazon-web-services' AS cloud_provider, %s FROM %s %s",
		selectEksFields,
		GetFullEksTableName(customerID),
		eksWhereClause(customerID, isCSP),
	)

	return EksTable
}

func GetFullEksTableName(customerID string) string {
	return getEksFullTableName(awsCloudConsts.EksTable + customerID)
}

func eksWhereClause(customerID string, isCSP bool) string {
	return "" //fmt.Sprintf(`WHERE %s = "%s"`, domainQuery.FieldCustomer, customerID)
}

func getSelectEksFields(isCSP bool, fullTableMode bool) string {
	fields := []string{
		domainQuery.FieldBillingAccountID,
		domainQuery.FieldProjectID,
		domainQuery.FieldProjectNumber,
		domainQuery.FieldProjectName,
		domainQuery.FieldServiceDescription,
		domainQuery.FieldServiceID,
		domainQuery.FieldSKUDescription,
		domainQuery.FieldSKUID,
		domainQuery.FieldOperation,
		domainQuery.FieldAWSProject,
		domainQuery.FieldUsageDateTime,
		domainQuery.FieldLabels,
		domainQuery.NullTags,
		domainQuery.FieldSystemLabels,
		domainQuery.FieldLocation,
		domainQuery.FieldExportTime,
		domainQuery.FieldPricingUsage,
		domainQuery.FieldCostType,
		domainQuery.FieldIsMarketplace,
		domainQuery.FieldInvoice,
		domainQuery.FieldCurrency,
		domainQuery.FieldCurrencyRate,
	}

	if !isCSP {
		fields = append(fields,
			domainQuery.FieldBillingReport,
			domainQuery.NullResourceGlobalID,
			domainQuery.FieldResourceID,
			domainQuery.NullKubernetesClusterName,
			domainQuery.NullKubernetesNamespace,
		)
	}

	return strings.Join(fields, consts.Comma)
}

func getEksFullTableName(tableName string) string {
	projectID := GetEksProject()

	return fmt.Sprintf(consts.FullTableTemplate, projectID, awsCloudConsts.EksDataset, tableName)
}

func GetEksTableName(customerID string) string {
	return fmt.Sprintf("%s%s", awsCloudConsts.EksTable, customerID)
}

func GetEksProject() string {
	projectID := awsCloudConsts.EksProjectDev
	if common.Production {
		projectID = awsCloudConsts.EksProjectProd
	}

	return projectID
}

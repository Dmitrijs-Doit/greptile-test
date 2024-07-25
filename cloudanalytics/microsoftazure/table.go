package microsoftazure

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// Project info
const (
	BillingProjectProd   string = "doitintl-cmp-azure-data"
	BillingProjectDev    string = "doitintl-cmp-azure-data-dev"
	BillingDatasetPrefix string = "azure_billing"
	BillingTablePrefix   string = "doitintl_billing_export_v1"
)

func GetBillingProject() string {
	if common.Production {
		return BillingProjectProd
	}

	return BillingProjectDev
}

func GetCustomerBillingDataset(suffix string) string {
	return fmt.Sprintf("%s_%s", BillingDatasetPrefix, suffix)
}

// GetCustomerBillingTable returns the table name for the Azure customer
func GetCustomerBillingTable(customerID string, tableSuffix string) string {
	table := fmt.Sprintf("%s_%s", BillingTablePrefix, customerID)
	if tableSuffix != "" {
		return table + "_" + tableSuffix
	}

	return table
}

func GetFullCustomerBillingTable(customerID string, aggInterval string) string {
	return fmt.Sprintf(consts.FullTableTemplate, GetBillingProject(), GetCustomerBillingDataset(customerID), GetCustomerBillingTable(customerID, aggInterval))
}

/** CSP Utils **/

func getCSPBillingSuffix() string {
	return domainQuery.CSPCustomerID
}

func GetCSPFullBillingTableName() string {
	return GetCustomerBillingTable(getCSPBillingSuffix(), domainQuery.BillingTableSuffixFull)
}

func GetCSPBillingTableName() string {
	return GetCustomerBillingTable(getCSPBillingSuffix(), "")
}

func GetCSPBillingDataset() string {
	return GetCustomerBillingDataset(getCSPBillingSuffix())
}

func GetFullCSPFullBillingTable() string {
	return fmt.Sprintf(consts.FullTableTemplate, GetBillingProject(), GetCSPBillingDataset(), GetCSPFullBillingTableName())
}

func GetFullCSPBillingTable() string {
	return fmt.Sprintf(consts.FullTableTemplate, GetBillingProject(), GetCSPBillingDataset(), GetCSPBillingTableName())
}

func GetRawBillingTable() string {
	return fmt.Sprintf(consts.FullTableTemplate, GetBillingProject(), BillingDatasetPrefix, BillingTablePrefix)
}

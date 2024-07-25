package domain

import (
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// Project info utils
const (
	BillingDataset               string = "gcp_billing"
	BillingProjectDev            string = "doitintl-cmp-gcp-data"
	BillingProjectProd           string = "doitintl-cmp-gcp-data"
	BillingStandaloneProjectProd string = "doitintl-cmp-gcp-data"
	BillingStandaloneProjectDev  string = "doitintl-cmp-dev"
	BillingStandaloneDataset     string = "gcp_billing_standalone"
	FlexsaveBillingTable         string = "cloud_analytics_flexsave_billing"
	FlexsaveDataset              string = "gcp_custom_billing"
	GKEDataset                   string = "customer_gke_usage"
	ResellerBillingExportProject string = "billing-explorer"
	ResellerBillingExportDataset string = "gcp"
)

type UpdateBillingAccountsTableInput struct {
	// modes:
	// none (empty, default): update everything, metadata depending of hour
	// "tables": create only regular table update tasks
	// "metadata": create only metadata update tasks
	// "aggregate": create only aggregate table tasks
	Mode string `json:"mode"`

	// update from specific date, when left empty and combined with "mode options will
	// update "all partitions"
	FromDate string `json:"from"`

	// update specific partition
	FromDateNumPartitions int `json:"numPartitions"`

	// Use to update/debug specific asset ids
	Assets []string `json:"assets"`

	// Use to skip specific asset ids
	ExceptAssets []string `json:"exceptAssets"`
}

func GetRawBillingTableName(useDetailedTable bool) string {
	var suffix string

	if useDetailedTable {
		suffix = "_v1"
	}

	return "gcp_raw_billing" + suffix
}

func GetBillingProject() string {
	if common.Production {
		return BillingProjectProd
	}

	return BillingProjectDev
}

func GetStandaloneProject() string {
	if common.Production {
		return BillingStandaloneProjectProd
	}

	return BillingStandaloneProjectDev
}

func GetCustomerBillingDataset(billingAccount string) string {
	return fmt.Sprintf("gcp_billing_%s", strings.Replace(billingAccount, "-", "_", -1))
}

func GetCustomerBillingTable(billingAccount string, tableSuffix string) string {
	var table string

	billingAccount = strings.Replace(billingAccount, "-", "_", -1)
	if common.Production {
		table = fmt.Sprintf("doitintl_billing_export_v1_%s", billingAccount)
	} else {
		table = fmt.Sprintf("doitintl_billing_export_v1beta_%s", billingAccount)
	}

	if tableSuffix != "" {
		return table + "_" + tableSuffix
	}

	return table
}

func GetFullCustomerBillingTable(billingAccount string, aggInterval string) string {
	return fmt.Sprintf("%s.%s.%s", GetBillingProject(), GetCustomerBillingDataset(billingAccount), GetCustomerBillingTable(billingAccount, aggInterval))
}

/** CSP Utils **/

func GetCSPBillingDataset() string {
	return GetCustomerBillingDataset(consts.MasterBillingAccount)
}

func GetCSPFullBillingTableName() string {
	return GetCustomerBillingTable(consts.MasterBillingAccount, queryDomain.BillingTableSuffixFull)
}

func GetCSPBillingTableName() string {
	return GetCustomerBillingTable(consts.MasterBillingAccount, "")
}

func GetFullCSPFullBillingTable() string {
	return fmt.Sprintf("%s.%s.%s", GetBillingProject(), GetCustomerBillingDataset(consts.MasterBillingAccount), GetCSPFullBillingTableName())
}

func GetFullCSPBillingTable() string {
	return fmt.Sprintf("%s.%s.%s", GetBillingProject(), GetCustomerBillingDataset(consts.MasterBillingAccount), GetCSPBillingTableName())
}

func GetBillingTableClustering() *bigquery.Clustering {
	return &bigquery.Clustering{Fields: []string{
		"project_id",
		"service_description",
		"sku_description",
	}}
}

func GetBillingSkusTableName() string {
	if common.Production {
		return "gcp_billing_skus_v1"
	}

	return "gcp_billing_skus_v1beta"
}

func GetPromotionalCreditsTableName() string {
	if common.Production {
		return "gcp_promotional_credits_v1"
	}

	return "gcp_promotional_credits_v1"
}

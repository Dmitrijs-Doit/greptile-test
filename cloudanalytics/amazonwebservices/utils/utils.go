package utils

import (
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

// Project info
const (
	BillingProjectProd   string = "doitintl-cmp-aws-data"
	BillingProjectDev    string = "cmp-aws-etl-dev"
	BillingDataset       string = "aws_billing"
	BillingTable         string = "doitintl_billing_export_v1"
	FullTableTemplate    string = "%s.%s.%s"
	moonactiveCustomerID string = "2Gi0e4pPA3wsfJNOOohW"
)

func GetBillingProject() string {
	if common.Production {
		return BillingProjectProd
	}

	return BillingProjectDev
}

func GetCustomerBillingDataset(suffix string) string {
	return fmt.Sprintf("aws_billing_%s", suffix)
}

// GetCustomerBillingTable returns the table name for AWS analytics customer
// customer ID may be CMP customer ID or CHT customer ID (until we deprecate CHT)
func GetCustomerBillingTable(customerID string, tableSuffix string) string {
	table := fmt.Sprintf("doitintl_billing_export_v1_%s", customerID)
	if tableSuffix != "" {
		return table + "_" + tableSuffix
	}

	return table
}

// FullCustomerBillingTableParams represents the params that should be passed to GetFullCustomerBillingTable
type FullCustomerBillingTableParams struct {
	Suffix              string
	CustomerID          string
	IsCSP               bool
	IsStandalone        bool
	AggregationInterval string
}

func GetFullCustomerBillingTable(params FullCustomerBillingTableParams) string {
	table := GetFullBillingTableName(params)
	// We should not try and union the line items table with CSP customer billing table
	if params.IsCSP || params.IsStandalone {
		return table
	}

	customerID := params.CustomerID

	return unionAWSLineItemsTable(table, customerID)
}

func GetFullBillingTableName(params FullCustomerBillingTableParams) string {
	return fmt.Sprintf(FullTableTemplate, GetBillingProject(), GetCustomerBillingDataset(params.Suffix), GetCustomerBillingTable(params.Suffix, params.AggregationInterval))
}

func GetAwsAssetsHistoryTableName() string {
	return fmt.Sprintf(FullTableTemplate, GetBillingProject(), "accounts", "accounts_history")
}

// GetDiscountsDatasetName ...
func GetDiscountsDatasetName() string {
	return "aws_billing_cmp"
}

func GetDiscountsTableName() string {
	if common.Production {
		return "aws_discounts_v1"
	}

	return "aws_discounts_v1beta"
}

func GetFullDiscountsTable() string {
	return fmt.Sprintf(FullTableTemplate, GetBillingProject(), GetDiscountsDatasetName(), GetDiscountsTableName())
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
	return fmt.Sprintf(FullTableTemplate, GetBillingProject(), GetCSPBillingDataset(), GetCSPFullBillingTableName())
}

func GetFullCSPBillingTable() string {
	return fmt.Sprintf(FullTableTemplate, GetBillingProject(), GetCSPBillingDataset(), GetCSPBillingTableName())
}

func GetRawBillingTable() string {
	return fmt.Sprintf("%s.%s.%s", GetBillingProject(), BillingDataset, BillingTable)
}

// unionAWSLineItemsTable returns the union between the customer billing table and the line items table from doitintl-cmp-global-data(-dev)
func unionAWSLineItemsTable(customerTable string, customerID string) string {
	fields := query.AWSLineItemsNonNullFields()

	q := `(
		SELECT
			` + fields + `, ` + domainQuery.FieldIsMarketplace + `, ` + domainQuery.FieldBillingReport + `, project, operation, resource_id
		FROM ` + customerTable + `
		UNION ALL
		SELECT
			` + fields + `, ` + domainQuery.NullIsMarketplace + `, ` +
		`ARRAY(
					SELECT AS STRUCT
					  cost,
					  usage,
					  savings,
					  CAST(NULL AS STRING) AS savings_description,
					  credit,
					  STRUCT('amortized_cost' AS key, IFNULL(cost,0.0) AS value, 'cost' AS type) AS ext_metric
					FROM
					  UNNEST(report)
				UNION ALL
					SELECT AS STRUCT
					  NULL AS cost,
					  NULL AS usage,
					  NULL AS savings,
					  CAST(NULL AS STRING) AS savings_description,
					  CAST(NULL AS STRING) as credit,
					  STRUCT('amortized_savings' AS key, IFNULL(savings,0.0) as value, 'cost' AS type) AS ext_metric  FROM
					  UNNEST(report)
			) AS report,
			NULL AS project,
			NULL AS operation,
			NULL AS resource_id
		FROM ` + querytable.GetAWSLineItemsTable() + `
		WHERE billing_account_id = "` + customerID + `")`

	return q
}

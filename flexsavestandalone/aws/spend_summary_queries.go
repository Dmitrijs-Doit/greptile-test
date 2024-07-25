package aws

import (
	"context"
	"errors"
	"net/http"
	"strings"

	c "github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/googleapi"
)

var (
	prod = "doitintl-cmp-aws-data"
	dev  = "cmp-aws-etl-dev"
)

var ErrNoTable = errors.New("no billing table found")

func (d *AwsStandaloneService) buildOnDemandCostEquivalentQuery(params BigQueryRequestParams) string {
	return queryReplacer(params).Replace(`
		SELECT
			sum(cost) as cost,TIMESTAMP(DATE_TRUNC(usage_date_time, DAY)) AS usage_date, label.value as payer_id
		FROM
			{env}.{customer_dataset}.{customer_table}, UNNEST(system_labels) AS label
		WHERE (label.value IN ({account_ids})
			 AND label.key='aws/payer_account_id')
		AND
			DATE(usage_start_time) >= DATETIME(@start)
			AND DATE(usage_start_time) < DATETIME(@end)
			AND (
				(operation LIKE '%%RunInstances%%' AND sku_description LIKE '%%Box%%'AND service_id = "AmazonEC2" AND cost_type IN ('Usage', 'Refund'))
				OR (service_id='AWSLambda' AND cost_type IN ('Usage')  and sku_description NOT LIKE '%%DataTransfer%%'  AND sku_description NOT LIKE '%%Out-Bytes%%' AND sku_description NOT LIKE '%%In-Bytes%%' )
				OR (operation = "FargateTask" AND  sku_description LIKE '%%Fargate%%' AND service_id = "AmazonECS" AND cost_type IN ('Usage', 'Refund'))
				OR (cost_type = "FlexsaveCoveredUsage")
		    )
			AND DATETIME(export_time)>=DATETIME(@start)
			AND DATETIME(export_time)<DATETIME(@end)
			GROUP BY usage_date, payer_id
	`)
}

func (d *AwsStandaloneService) buildStandaloneSavingsQuery(params BigQueryRequestParams) string {
	return queryReplacer(params).Replace(`
		SELECT
			SUM(cost) as cost,TIMESTAMP(DATE_TRUNC(usage_date_time, DAY)) AS usage_date, label.value as payer_id
		FROM
			{env}.{customer_dataset}.{customer_table}, UNNEST(system_labels) AS label
		WHERE (label.value IN ({account_ids})
			 AND label.key='aws/payer_account_id')
		AND
			DATE(usage_start_time) >= DATETIME(@start)
			AND DATE(usage_start_time) < DATETIME(@end)
			AND cost_type IN ("FlexsaveCharges", "FlexsaveNegation")
			AND DATETIME(export_time)>=DATETIME(@start)
			AND DATETIME(export_time)<DATETIME(@end)
			GROUP BY usage_date, payer_id
	`)
}

func (d *AwsStandaloneService) buildStandaloneSavingsRecurringFeeQuery(params BigQueryRequestParams) string {
	return queryReplacer(params).Replace(`
		SELECT
			SUM(cost) as cost,TIMESTAMP(DATE_TRUNC(usage_date_time, DAY)) AS usage_date, label.value as payer_id
		FROM
			{env}.{customer_dataset}.{customer_table}, UNNEST(system_labels) AS label
		WHERE (label.value IN ({account_ids})
			 AND label.key='aws/payer_account_id')
		AND
			DATE(usage_start_time) >= DATETIME(@start)
			AND DATE(usage_start_time) < DATETIME(@end)
			AND cost_type = "FlexsaveRecurringFee"
			AND DATETIME(export_time)>=DATETIME(@start)
			AND DATETIME(export_time)<DATETIME(@end)
			GROUP BY usage_date, payer_id
	`)
}

func (d *AwsStandaloneService) checkBillingTableExists(ctx context.Context, customerID string) error {
	dataset := d.bigQueryClient.Dataset(getCustomerDataset(customerID))

	_, err := d.bqmh.GetTableMetadata(ctx, dataset, getCustomerTable(customerID))
	if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
		return ErrNoTable
	} else {
		return err
	}
}

func queryReplacer(params BigQueryRequestParams) *strings.Replacer {
	return strings.NewReplacer(
		"{env}", getEnvironment(),
		"{customer_dataset}", getCustomerDataset(params.CustomerID),
		"{customer_table}", getCustomerTable(params.CustomerID),
		"{account_ids}", makeAccountIDsArrayForQuery(params.AccountIDs),
	)
}

func getEnvironment() string {
	if c.Production {
		return prod
	}

	return dev
}

func getCustomerDataset(customerID string) string {
	return "aws_billing_" + customerID
}

func getCustomerTable(customerID string) string {
	return "doitintl_billing_export_v1_" + customerID
}

func makeAccountIDsArrayForQuery(accountIDs []string) string {
	return "'" + strings.Join(accountIDs, "','") + "'"
}

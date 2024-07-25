package cspreport

import (
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func GetCSPTableMetadataQuery(onePartition bool, data *domain.CSPMetadataQueryData) string {
	joinQuery := make([]string, 0)
	joinQuery = append(joinQuery, data.EnchancedBillingDataQuery+`
SELECT
	B.*{fields_replace},
	MD.classification,
	MD.primary_domain,
	MD.territory,
	MD.payee_country,
	MD.payer_country,
	MD.field_sales_representative,
	MD.strategic_account_manager,
	MD.technical_account_manager,
	MD.customer_success_manager
FROM
	{billing_data} AS B
LEFT JOIN
	{metadata} AS MD
ON
	B.{bind_id} = MD.{metadata_bind_id}
WHERE
	{partition_filter}
`)

	var fieldsReplace string

	switch data.Cloud {
	// AWS table does not have margin fields in `report`, create it by replacing the original `report` field.
	case common.Assets.AmazonWebServices:
		fieldsReplace = `
			REPLACE(
				ARRAY(SELECT AS STRUCT
					cost,
					usage,
					savings,
					credit,
					CAST(NULL AS FLOAT64) AS margin,
					ext_metric
				FROM UNNEST(B.report)) AS report
			)`
	case common.Assets.GoogleCloud:
		// ignore price field, not used in CSP table at this time
		fieldsReplace = ` EXCEPT(price)`
	}

	selectFrom := data.BillingDataTableFullName
	if data.EnchancedBillingDataSelectFrom != "" {
		selectFrom = data.EnchancedBillingDataSelectFrom
	}

	var partitionFilter string

	if onePartition {
		partitionFilter = `DATE(export_time) = DATE(@partition)`
	} else {
		partitionFilter = `DATE(export_time) >= DATE("2018-01-01")`
	}

	replacer := strings.NewReplacer(
		"{billing_data}", selectFrom,
		"{metadata}", data.MetadataTableFullName,
		"{bind_id}", data.BindIDField,
		"{metadata_bind_id}", data.MetadataBindIDField,
		"{fields_replace}", fieldsReplace,
		"{partition_filter}", partitionFilter,
	)
	query := replacer.Replace(strings.Join(joinQuery, " "))

	return query
}

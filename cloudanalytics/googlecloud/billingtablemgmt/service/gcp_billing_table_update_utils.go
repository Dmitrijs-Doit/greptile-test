package googlecloud

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"golang.org/x/exp/slices"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/services/tablemanagement"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDomain "github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	plpsDomain "github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/pricing"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	nullPLPSPercent = "CAST(NULL AS FLOAT64) AS plps_doit_percent"

	comma string = ", "
)

func getReportDataUDF(isCSP bool) string {
	if isCSP {
		if common.Production {
			return fmt.Sprintf("`%s.%s.UDF_REPORT_DATA_CSP_V1`", domain.BillingProjectProd, domain.BillingDataset)
		}

		return fmt.Sprintf("`%s.%s.UDF_REPORT_DATA_CSP_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
	}

	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_REPORT_DATA_V1`", domain.BillingProjectProd, domain.BillingDataset)
	}

	return fmt.Sprintf("`%s.%s.UDF_REPORT_DATA_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
}

func getCreditsConversionUDF() string {
	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_CONVERT_DIRECT_CREDITS_USD_V1`", domain.BillingProjectProd, domain.BillingDataset)
	}

	return fmt.Sprintf("`%s.%s.UDF_CONVERT_DIRECT_CREDITS_USD_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
}

func getFilterCreditsUDF() string {
	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_FILTER_CREDITS_V1`", domain.BillingProjectProd, domain.BillingDataset)
	}

	return fmt.Sprintf("`%s.%s.UDF_FILTER_CREDITS_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
}

func getEnrichSystemLabelsUDF(isCSP bool) string {
	udfArgs := strings.Join([]string{
		"system_labels", "project.id", "service_id", "sku_description", "cost_type", "is_preemptible", strconv.FormatBool(isCSP)}, comma,
	)
	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_ENRICH_SYSTEM_LABELS_V1`(%s)", domain.BillingProjectProd, domain.BillingDataset, udfArgs)
	}

	return fmt.Sprintf("`%s.%s.UDF_ENRICH_SYSTEM_LABELS_V1BETA`(%s)", domain.BillingProjectDev, domain.BillingDataset, udfArgs)
}

func getExcludeDiscountUDF() string {
	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_CUSTOM_EXCLUDE_DISCOUNT_V1`", domain.BillingProjectProd, domain.BillingDataset)
	}

	return fmt.Sprintf("`%s.%s.UDF_CUSTOM_EXCLUDE_DISCOUNT_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
}

func getEnrichProjectUDF() string {
	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_ENRICH_PROJECT_V1`", domain.BillingProjectProd, domain.BillingDataset)
	}

	return fmt.Sprintf("`%s.%s.UDF_ENRICH_PROJECT_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
}

func replaceHyphenWithUnderscore(s string) string {
	return strings.Replace(s, "-", "_", -1)
}

func getDirectAccountDatasetAndTable(billingAccountID string) string {
	ba := replaceHyphenWithUnderscore(billingAccountID)

	if common.Production {
		return fmt.Sprintf("`%s.gcp_billing_%s.gcp_billing_export_resource_v1_%s`", domain.BillingProjectProd, ba, ba)
	}

	return fmt.Sprintf("`%s.gcp_billing_%s.gcp_billing_export_resource_v1_%s`", domain.BillingProjectDev, ba, ba)
}

func getStandaloneSourceTable(billingAccountID string) string {
	ba := replaceHyphenWithUnderscore(billingAccountID)

	if common.Production {
		return fmt.Sprintf("`%s.gcp_billing_standalone.gcp_billing_export_resource_v1_%s`",
			domain.BillingStandaloneProjectProd, ba)
	}

	return fmt.Sprintf("`%s.gcp_billing_standalone.gcp_billing_export_resource_v1_%s`",
		domain.BillingStandaloneProjectProd, ba)
}

func getOriginalCreditsUDF() string {
	if common.Production {
		return fmt.Sprintf("`%s.%s.UDF_FILTER_CREDITS_RESELLER_V1`", domain.BillingProjectProd, domain.BillingDataset)
	}

	return fmt.Sprintf("`%s.%s.UDF_FILTER_CREDITS_RESELLER_V1BETA`", domain.BillingProjectDev, domain.BillingDataset)
}

func getFlexsaveBillingTable(production bool) string {
	if production {
		return fmt.Sprintf("`%s.%s.%s`", consts.CustomBillingProd, domain.FlexsaveDataset, domain.FlexsaveBillingTable)
	}

	return fmt.Sprintf("`%s.%s.%s`", consts.CustomBillingDev, domain.FlexsaveDataset, domain.FlexsaveBillingTable)
}

// getFlexsaveBillingQuery allows us to handle different schemas of the custom billing tables.
func getFlexsaveBillingQuery() string {
	return fmt.Sprintf(`SELECT
				* EXCEPT(gcp_metrics, customer_type),
				%s,
				%s,
				%s,
				%s,
				%s,
				%s,
				%s,
				customer_type,
				gcp_metrics,
				{plps_field}
			FROM %s`,
		queryDomain.NullTags,
		queryDomain.NullPrice,
		queryDomain.NullCostAtList,
		queryDomain.NullTransactionType,
		queryDomain.NullSellerName,
		queryDomain.NullSubscription,
		queryDomain.NullResource,
		getFlexsaveBillingTable(common.Production),
	)
}

func getDefaultBillingQuery(contractConditionalClause string) string {
	query := `
SELECT
	*
	REPLACE(IF(sku.id = "{plps_sku}", ARRAY_CONCAT(system_labels, [STRUCT("plps-source", "google")]), system_labels) AS system_labels),
	"resold" AS customer_type,
	{null_gcp_metrics},
	CAST(NULL AS FLOAT64) AS plps_doit_percent,
FROM
	{raw_billing_table}`

	// Indicates whether there are PLPS contracts for the customer
	if contractConditionalClause != "" {
		query += `

UNION ALL

SELECT
	*
	REPLACE(IF(sku.id = "{plps_sku}", ARRAY_CONCAT(system_labels, [STRUCT("plps-source", "doit")]), system_labels) AS system_labels),
	"resold" AS customer_type,
	{null_gcp_metrics},
	{contract_conditional_clause}
FROM
	{raw_billing_table}
WHERE
	sku.id = "{plps_sku}"`
	}

	replacements := strings.NewReplacer(
		"{plps_sku}", plpsDomain.PLPSSkuID,
		"{null_gcp_metrics}", queryDomain.NullGCPMetrics,
		"{raw_billing_table}", domain.GetRawBillingTableName(true),
		"{contract_conditional_clause}", contractConditionalClause,
	)

	return replacements.Replace(query)
}

func getStandaloneBillingQuery(billingAccountID string) string {
	return fmt.Sprintf(`SELECT
	billing_account_id,
	service,
	sku,
	usage_start_time,
	usage_end_time,
	project,
	labels,
	system_labels,
	location,
	export_time,
	cost,
	currency,
	currency_conversion_rate,
	usage,
	credits,
	invoice,
	cost_type,
	adjustment_info,
	tags,
    price,
	cost_at_list,
	transaction_type,
	seller_name,
	subscription,
	resource,
	"standalone" AS customer_type,
	%s,
    %s
FROM
	%s`,
		queryDomain.NullGCPMetrics,
		nullPLPSPercent,
		getStandaloneSourceTable(billingAccountID),
	)
}

func getDirectBillingQuery(billingAccountID string) string {
	return fmt.Sprintf(`SELECT *
	EXCEPT(resource)
	REPLACE(
		IF(currency != "USD", cost/currency_conversion_rate, cost) AS cost,
		"USD" AS currency,
		1.0 AS currency_conversion_rate,
		{credits_conversion_udf}(credits, currency_conversion_rate) AS credits
	),
	resource,
	"direct" AS customer_type,
	%s
FROM
	%s`,
		queryDomain.NullGCPMetrics,
		getDirectAccountDatasetAndTable(billingAccountID),
	)
}

func (s *BillingTableManagementService) updateBillingAccountTable(ctx context.Context, billingAccountID string, data *tableMgmtDomain.BigQueryTableUpdateRequest) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	// Get asset from Firestore
	assetTypes := []string{
		common.Assets.GoogleCloud,
		common.Assets.GoogleCloudReseller,
		common.Assets.GoogleCloudStandalone,
	}

	// Direct accounts are not relevant for CSP operations
	if !data.CSP {
		assetTypes = append(assetTypes, common.Assets.GoogleCloudDirect)
	}

	docSnaps, err := fs.Collection("assets").
		Where("properties.billingAccountId", "==", billingAccountID).
		Where("type", "in", assetTypes).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	} else if len(docSnaps) > 1 {
		return fmt.Errorf("conflict: fetched more than one document from assets with properties.billingAccountId == '%s'", billingAccountID)
	} else if len(docSnaps) == 0 {
		return fmt.Errorf("not found: could not find assets with properties.billingAccountId == '%s'", billingAccountID)
	}

	// We should match only 1 doc with given billingAccountID
	docSnap := docSnaps[0]

	var asset googlecloud.Asset
	if err := docSnap.DataTo(&asset); err != nil {
		return err
	}

	data.IsStandalone = asset.AssetType == common.Assets.GoogleCloudStandalone

	data.DatasetMetadata = bigquery.DatasetMetadata{
		Name:        billingAccountID,
		Description: fmt.Sprintf("Billing Export for %s", billingAccountID),
	}

	if !data.CSP {
		if tableExists, err := tablemanagement.CheckDestinationTable(ctx, bq, data); err != nil {
			return err
		} else if !tableExists {
			// If table does not exist and a specific from date was not set, then update all partitions.
			data.AllPartitions = data.FromDate == ""
		}
	}

	// define asset related properties
	assetType := asset.AssetType
	customerRef := asset.Customer
	entityRef := asset.Entity
	docID := fmt.Sprintf("%s-%s", assetType, billingAccountID)
	assetRef := fs.Collection("assets").Doc(docID)

	// support direct accounts
	isDirectAccount := asset.AssetType == common.Assets.GoogleCloudDirect

	// get customer from Firestore
	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		return err
	}

	var pricebooks []*pricing.CustomerPricebookGoogleCloud
	if !isDirectAccount && !data.IsStandalone {
		pricebooks, err = getGoogleCloudPricebooks(ctx, customerRef, entityRef, assetRef)
		if err != nil {
			return err
		}
	}

	var pricebookTable *string

	if len(pricebooks) > 0 {
		tables := make([]string, 0)

		for _, pricebook := range pricebooks {
			t := strings.NewReplacer(
				"{pricebook_table}",
				pricebook.Table,
				"{start_date}",
				pricebook.StartDate.Format(times.YearMonthDayLayout),
				"{end_date}",
				pricebook.EndDate.Format(times.YearMonthDayLayout)).
				Replace(`SELECT DISTINCT(sku_id) AS sku_id, (1 - 0.01 * custom_discount) AS discount, custom_usage_pricing_unit AS unit_price, DATE("{start_date}") AS start_date, DATE("{end_date}") AS end_date FROM {pricebook_table}`)
			tables = append(tables, t)
		}

		if len(tables) > 0 {
			u := strings.Join(tables, "\n\tUNION ALL\n\t\t")
			pricebookTable = &u
		}
	}

	plpsContracts, err := s.contractDAL.GetContractsByType(ctx, customerRef, contractDomain.ContractTypeGoogleCloudPLPS)
	if err != nil {
		return err
	}

	plpsClause := getPLPSClause(plpsContracts)

	var projectField string
	if customer.SecurityMode != nil && *customer.SecurityMode == common.CustomerSecurityModeRestricted {
		projectField = `IF(REGEXP_CONTAINS(T.project.id, r"^doitintl-fs-[0-9a-z]{6,}$"), "doitintl-fs", T.project.number)`
	} else {
		projectField = `IF(REGEXP_CONTAINS(T.project.id, r"^doitintl-fs-[0-9a-z]{6,}$"), "doitintl-fs", T.project.id)`
	}

	replacer := strings.NewReplacer(
		"{billing_account_id}", billingAccountID,
		"{project_field}", projectField,
		"{discounts_table}", GetDiscountsTableName(),
		"{gcp_billing_skus}", domain.GetBillingSkusTableName(),
		"{promotional_credits}", domain.GetPromotionalCreditsTableName(),
		"{iam_resources}", GetIAMResourcesTableName(),
		"{udf_enrich_system_labels}", getEnrichSystemLabelsUDF(data.CSP),
		"{udf_filter_credits}", getFilterCreditsUDF(),
		"{udf_reseller_credits}", getOriginalCreditsUDF(),
		"{udf_exclude_discount}", getExcludeDiscountUDF(),
		"{udf_enrich_project}", getEnrichProjectUDF(),
		"{credits_conversion_udf}", getCreditsConversionUDF(),
	)

	// validate from date argument is in correct format
	if data.FromDate != "" {
		if _, err := time.Parse(times.YearMonthDayLayout, data.FromDate); err != nil {
			return fmt.Errorf("failed parsing from date with error: %s", err)
		}
	}

	query := replacer.Replace(getQuery(billingAccountID, pricebookTable, data, isDirectAccount, plpsClause))

	data.QueryParameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerRef.ID},
		{Name: "billing_account_id", Value: billingAccountID},
	}

	if err := service.RunBillingTableUpdateQuery(ctx, bq, query, data); err != nil {
		return err
	}

	if !data.CSP {
		l.Infof("Updating report status for customer %s", customerRef.ID)

		if err := s.reportStatusService.UpdateReportStatus(ctx, customerRef.ID, common.ReportStatus{
			Status: map[string]common.StatusInfo{
				string(common.GoogleCloudReportStatus): {
					LastUpdate: time.Now(),
				},
			},
		}); err != nil {
			l.Error(err)
		}
	}

	return nil
}

func getQuery(
	billingAccountID string,
	pricebookTable *string,
	data *tableMgmtDomain.BigQueryTableUpdateRequest,
	isDirectAccount bool,
	contractConditionalClause string,
) string {
	withClauses := make([]string, 0)

	promotionalCreditsWithClause := `promotional_credits AS (
	SELECT
		STRUCT(service_id, credit_id, credit_name)
	FROM
		{promotional_credits}
	WHERE
		billing_account_id IS NULL OR billing_account_id = @billing_account_id
)`
	withClauses = append(withClauses, promotionalCreditsWithClause)

	iamResourcesWithClause := `iam_resources AS (
	SELECT
		STRUCT(id, name)
	FROM
		{iam_resources}
	WHERE
		customer = @customer_id
)`
	withClauses = append(withClauses, iamResourcesWithClause)

	skuMetadataWithClause := `skus_metadata AS (
	SELECT
		service_id,
		sku_id,
		LOGICAL_OR(IFNULL(T.is_marketplace, FALSE) OR IFNULL(S.is_marketplace, FALSE)) AS is_marketplace,
		LOGICAL_OR(IFNULL(T.is_preemptible, FALSE) OR IFNULL(S.is_preemptible, FALSE)) AS is_preemptible,
		LOGICAL_OR(IFNULL(T.is_premium_image, FALSE) OR IFNULL(S.is_premium_image, FALSE)) AS is_premium_image
	FROM
		(
			SELECT
			service.id AS service_id,
			sku.id AS sku_id,
			properties.isMarketplace AS is_marketplace,
			properties.isPreemptible AS is_preemptible,
			properties.isPremiumImage AS is_premium_image
			FROM {gcp_billing_skus}
			WHERE properties.isMarketplace OR properties.isPreemptible OR properties.isPremiumImage
		) AS T
		FULL OUTER JOIN
		(
			SELECT *
			FROM gcp_billing_skus_metadata_v1
			WHERE is_marketplace OR is_preemptible OR is_premium_image
		) AS S
	USING (service_id, sku_id)
	GROUP BY service_id, sku_id
)`
	withClauses = append(withClauses, skuMetadataWithClause)

	var (
		rawBillingDataTable   string
		isDefaultBillingQuery bool
		shouldFilterCredits   bool
	)

	switch {
	case isDirectAccount:
		rawBillingDataTable = getDirectBillingQuery(billingAccountID)
	case data.IsStandalone:
		rawBillingDataTable = getStandaloneBillingQuery(billingAccountID)
		isDefaultBillingQuery = true
	default:
		rawBillingDataTable = getDefaultBillingQuery(contractConditionalClause)
		isDefaultBillingQuery = true
		shouldFilterCredits = true
	}

	withClauses = append(withClauses, fmt.Sprintf(`raw_billing_data_table AS (
	%s
)`, rawBillingDataTable))

	var r []string

	if data.CSP {
		r = append(r, "{reseller_margin_credits}", "{udf_reseller_credits}(credits) AS margin_credits")
	} else {
		r = append(r, "{reseller_margin_credits}", "")
	}

	if shouldFilterCredits {
		r = append(r, "{replace_credits_field}", "{udf_filter_credits}(billing_account_id, service_id, usage_date_time, credits, promotional_credits) AS credits")
	} else {
		r = append(r, "{replace_credits_field}", "ARRAY(SELECT STRUCT(COALESCE(c.full_name, c.name) AS name, c.amount AS amount, c.full_name AS full_name, c.id AS id, c.type AS type) FROM UNNEST(credits) AS c) AS credits")
	}

	// Query partition filter:
	// 1. fromDate: filter for all partitions >= YYYY-MM-DD, if truncating then must pick specific
	// 		partition using the @partition query parameter
	// 2. allPartitions: all partitions from Jan 1st 2018 (start of data)
	// 3. default: specific partition set by the @partition query parameter (today or yesterday)
	var partitionFilter string

	if data.AllPartitions {
		partitionFilter = `AND DATE(export_time) >= DATE("2018-01-01")`
	} else if data.WriteDisposition == bigquery.WriteAppend && data.FromDate != "" {
		switch data.FromDateNumPartitions {
		case 0:
			partitionFilter = fmt.Sprintf(`AND DATE(export_time) >= DATE("%s")`, data.FromDate)
		case 1:
			partitionFilter = fmt.Sprintf(`AND DATE(export_time) = DATE("%s")`, data.FromDate)
		default:
			partitionFilter = fmt.Sprintf(`AND DATE(export_time) >= DATE("%s") AND DATE(export_time) < DATE_ADD(DATE("%s"), INTERVAL %d DAY)`,
				data.FromDate, data.FromDate, data.FromDateNumPartitions)
		}
	} else {
		partitionFilter = "AND DATE(export_time) = DATE(@partition)"
	}

	r = append(r, "{partition_filter}", partitionFilter)

	joinedBillingDataTable := fmt.Sprintf(`raw_enhanced_billing_data_table AS (
			SELECT
				*
				REPLACE(
					STRUCT(project.id AS id, project.number AS number, project.name AS name, project.labels as labels, project.ancestry_numbers AS ancestry_numbers) AS project
				)
			FROM
				raw_billing_data_table
			UNION ALL
			%s
		)`, getFlexsaveBillingQuery())

	plpsDoitValue := "plps_doit_percent"

	if isDefaultBillingQuery {
		joinedBillingDataTable = strings.Replace(joinedBillingDataTable, "{plps_field}", nullPLPSPercent, 1)
	} else {
		joinedBillingDataTable = strings.Replace(joinedBillingDataTable, "{plps_field}", "", 1)
		plpsDoitValue = "null"
	}

	withClauses = append(withClauses, joinedBillingDataTable)

	rawDataWithClause := strings.NewReplacer(r...).Replace(`raw_data AS (
	SELECT
		*
		EXCEPT(promotional_credits, iam_resources)
		REPLACE (
			{udf_enrich_project}(project, iam_resources) AS project,
			{udf_enrich_system_labels} AS system_labels,
			{replace_credits_field}
		),
		{reseller_margin_credits}
	FROM
		(SELECT
			T.billing_account_id AS billing_account_id,
			{project_field} AS project_id,
			T.service.description AS service_description,
			T.service.id AS service_id,
			T.sku.description AS sku_description,
			T.sku.id AS sku_id,
			DATETIME(T.usage_start_time, "America/Los_Angeles") AS usage_date_time,
			T.* EXCEPT(billing_account_id, service, sku, resource),
			T.resource.name AS resource_id,
			T.resource.global_name as resource_global_id,
			IFNULL(S.is_marketplace, FALSE) AS is_marketplace,
			IFNULL(S.is_preemptible, FALSE) AS is_preemptible,
			IFNULL(S.is_premium_image, FALSE) AS is_premium_image,
			{udf_exclude_discount}(T.service.id, T.sku.id, T.sku.description, DATE(T.usage_start_time, "America/Los_Angeles")) AS exclude_discount,
			(ARRAY(SELECT * FROM promotional_credits)) AS promotional_credits,
			(ARRAY(SELECT * FROM iam_resources)) AS iam_resources,
			(SELECT value FROM UNNEST(T.labels) WHERE key = "goog-k8s-cluster-name" LIMIT 1) AS kubernetes_cluster_name,
			(SELECT value FROM UNNEST(T.labels) WHERE key = "k8s-namespace" LIMIT 1) AS kubernetes_namespace
		FROM raw_enhanced_billing_data_table AS T
		LEFT JOIN skus_metadata AS S
		ON T.service.id = S.service_id AND T.sku.id = S.sku_id
		WHERE
			billing_account_id = @billing_account_id
			{partition_filter}
		)
)`)
	withClauses = append(withClauses, rawDataWithClause)

	discountsDataWithClause := `discounts_data AS (
		SELECT
			*
			EXCEPT(contract, contract_start_date, contract_end_date, next_start_date, periods)
			REPLACE(IFNULL(LEAST(next_start_date, end_date), next_start_date) AS end_date)
		FROM (
			SELECT
				contract,
				contract_start_date,
				contract_end_date,
				IFNULL(LAG(contract_start_date) OVER (ORDER BY contract_start_date DESC), contract_end_date) AS next_start_date,
				ARRAY_AGG(STRUCT(
					start_date,
					end_date,
					discount,
					rebase_modifier,
					allow_preemptible,
					CAST(is_commitment as STRING) AS is_commitment
				)) AS periods,
			FROM
				{discounts_table}
			WHERE
				billing_account_id = @billing_account_id
				AND _PARTITIONDATE >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)
				AND _PARTITIONDATE = (
					SELECT MAX(_PARTITIONDATE)
					FROM {discounts_table}
					WHERE _PARTITIONDATE >= DATE_SUB(CURRENT_DATE(), INTERVAL 7 DAY)
				)
				AND is_active
			GROUP BY
				contract, contract_start_date, contract_end_date
		) LEFT JOIN UNNEST(periods)
		WHERE
			start_date < next_start_date OR next_start_date IS NULL
		ORDER BY
			start_date DESC
	)`
	withClauses = append(withClauses, discountsDataWithClause)

	if pricebookTable != nil {
		pricebookDataWithClause := fmt.Sprintf(`pricebook_data AS (
	SELECT * FROM (
		%s
	)
)`, *pricebookTable)

		dataWithClause := `data AS (
	SELECT
		T.*,
		STRUCT(CPL.discount AS discount, CPL.unit_price AS unit_price) AS price_book
	FROM raw_data AS T
	LEFT JOIN pricebook_data AS CPL
	ON T.sku_id = CPL.sku_id
	AND DATE(T.usage_date_time) >= CPL.start_date
	AND DATE(T.usage_date_time) < CPL.end_date
)`
		withClauses = append(withClauses, pricebookDataWithClause, dataWithClause)
	} else {
		dataWithClause := `data AS (
	SELECT
		T.*,
		STRUCT(CAST(NULL AS FLOAT64) AS discount, CAST(NULL AS FLOAT64) AS unit_price) AS price_book
	FROM raw_data AS T
)`
		withClauses = append(withClauses, dataWithClause)
	}

	replacer := strings.NewReplacer(
		"{with_clauses}", strings.Join(withClauses, ",\n"),
		"{report_udf}", getReportDataUDF(data.CSP),
		"{plps_google_percent}", strconv.FormatFloat(plpsDomain.GooglePLPSChargePercentage, 'f', -1, 64),
		"{plps_doit_percent}", plpsDoitValue,
	)
	query := replacer.Replace(`-- {billing_account_id}
WITH
{with_clauses}
SELECT
	*,
	{report_udf}(is_marketplace, is_preemptible, is_premium_image, exclude_discount, cost, usage.amount_in_pricing_units, credits, {margin_credits} price_book, discount, gcp_metrics, {plps_doit_percent}, {plps_google_percent}) AS report
FROM (
	SELECT
		T.*,
		IFNULL(ARRAY(
			SELECT AS STRUCT
				D.discount AS value,
				D.rebase_modifier AS rebase_modifier,
				D.allow_preemptible AS allow_preemptible,
				D.is_commitment AS is_commitment
			FROM
				discounts_data AS D
			WHERE
				DATE(T.usage_date_time) >= D.start_date
				AND (D.end_date IS NULL OR DATE(T.usage_date_time) < D.end_date)
			)[SAFE_OFFSET(0)],
			STRUCT(NULL AS value, NULL AS rebase_modifier, NULL AS allow_preemptible, NULL as is_commitment)
		) AS discount
	FROM data AS T
)`)

	if data.CSP {
		query = strings.Replace(query, "{margin_credits}", "margin_credits,", 1)
	} else {
		query = strings.Replace(query, "{margin_credits}", "", 1)
	}

	return query
}

func getGoogleCloudPricebooks(ctx context.Context, customerRef, entityRef, assetRef *firestore.DocumentRef) ([]*pricing.CustomerPricebookGoogleCloud, error) {
	pricebooks := make([]*pricing.CustomerPricebookGoogleCloud, 0)
	if customerRef == nil || entityRef == nil {
		return pricebooks, nil
	}

	docSnaps, err := customerRef.Collection("customerPricebooks").
		Where("type", "==", common.Assets.GoogleCloud).
		Where("entity", "==", entityRef).
		OrderBy("endDate", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var pricebook pricing.CustomerPricebookGoogleCloud
		if err := docSnap.DataTo(&pricebook); err != nil {
			return nil, err
		}

		if pricebook.Assets != nil && len(pricebook.Assets) > 0 && doitFirestore.FindIndex(pricebook.Assets, assetRef) == -1 {
			continue
		}

		pricebooks = append(pricebooks, &pricebook)
	}

	return pricebooks, nil
}

func getPLPSClause(contracts []common.Contract) string {
	var contractConditionalClause string

	contracts = slices.DeleteFunc(contracts, func(c common.Contract) bool {
		return !c.Active
	})

	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].StartDate.After(contracts[j].StartDate)
	})

	if len(contracts) > 0 {
		contractConditionalClause = "CASE\n"

		for i, contract := range contracts {
			var contractEndDate string

			if !contract.EndDate.IsZero() {
				contractEndDate = contract.EndDate.Format(times.YearMonthDayLayout)
			} else if i < len(contracts)-1 {
				// Use the start date of the next contract
				contractEndDate = contracts[i+1].StartDate.Add(-time.Second).Format(times.YearMonthDayLayout)
			} else {
				// If it's the last contract, use the current time
				contractEndDate = times.CurrentDayUTC().Format(times.YearMonthDayLayout)
			}

			whenClause := fmt.Sprintf("\t\tWHEN DATE(usage_start_time, 'America/Los_Angeles') BETWEEN DATE('%s') AND DATE('%s') THEN %f\n",
				contract.StartDate.Format(times.YearMonthDayLayout),
				contractEndDate,
				contract.PLPSPercent,
			)
			contractConditionalClause += whenClause
		}

		contractConditionalClause += "\tELSE 0.0 END AS plps_doit_percent"
	}

	return contractConditionalClause
}

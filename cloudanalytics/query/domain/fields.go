package domain

const (
	FieldAdjustmentInfo          string = "adjustment_info"
	NullAdjustmentInfo           string = "CAST(NULL AS STRUCT<id STRING, description STRING, mode STRING, type STRING>)"
	FieldCustomer                string = "customer"
	FieldCloudProvider           string = "cloud_provider"
	FieldBillingAccountID        string = "billing_account_id"
	FieldProject                 string = "project"
	FieldProjectID               string = "project_id"
	FieldProjectNumber           string = "project.number AS project_number"
	NullProjectNumber            string = "CAST(NULL AS STRING) AS project_number"
	FieldProjectName             string = "project.name AS project_name"
	NullProjectName              string = "CAST(NULL AS STRING) AS project_name"
	FieldServiceDescription      string = "service_description"
	FieldServiceID               string = "service_id"
	FieldSKUDescription          string = "sku_description"
	FieldSKUID                   string = "sku_id"
	NullOperation                string = "CAST(NULL AS STRING) AS operation"
	FieldOperation               string = "operation"
	NullResource                 string = "STRUCT(CAST(NULL AS STRING) AS name, CAST(NULL AS STRING) as global_name) AS resource"
	NullResourceID               string = "CAST(NULL AS STRING) AS resource_id"
	NullResourceGlobalID         string = "CAST(NULL AS STRING) AS resource_global_id"
	FieldResourceID              string = "resource_id"
	FieldResourceGlobalID        string = "resource_global_id"
	NullCommitment               string = "NULL AS is_commitment"
	FieldCommitment              string = "is_commitment"
	FieldGCPProject              string = "STRUCT(project.ancestry_names, project.labels) AS project"
	FieldAWSProject              string = "STRUCT(NULL AS ancestry_names, project.labels) AS project"
	NullProject                  string = "NULL AS project"
	FieldUsageDateTime           string = "usage_date_time"
	FieldUsageStartTime          string = "usage_start_time"
	FieldUsageEndTime            string = "usage_end_time"
	FieldLabels                  string = "labels"
	FieldSystemLabels            string = "system_labels"
	NullSystemLabels             string = "CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>) AS system_labels"
	FieldLocation                string = "location"
	FieldExportTime              string = "export_time"
	FieldExportTimeUsageDateTime string = "CAST(usage_date_time AS TIMESTAMP) AS export_time"
	FieldPricingUsage            string = "STRUCT(usage.pricing_unit AS pricing_unit) AS usage"
	NullPricingUsage             string = "CAST(NULL AS STRUCT<pricing_unit STRING>) AS usage"
	FieldUsage                   string = "usage"
	FieldCostType                string = "cost_type"
	FieldPricingUnit             string = "pricing_unit"
	FieldPrimaryDomain           string = "primary_domain"
	FieldClassification          string = "classification"
	FieldTerritory               string = "territory"
	FieldPayerCountry            string = "payer_country"
	FieldPayeeCountry            string = "payee_country"
	FieldFSR                     string = "field_sales_representative"
	FieldSAM                     string = "strategic_account_manager"
	FieldTAM                     string = "technical_account_manager"
	FieldCSM                     string = "customer_success_manager"
	FieldCurrency                string = "currency"
	FieldCurrencyRate            string = "currency_conversion_rate"
	FieldCredit                  string = "credit"
	FieldCredits                 string = "credits"
	FieldCost                    string = "cost"
	FieldDiscount                string = "discount"
	NullDiscount                 string = "CAST(NULL AS STRUCT<value FLOAT64, rebase_modifier FLOAT64, allow_preemptible BOOLEAN, is_commitment STRING>) AS discount"
	FieldInvoice                 string = "invoice"
	NullInvoice                  string = "CAST(NULL AS STRUCT<month STRING>) AS invoice"
	FieldIsMarketplace           string = "is_marketplace"
	NullIsMarketplace            string = "CAST(NULL AS BOOLEAN) is_marketplace"
	FalseIsMarketplace           string = "false AS is_marketplace"
	FieldIsPreemptible           string = "is_preemptible"
	NullIsPreemptible            string = "CAST(NULL AS BOOLEAN) AS is_preemptible"
	FieldIsPremiumImage          string = "is_premium_image"
	NullIsPremiumImage           string = "CAST(NULL AS BOOLEAN) AS is_premium_image"
	FieldExcludeDiscount         string = "exclude_discount"
	NullExcludeDiscount          string = "CAST(NULL AS BOOLEAN) AS exclude_discount"
	FieldWeekDay                 string = "week_day"
	FieldCustomerType            string = "customer_type"
	NullCustomerType             string = "CAST(NULL AS STRING) AS customer_type"
	FieldTags                    string = "tags"
	FieldReport                  string = "report"
	NullTags                     string = "CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING, inherited BOOLEAN, namespace STRING>>) AS tags"
	NullMarginCredits            string = "CAST([] AS ARRAY<STRUCT<name STRING, amount FLOAT64, full_name STRING, id STRING, type STRING>>) AS margin_credits"
	NullPrice                    string = "CAST(NULL AS STRUCT<effective_price NUMERIC, tier_start_amount NUMERIC, unit STRING, pricing_unit_quantity NUMERIC>) AS price"
	NullPriceBook                string = "CAST(NULL AS STRUCT<discount FLOAT64, unit_price FLOAT64>) AS price_book"
	NullCostAtList               string = "CAST(NULL AS FLOAT64) AS cost_at_list"
	NullClassification           string = "CAST(NULL AS STRING) AS classification"
	NullTransactionType          string = "CAST(NULL AS STRING) AS transaction_type"
	NullSellerName               string = "CAST(NULL AS STRING) AS seller_name"
	NullSubscription             string = "CAST(NULL AS STRUCT<instance_id STRING>) AS subscription"
	NullPrimaryDomain            string = "CAST(NULL AS STRING) AS primary_domain"
	NullTerritory                string = "CAST(NULL AS STRING) AS territory"
	NullPayeeCountry             string = "CAST(NULL AS STRING) AS payee_country"
	NullPayerCountry             string = "CAST(NULL AS STRING) AS payer_country"
	NullFieldSalesRepresentative string = "CAST(NULL AS STRING) AS sales_representative"
	NullStrategicAccountManager  string = "CAST(NULL AS STRING) AS strategic_account_manager"
	NullTechnicalAccountManager  string = "CAST(NULL AS STRING) AS technical_account_manager"
	NullCustomerSuccessManager   string = "CAST(NULL AS STRING) AS customer_success_manager"
	FieldKubernetesClusterName   string = "kubernetes_cluster_name"
	FieldKubernetesNamespace     string = "kubernetes_namespace"
	NullKubernetesClusterName    string = "CAST(NULL AS STRING) AS kubernetes_cluster_name"
	NullKubernetesNamespace      string = "CAST(NULL AS STRING) AS kubernetes_namespace"
	FieldBillingReport           string = "report"
	FieldFeature                 string = "feature"
	FieldFeatureType             string = "feature_type"
	FieldFeaturePlaceholder      string = `["Anomalies Count", "Anomalies Cost"] AS feature`
	FieldProjectNumberCF         string = "project_number"
	FieldProjectNameCF           string = "project_name"
	NullGCPMetrics               string = "CAST([] AS ARRAY<STRUCT<key STRING, value FLOAT64, type STRING>>) AS gcp_metrics"

	FieldBillingReportGCP    string = "ARRAY(SELECT AS STRUCT cost, usage, savings, CAST(NULL AS STRING) AS savings_description, credit, ext_metric FROM UNNEST(report)) AS report"
	FieldBillingReportGKE    string = "ARRAY(SELECT AS STRUCT cost, usage, savings FROM UNNEST(report)) AS report"
	FieldBillingReportLegacy string = "ARRAY(SELECT AS STRUCT cost, usage, savings, CAST(NULL AS STRING) AS savings_description, credit, CAST(NULL AS STRUCT<key STRING, value FLOAT64, type STRING>) AS ext_metric FROM UNNEST(report)) AS report"

	FieldBillingReportCFFull string = "[STRUCT( (CASE WHEN feature_type='cost' THEN CAST(value AS FLOAT64) ELSE NULL END) AS cost, (CASE WHEN feature_type='usage' THEN CAST(value AS FLOAT64) ELSE NULL END) AS usage, CAST(NULL AS FLOAT64) AS savings, CAST(NULL AS STRING) AS credit, CAST(NULL AS FLOAT64) AS margin, STRUCT(NULL AS key, NULL AS value, NULL AS type) AS ext_metric)]"
	FieldBillingReportCF     string = "[STRUCT( (CASE WHEN feature_type='cost' THEN CAST(value AS FLOAT64) ELSE NULL END) AS cost, (CASE WHEN feature_type='usage' THEN CAST(value AS FLOAT64) ELSE NULL END) AS usage, CAST(NULL AS FLOAT64) AS savings, CAST(NULL AS FLOAT64) AS margin, CAST(NULL AS STRING) AS credit, STRUCT(NULL AS key, NULL AS value, NULL AS type) AS ext_metric)]"
	FieldBillingReportCFDoit string = "[STRUCT( (CASE WHEN feature_type='cost' THEN CAST(value AS FLOAT64) ELSE NULL END) AS cost, (CASE WHEN feature_type='usage' THEN CAST(value AS FLOAT64) ELSE NULL END) AS usage, CAST(NULL AS FLOAT64) AS savings, CAST(NULL AS STRING) AS savings_description, CAST(NULL AS STRING) AS credit, STRUCT(CAST(NULL AS STRING) AS key, CAST(NULL AS FLOAT64) AS value, CAST(NULL AS STRING) AS type) AS ext_metric)]"
	FieldUsageCFDoit         string = "STRUCT(NULL AS pricing_unit)"
	FieldProjectCFDoit       string = "STRUCT(CAST(NULL AS STRING) AS ancestry_names, [STRUCT(CAST(NULL AS STRING) AS key, CAST(NULL AS STRING) AS value)] AS labels)"

	FieldLogsUser                         string = "user"
	FieldLogsCallerIP                     string = "caller_ip"
	FieldLogsEventName                    string = "event_name"
	FieldLogsQueryPriority                string = "query_priority"
	FieldLogsStatementType                string = "statement_type"
	FieldLogsJobStatus                    string = "job_status"
	FieldReservation                      string = "reservation"
	FieldLogsReportExtendedMetricTemplate string = `STRUCT(CAST(NULL AS FLOAT64) AS cost, CAST(NULL AS FLOAT64) AS usage, CAST(NULL AS FLOAT64) AS savings, CAST(NULL AS STRING) AS savings_description, CAST(NULL AS STRING) AS credit, STRUCT(%s AS key, CAST(%s AS FLOAT64) AS value, "usage" AS type) AS ext_metric)`
	// eks billing
	FieldAWSMetric   string = "aws_metric"
	FieldRowID       string = "row_id"
	FieldDescription string = "description"
	FieldCustomerID  string = "customer_id"
	FieldNulletl     string = "NULL AS etl"

	// DataHub
	FieldDataHubCloudProvider        string = "cloud AS cloud_provider"
	FieldDataHubCostTypeOverride     string = "'regular' AS cost_type"
	FieldDataHubCurrencyOverride     string = "'USD' AS currency"
	FieldDataHubCurrencyRateOverride string = "1.0 AS currency_conversion_rate"
	FieldDataHubPricingUsage         string = "STRUCT(pricing_unit AS pricing_unit) AS usage"
	FieldDataHubProjectNumber        string = "project_number"
	FieldDataHubProjectName          string = "project_name"
	FieldDataHubProject              string = "STRUCT(CAST(NULL AS STRING) AS ancestry_names, project_labels AS labels) AS project"
	FieldDataHubReport               string = `ARRAY(SELECT AS STRUCT
		CASE WHEN metrics.type = 'cost' THEN metrics.value
			ELSE 0.0
			END
		AS cost,
		CASE WHEN metrics.type = 'usage' THEN metrics.value
			ELSE 0.0
			END
		AS usage,
		CASE WHEN metrics.type = 'savings' THEN metrics.value
			ELSE 0.0
			END
		AS savings,
		CAST(NULL AS STRING) AS savings_description,
		CAST(NULL AS STRING) AS credit,
		CASE
		   WHEN metrics.type IN ('cost', 'usage', 'savings') THEN
			   CAST(NULL AS STRUCT <key STRING, value FLOAT64, type STRING>)
		   ELSE
			   STRUCT(metrics.type AS key, metrics.value AS value, "usage" AS type)
		 END AS ext_metric
		) AS report`
)

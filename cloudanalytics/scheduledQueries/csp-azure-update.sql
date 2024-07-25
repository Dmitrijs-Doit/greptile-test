-- -- Create table if not exists
-- CREATE TABLE IF NOT EXISTS `doitintl-cmp-azure-data.azure_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL` (
--   billing_account_id STRING,
--   project_id STRING,
--   service_description STRING,
--   service_id STRING,
--   sku_description STRING,
--   sku_id STRING,
--   usage_date_time DATETIME,
--   usage_start_time TIMESTAMP,
--   usage_end_time TIMESTAMP,
--   project STRUCT<id STRING, name STRING, labels ARRAY<STRUCT<key STRING, value STRING>>, ancestry_numbers STRING, number STRING>,
--   labels ARRAY<STRUCT<key STRING, value STRING>>,
--   system_labels ARRAY<STRUCT<key STRING, value STRING>>,
--   location STRUCT<location STRING, country STRING, region STRING, zone STRING>,
--   export_time TIMESTAMP,
--   cost FLOAT64,
--   currency STRING,
--   currency_conversion_rate FLOAT64,
--   usage STRUCT<amount FLOAT64, unit STRING, amount_in_pricing_units FLOAT64, pricing_unit STRING>,
--   invoice STRUCT<month STRING>,
--   cost_type STRING,
--   report ARRAY<STRUCT<cost FLOAT64, usage FLOAT64, savings FLOAT64, credit STRING, margin FLOAT64, ext_metric STRUCT<key STRING, value FLOAT64, type STRING>>>,
--   resource_id STRING,
--   operation STRING,
--   customer_type STRING,
--   discount STRUCT<is_commitment STRING>,
--   classification STRING,
--   primary_domain STRING,
--   territory STRING,
--   payee_country STRING,
--   payer_country STRING,
--   field_sales_representative STRING,
--   strategic_account_manager STRING,
--   technical_account_manager STRING,
--   customer_success_manager STRING
--   is_marketplace BOOL
-- )
-- PARTITION BY DATE(export_time)
-- CLUSTER BY territory, primary_domain;

BEGIN TRANSACTION;

-- Delete the partitions to be replaced
DELETE `doitintl-cmp-azure-data.azure_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL`
WHERE DATE(export_time) BETWEEN @start_date AND @end_date;

-- Azure billing data
INSERT `doitintl-cmp-azure-data.azure_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL`
SELECT *
FROM (
         WITH
             enhanced_data AS (
                 SELECT
                     T.*
                        EXCEPT(etl, azure_metric, row_id, description, customer_id, tenant_id, provider),
                        STRUCT(CAST(NULL AS STRING) AS is_commitment) AS discount,
                 FROM `doitintl-cmp-azure-data.azure_billing.doitintl_billing_export_v1` AS T
     ),
     grouped_md AS (
SELECT
    MD.billing_join_id AS billing_join_id,
    MIN(MD.primary_domain) AS primary_domain,
    MIN(MD.classification) AS classification,
    MIN(MD.territory) AS territory,
    MIN(MD.payee_country) AS payee_country,
    MIN(MD.payer_country) AS payer_country,
    MIN(MD.field_sales_representative) AS field_sales_representative,
    MIN(MD.strategic_account_manager) AS strategic_account_manager,
    MIN(MD.technical_account_manager) AS technical_account_manager,
    MIN(MD.customer_success_manager) AS customer_success_manager,
FROM
    `me-doit-intl-com.cloud_analytics.doitintl_csp_metadata_v1` AS MD
WHERE
        MD.type = "microsoft-azure"
GROUP BY
    MD.billing_join_id
    )

SELECT
    B.* EXCEPT(is_marketplace)
REPLACE(
      ARRAY(SELECT AS STRUCT r.cost AS cost, r.usage AS usage, r.savings AS savings, r.credit AS credit, r.margin AS margin, r.ext_metric AS ext_metric FROM UNNEST(B.report) AS r) AS report
    ),
    MD.classification,
    MD.primary_domain,
    MD.territory,
    MD.payee_country,
    MD.payer_country,
    MD.field_sales_representative,
    MD.strategic_account_manager,
    MD.technical_account_manager,
    MD.customer_success_manager,
    B.is_marketplace
  FROM
    enhanced_data AS B
  LEFT JOIN
    grouped_md AS MD
  ON
    B.billing_account_id = MD.billing_join_id
  WHERE
    DATE(export_time) BETWEEN @start_date AND @end_date
);


COMMIT TRANSACTION;

DECLARE start_date DATE DEFAULT IF(EXTRACT(DAY FROM @run_date) > 10, DATE_TRUNC(@run_date, MONTH), DATE_SUB(DATE_TRUNC(@run_date, MONTH), INTERVAL 1 MONTH));
DECLARE end_date DATE DEFAULT LAST_DAY(@run_date);

DECLARE current_day DATE DEFAULT start_date;

-- -- Create table if not exists
-- CREATE TABLE IF NOT EXISTS `doitintl-cmp-aws-data.aws_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL` (
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
DELETE `doitintl-cmp-aws-data.aws_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL`
WHERE DATE(export_time) BETWEEN start_date AND end_date;

WHILE current_day <= end_date DO

-- AWS billing data
INSERT `doitintl-cmp-aws-data.aws_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL`
SELECT *
FROM (
  WITH
  enhanced_data AS (
    SELECT
      T.*
      EXCEPT(etl, aws_metric, row_id, description, customer_id),
      IFNULL(ARRAY(
        SELECT
          STRUCT(CAST(D.is_commitment as STRING) as is_commitment)
            FROM
            `doitintl-cmp-aws-data.aws_billing_cmp.aws_discounts_v1` AS D
            WHERE
            T.project_id = D.project_id
              AND DATE(T.usage_date_time) >= D.start_date
              AND (D.end_date IS NULL OR DATE(T.usage_date_time) < D.end_date)
              AND DATE(_PARTITIONTIME) <= CURRENT_DATE()
            )[SAFE_OFFSET(0)], STRUCT(NULL as is_commitment)
        )
        AS discount
    FROM `doitintl-cmp-aws-data.aws_billing.doitintl_billing_export_v1` AS T
    WHERE
      DATE(export_time) = current_day
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
      MD.type = "amazon-web-services" OR MD.type = "amazon-web-services-standalone"
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
    DATE(export_time) = current_day
);

-- AWS Custom billing data
INSERT `doitintl-cmp-aws-data.aws_billing_CIgtnEximnd4fevT3qIU.doitintl_billing_export_v1_CIgtnEximnd4fevT3qIU_FULL`
SELECT *
FROM (
  WITH
  enhanced_data AS (
    SELECT
      T.*
      EXCEPT(etl, row_id, description, aws_metric, customer, max_timestamp)
      REPLACE(STRUCT(project.id, project.name, CAST([] AS ARRAY<STRUCT<key STRING, value STRING>>), project.ancestry_numbers, project.number) AS project),
      CAST(NULL AS STRING) AS resource_id,
      CAST(NULL AS STRING) AS operation,
      "resold" AS customer_type,
      CAST(NULL AS STRUCT<is_commitment STRING>) AS discount,
    FROM
      `doitintl-cmp-global-data.aws_custom_billing.aws_custom_billing_export_recent` AS T
  ),
  grouped_md AS (
    SELECT
      MD.customer_id AS customer_id,
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
      MD.type = "amazon-web-services" OR MD.type = "amazon-web-services-standalone"
    GROUP BY
      MD.customer_id
  )

  SELECT
    B.* REPLACE(
      ARRAY(SELECT AS STRUCT r.cost AS cost, r.usage AS usage, r.savings AS savings, r.credit AS credit, CAST(NULL AS FLOAT64) AS margin, CAST(NULL AS STRUCT<key STRING, value FLOAT64, type STRING>) AS ext_metric FROM UNNEST(B.report) AS r) AS report
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
    FALSE AS is_marketplace
  FROM
    enhanced_data AS B
  LEFT JOIN
    grouped_md AS MD
  ON
    B.billing_account_id = MD.customer_id
  WHERE
    DATE(export_time) = current_day
);

SET current_day = DATE_ADD(current_day, INTERVAL 1 DAY);
END WHILE;

COMMIT TRANSACTION;

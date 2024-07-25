DECLARE start_date DATE DEFAULT DATE_SUB(DATE_TRUNC(@run_date, DAY), INTERVAL 3 DAY);
DECLARE end_date DATE DEFAULT DATE_TRUNC(@run_date, DAY);

BEGIN TRANSACTION;
-- Delete the partitions to be replaced
DELETE `doitintl-cmp-gcp-data.gcp_billing_E2EE2E_E2EE2E_E2EE2E.doitintl_billing_export_v1beta_E2EE2E_E2EE2E_E2EE2E`
WHERE DATE(export_time) BETWEEN start_date AND end_date;

DELETE `doitintl-cmp-gcp-data.gcp_billing_E2EE2E_E2EE2E_E2EE2E.doitintl_billing_export_v1beta_E2EE2E_E2EE2E_E2EE2E_DAY`
WHERE DATE(export_time) BETWEEN start_date AND end_date;

-- GCP billing data
INSERT `doitintl-cmp-gcp-data.gcp_billing_E2EE2E_E2EE2E_E2EE2E.doitintl_billing_export_v1beta_E2EE2E_E2EE2E_E2EE2E`
SELECT
    *
        EXCEPT(plps_doit_percent)
        REPLACE("E2EE2E-E2EE2E-E2EE2E" AS billing_account_id)
FROM
    `doitintl-cmp-gcp-data.gcp_billing_04C2FF_77782A_6D4B18.doitintl_billing_export_v1_04C2FF_77782A_6D4B18`
WHERE DATE(export_time) BETWEEN start_date AND end_date;

INSERT `doitintl-cmp-gcp-data.gcp_billing_E2EE2E_E2EE2E_E2EE2E.doitintl_billing_export_v1beta_E2EE2E_E2EE2E_E2EE2E_DAY`
SELECT * REPLACE("E2EE2E-E2EE2E-E2EE2E" AS billing_account_id) FROM `doitintl-cmp-gcp-data.gcp_billing_04C2FF_77782A_6D4B18.doitintl_billing_export_v1_04C2FF_77782A_6D4B18_DAY`
WHERE DATE(export_time) BETWEEN start_date AND end_date;

COMMIT TRANSACTION;

package domain

// CspFieldsGCP - for metadata queries of GCP CSP
const CspFieldsGCP string = `ARRAY_AGG(DISTINCT territory IGNORE NULLS ORDER BY territory) AS csp_territory,
			ARRAY_AGG(DISTINCT primary_domain IGNORE NULLS ORDER BY primary_domain) AS csp_primary_domain,
			ARRAY_AGG(DISTINCT classification IGNORE NULLS ORDER BY classification) AS csp_classification,
			ARRAY_AGG(DISTINCT field_sales_representative IGNORE NULLS ORDER BY field_sales_representative) AS csp_field_sales_representative,
			ARRAY_AGG(DISTINCT strategic_account_manager IGNORE NULLS ORDER BY strategic_account_manager) AS csp_strategic_account_manager,
			ARRAY_AGG(DISTINCT technical_account_manager IGNORE NULLS ORDER BY technical_account_manager) AS csp_technical_account_manager,
			ARRAY_AGG(DISTINCT customer_success_manager IGNORE NULLS ORDER BY customer_success_manager) AS csp_customer_success_manager,
			ARRAY_AGG(DISTINCT payee_country IGNORE NULLS ORDER BY payee_country) AS csp_payee_country,
			ARRAY_AGG(DISTINCT payer_country IGNORE NULLS ORDER BY payer_country) AS csp_payer_country,
			ARRAY_AGG(DISTINCT customer_type IGNORE NULLS ORDER BY customer_type) AS customer_type,
			["true", "false"] AS csp_committed`

// CspFieldsAWS - for metadata queries of AWS CSP
const CspFieldsAWS string = `ARRAY_AGG(DISTINCT territory IGNORE NULLS ORDER BY territory) AS csp_territory,
			ARRAY_AGG(DISTINCT primary_domain IGNORE NULLS ORDER BY primary_domain) AS csp_primary_domain,
			ARRAY_AGG(DISTINCT classification IGNORE NULLS ORDER BY classification) AS csp_classification,
			ARRAY_AGG(DISTINCT field_sales_representative IGNORE NULLS ORDER BY field_sales_representative) AS csp_field_sales_representative,
			ARRAY_AGG(DISTINCT strategic_account_manager IGNORE NULLS ORDER BY strategic_account_manager) AS csp_strategic_account_manager,
			ARRAY_AGG(DISTINCT technical_account_manager IGNORE NULLS ORDER BY technical_account_manager) AS csp_technical_account_manager,
			ARRAY_AGG(DISTINCT customer_success_manager IGNORE NULLS ORDER BY customer_success_manager) AS csp_customer_success_manager,
			ARRAY_AGG(DISTINCT payee_country IGNORE NULLS ORDER BY payee_country) AS csp_payee_country,
			ARRAY_AGG(DISTINCT payer_country IGNORE NULLS ORDER BY payer_country) AS csp_payer_country,
			ARRAY_AGG(DISTINCT customer_type IGNORE NULLS ORDER BY customer_type) AS customer_type,
			["true", "false"] AS csp_committed`

const CspFieldsAzure string = `ARRAY_AGG(DISTINCT territory IGNORE NULLS ORDER BY territory) AS csp_territory,
			ARRAY_AGG(DISTINCT primary_domain IGNORE NULLS ORDER BY primary_domain) AS csp_primary_domain,
			ARRAY_AGG(DISTINCT classification IGNORE NULLS ORDER BY classification) AS csp_classification,
			ARRAY_AGG(DISTINCT field_sales_representative IGNORE NULLS ORDER BY field_sales_representative) AS csp_field_sales_representative,
			ARRAY_AGG(DISTINCT strategic_account_manager IGNORE NULLS ORDER BY strategic_account_manager) AS csp_strategic_account_manager,
			ARRAY_AGG(DISTINCT technical_account_manager IGNORE NULLS ORDER BY technical_account_manager) AS csp_technical_account_manager,
			ARRAY_AGG(DISTINCT customer_success_manager IGNORE NULLS ORDER BY customer_success_manager) AS csp_customer_success_manager,
			ARRAY_AGG(DISTINCT payee_country IGNORE NULLS ORDER BY payee_country) AS csp_payee_country,
			ARRAY_AGG(DISTINCT payer_country IGNORE NULLS ORDER BY payer_country) AS csp_payer_country,
			ARRAY_AGG(DISTINCT customer_type IGNORE NULLS ORDER BY customer_type) AS customer_type,
			["true", "false"] AS csp_committed`

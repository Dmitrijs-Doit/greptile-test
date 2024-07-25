package service

const (
	fixedFields  string = "fixed_fields"
	labels       string = "labels"
	systemLabels string = "system_labels"
	reportValues string = "report_values"
)

const comma string = ", "

const (
	rawDataTableString string = `WITH
	raw_data AS (
		SELECT *,
		@cloud_provider AS cloud_provider
		FROM {table} AS T
		WHERE DATE(export_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL @days_lookback DAY)
	)`

	filteredDataTableString string = `
	filtered_data AS (
		SELECT *
		FROM raw_data AS T
		{attributions_filters}
	)`

	fixedFieldsString string = `
	fixed_fields AS (
		SELECT
			NULL AS year,
			NULL AS quarter,
			NULL AS month,
			NULL AS week,
			NULL AS day,
			NULL AS hour,
			NULL AS week_day,
			NULL AS attribution,
			[@cloud_provider] AS cloud_provider,
			ARRAY_AGG(DISTINCT billing_account_id IGNORE NULLS ORDER BY billing_account_id LIMIT @values_limit) AS billing_account_id,
			ARRAY_AGG(DISTINCT project_id IGNORE NULLS ORDER BY project_id LIMIT @projects_limit) AS project_id,
			ARRAY_AGG(DISTINCT project.number IGNORE NULLS ORDER BY project.number LIMIT @projects_limit) AS project_number,
			ARRAY_AGG(DISTINCT project.name IGNORE NULLS ORDER BY project.name LIMIT @projects_limit) AS project_name,
			ARRAY_AGG(DISTINCT service_description IGNORE NULLS ORDER BY service_description LIMIT @values_limit) AS service_description,
			ARRAY_AGG(DISTINCT service_id IGNORE NULLS ORDER BY service_id LIMIT @values_limit) AS service_id,
			ARRAY_AGG(DISTINCT sku_description IGNORE NULLS ORDER BY sku_description LIMIT @values_limit) AS sku_description,
			ARRAY_AGG(DISTINCT sku_id IGNORE NULLS ORDER BY sku_id LIMIT @values_limit) AS sku_id,
			ARRAY_AGG(DISTINCT operation IGNORE NULLS ORDER BY operation LIMIT @values_limit) AS operation,
			ARRAY_AGG(DISTINCT location.country IGNORE NULLS ORDER BY location.country LIMIT @values_limit) AS country,
			ARRAY_AGG(DISTINCT location.region IGNORE NULLS ORDER BY location.region LIMIT @values_limit) AS region,
			ARRAY_AGG(DISTINCT location.zone IGNORE NULLS ORDER BY location.zone LIMIT @values_limit) AS zone,
			ARRAY_AGG(DISTINCT usage.pricing_unit IGNORE NULLS ORDER BY usage.pricing_unit LIMIT @values_limit) AS pricing_unit,
			["true", "false"] AS is_marketplace,
			ARRAY_AGG(DISTINCT invoice.month IGNORE NULLS ORDER BY invoice.month LIMIT @values_limit) AS invoice_month,
			{customer_feature_field},
			{additional_fields}
		FROM filtered_data
		LIMIT 1
		)
	`

	labelsString string = `
	labels AS (
		SELECT
		  ARRAY_AGG(STRUCT(label_key AS key, label_values AS values)) AS labels,
          ARRAY_AGG(DISTINCT label_key IGNORE NULLS ORDER BY label_key) AS labels_keys
		FROM (
			SELECT
				label_key,
				ARRAY_AGG(label_value ORDER BY frequency DESC LIMIT @labels_limit) AS label_values
			FROM (
				SELECT
					label.key AS label_key,
					IFNULL(label.value, @empty_labels_value) AS label_value,
					COUNT(label.value) AS frequency
				FROM filtered_data
				LEFT JOIN
					UNNEST (labels) AS label
				WHERE
					label IS NOT NULL
					AND label.key IS NOT NULL
				GROUP BY label_key, label_value
				)
			GROUP BY label_key)
		LIMIT 1
	)`

	systemLabelsString string = `
	system_labels AS (
		SELECT
		  ARRAY_AGG(STRUCT(label_key AS key, label_values AS values)) AS system_labels,
	      ARRAY_AGG(DISTINCT label_key IGNORE NULLS ORDER BY label_key) AS system_labels_keys
		FROM (
			SELECT
				label_key,
				ARRAY_AGG(label_value ORDER BY frequency DESC LIMIT @labels_limit) AS label_values
			FROM (
				SELECT
					label.key AS label_key,
					IFNULL(label.value, @empty_labels_value) AS label_value,
					COUNT(label.value) AS frequency
				FROM filtered_data
				LEFT JOIN
					UNNEST (system_labels) AS label
				WHERE
					label IS NOT NULL
					AND label.key IS NOT NULL
					AND LENGTH(label.value) <= 256
				GROUP BY label_key, label_value
				)
			GROUP BY label_key)
		LIMIT 1
	)`

	reportValuesTableString string = `
	report_values AS (
		SELECT
		  	ARRAY_AGG(DISTINCT credit IGNORE NULLS ORDER BY credit) AS credit,
		  	ARRAY_AGG(DISTINCT savings_description IGNORE NULLS ORDER BY savings_description) AS savings_description,
			ARRAY_AGG(DISTINCT cost_type IGNORE NULLS ORDER BY cost_type) AS cost_type
			FROM (
		  		SELECT r.credit AS credit, r.savings_description AS savings_description, cost_type FROM filtered_data LEFT JOIN UNNEST(report) AS r
		  		UNION ALL
		  		SELECT r.credit AS credit, CAST(NULL AS STRING) AS savings_description, cost_type FROM {credits_table} LEFT JOIN UNNEST(report) AS r {credits_where_clause}
			)
	)`

	// TODO: Change back to reportValuesTableString when CSP tables support savings_description
	reportValuesCSPTableStringTemp string = `
	report_values AS (
		SELECT
		  	ARRAY_AGG(DISTINCT credit IGNORE NULLS ORDER BY credit) AS credit,
			ARRAY_AGG(DISTINCT cost_type IGNORE NULLS ORDER BY cost_type) AS cost_type
			FROM (
		  		SELECT r.credit AS credit, cost_type FROM filtered_data LEFT JOIN UNNEST(report) AS r
		  		UNION ALL
		  		SELECT r.credit AS credit, cost_type FROM {credits_table} LEFT JOIN UNNEST(report) AS r {credits_where_clause}
			)
	)`
)

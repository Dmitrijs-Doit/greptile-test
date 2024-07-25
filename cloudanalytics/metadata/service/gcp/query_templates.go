package service

const (
	fixedFields             string = "fixed_fields"
	labels                  string = "labels"
	tags                    string = "tags"
	systemLabels            string = "system_labels"
	projectLabels           string = "project_labels"
	credits                 string = "credits"
	rawDataUnionSelectStart string = "SELECT * FROM ("
	rawDataUnionSelectEnd   string = ")"
)

const comma string = ", "

const (
	rawDataTableString string = `WITH
	raw_data AS (
		{union_select_start}
			SELECT * EXCEPT ({except_fields}),
			{aliased_fields},
			@cloud_provider AS cloud_provider,
			CAST(NULL AS STRING) AS operation
			FROM {table}
			{looker_table_select}
		{union_select_end} AS T
		 WHERE 
            DATE(export_time) >= 
            DATE_SUB(
                CURRENT_DATE(),
                INTERVAL 
                    CASE 
                        WHEN CAST(is_marketplace AS STRING) = 'true' THEN @marketplace_lookback 
                        ELSE @days_lookback 
                    END 
                DAY
            )
		AND export_time IS NOT NULL
	)`
	filterDataTableString string = `
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
			ARRAY_AGG(DISTINCT project.ancestry_names IGNORE NULLS ORDER BY project.ancestry_names LIMIT @values_limit) AS project_ancestry_names,
			ARRAY_AGG(DISTINCT project_id IGNORE NULLS ORDER BY project_id LIMIT @projects_limit) AS project_id,
			ARRAY_AGG(DISTINCT project.number IGNORE NULLS ORDER BY project.number LIMIT @projects_limit) AS project_number,
			ARRAY_AGG(DISTINCT project.name IGNORE NULLS ORDER BY project.name LIMIT @projects_limit) AS project_name,
			ARRAY_AGG(DISTINCT service_description IGNORE NULLS ORDER BY service_description LIMIT @values_limit) AS service_description,
			ARRAY_AGG(DISTINCT service_id IGNORE NULLS ORDER BY service_id LIMIT @values_limit) AS service_id,
			ARRAY_AGG(DISTINCT sku_description IGNORE NULLS ORDER BY sku_description LIMIT @values_limit) AS sku_description,
			ARRAY_AGG(DISTINCT sku_id IGNORE NULLS ORDER BY sku_id LIMIT @values_limit) AS sku_id,
			ARRAY_AGG(DISTINCT location.country IGNORE NULLS ORDER BY location.country LIMIT @values_limit) AS country,
			ARRAY_AGG(DISTINCT location.region IGNORE NULLS ORDER BY location.region LIMIT @values_limit) AS region,
			ARRAY_AGG(DISTINCT location.zone IGNORE NULLS ORDER BY location.zone LIMIT @values_limit) AS zone,
			ARRAY_AGG(DISTINCT usage.pricing_unit IGNORE NULLS ORDER BY usage.pricing_unit LIMIT @values_limit) AS pricing_unit,
			["true", "false"] AS is_marketplace,
			ARRAY_AGG(DISTINCT invoice.month IGNORE NULLS ORDER BY invoice.month LIMIT @values_limit) AS invoice_month,
			{customer_feature_field},
			{additional_fields}
		FROM filtered_data
	)`
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
	tagsString string = `
		tags AS (
			SELECT
			  ARRAY_AGG(STRUCT(tag_key AS key, tag_values AS values)) AS tags,
			  ARRAY_AGG(DISTINCT tag_key IGNORE NULLS ORDER BY tag_key) AS tags_keys
			FROM (
				SELECT
					tag_key,
					ARRAY_AGG(tag_value ORDER BY frequency DESC LIMIT @labels_limit) AS tag_values
				FROM (
					SELECT
						tag.key AS tag_key,
						IFNULL(tag.value, @empty_labels_value) AS tag_value,
						COUNT(tag.value) AS frequency
					FROM filtered_data
					LEFT JOIN
						UNNEST (tags) AS tag
					WHERE
						tag IS NOT NULL
						AND tag.key IS NOT NULL
					GROUP BY tag_key, tag_value
					)
				GROUP BY tag_key)
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
	projectLabelsString string = `
		project_labels AS (
			SELECT
			  ARRAY_AGG(STRUCT(label_key AS key, label_values AS values)) AS project_labels,
			  ARRAY_AGG(DISTINCT label_key IGNORE NULLS ORDER BY label_key) AS project_labels_keys 
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
						UNNEST (project.labels) AS label
					WHERE
						label IS NOT NULL
						AND label.key IS NOT NULL
					GROUP BY label_key, label_value
					)
				GROUP BY label_key)
			LIMIT 1
		)`
	creditsTableString string = `
		credits AS (
			SELECT
		  	ARRAY_AGG(DISTINCT credit IGNORE NULLS ORDER BY credit) AS credit,
			ARRAY_AGG(DISTINCT cost_type IGNORE NULLS ORDER BY cost_type) AS cost_type
			FROM (
		  		SELECT r.credit AS credit, cost_type FROM filtered_data LEFT JOIN UNNEST(report) AS r
		  		UNION ALL
		  		SELECT r.credit AS credit, cost_type FROM {creditsTable} LEFT JOIN UNNEST(report) AS r {creditsWhereClause}
			)
	  )`
)

package service

const (
	fixedFieldsNest = `
		fixed_fields AS (
			SELECT
				NULL AS year,
				NULL AS quarter,
				NULL AS month,
				NULL AS week,
				NULL AS day,
				NULL AS hour,
				NULL AS week_day,
				ARRAY_AGG(DISTINCT service_description IGNORE NULLS ORDER BY service_description LIMIT @values_limit) AS service_description,
				ARRAY_AGG(DISTINCT projectId IGNORE NULLS ORDER BY projectId LIMIT @projects_limit) AS project_name,
				ARRAY_AGG(DISTINCT location IGNORE NULLS ORDER BY location LIMIT @values_limit) AS region,
				ARRAY_AGG(DISTINCT pricing_unit IGNORE NULLS ORDER BY pricing_unit LIMIT @values_limit) AS pricing_unit,
				ARRAY_AGG(DISTINCT resource_id IGNORE NULLS ORDER BY resource_id LIMIT @values_limit) AS resource_id,
				ARRAY_AGG(DISTINCT jobStatus IGNORE NULLS ORDER BY jobStatus LIMIT @values_limit) AS job_status,
				ARRAY_AGG(DISTINCT statementType IGNORE NULLS ORDER BY statementType LIMIT @values_limit) AS statement_type,
				ARRAY_AGG(DISTINCT queryPriority IGNORE NULLS ORDER BY queryPriority LIMIT @values_limit) AS query_priority,
				ARRAY_AGG(DISTINCT callerIp IGNORE NULLS ORDER BY callerIp LIMIT @values_limit) AS caller_ip,
				ARRAY_AGG(DISTINCT user IGNORE NULLS ORDER BY user LIMIT @values_limit) AS user,
				ARRAY_AGG(DISTINCT eventName IGNORE NULLS ORDER BY eventName LIMIT @values_limit) AS event_name,
				ARRAY_AGG(DISTINCT reservation IGNORE NULLS ORDER BY reservation LIMIT @values_limit) AS reservation,
			FROM filtered_data
			LIMIT 1
		)`

	labelsString = `
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
					ORDER BY label_key, frequency DESC)
				GROUP BY label_key)
			LIMIT 1
		)`
)

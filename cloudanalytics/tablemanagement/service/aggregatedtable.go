package service

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	tablemanagementDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func GetAggregatedTableQuery(allPartitions bool, data *tablemanagementDomain.AggregatedQueryData) string {
	if data.IsCSP {
		return getAggregatedTableQueryCSP(allPartitions, data)
	}

	query := `WITH grouped_data AS (
	SELECT
		{select_fields},
		TIMESTAMP_TRUNC(export_time, {interval}) AS export_time,
		DATETIME_TRUNC(usage_date_time, {interval}) AS usage_date_time,
		TIMESTAMP_TRUNC(usage_start_time, {interval}) AS usage_start_time,
		TIMESTAMP_TRUNC(usage_end_time, {interval}) AS usage_end_time,
		STRUCT(
			ANY_VALUE(project.id) AS id,
			ANY_VALUE(project.number) AS number,
			ANY_VALUE(project.name) AS name,
			ANY_VALUE(CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>)) AS labels,
			ANY_VALUE(project.ancestry_numbers) AS ancestry_numbers,
			{project_ancestry_names} AS ancestry_names
		) AS project,
		CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>) AS labels,
		CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>) AS system_labels,
		STRUCT(
			ANY_VALUE(location.location) AS location,
			ANY_VALUE(location.country) AS country,
			ANY_VALUE(location.region) AS region,
			ANY_VALUE(location.zone) AS zone
		) AS location,
		ARRAY_TO_STRING([location.location, location.country, location.region, location.zone], ",", "NULL") AS location_agg_key,
		SUM(cost) AS cost,
		STRUCT(
			SUM(usage.amount) AS amount,
			ANY_VALUE(usage.unit) AS unit,
			SUM(usage.amount_in_pricing_units) AS amount_in_pricing_units,
			ANY_VALUE(usage.pricing_unit) AS pricing_unit
		) AS usage,
		STRUCT(
			ANY_VALUE(invoice.month) AS month
		) AS invoice,
		ARRAY_CONCAT_AGG(report) AS report,
		{null_tags_field},
	FROM
		{full_table_name}
	WHERE
		{where_clause}
	GROUP BY
		{group_by_clause}
)

SELECT
	*
	EXCEPT(location_agg_key)
	REPLACE(
		(SELECT ARRAY(
		SELECT AS STRUCT
			{report_select_fields},
			r.credit AS credit,
			STRUCT (
				r.ext_metric.key AS key,
				SUM(IFNULL(r.ext_metric.value, 0)) AS value,
				r.ext_metric.type AS type
			) AS ext_metric
		FROM UNNEST(report) AS r
		GROUP BY {report_group_by_clause})) AS report
	)
FROM grouped_data`

	var whereClause string
	if allPartitions {
		whereClause = `DATE(export_time) >= DATE("2018-01-01")`
	} else {
		whereClause = "DATE(export_time) = DATE(@partition)"
	}

	if data.BillingAccountID != "" {
		whereClause += fmt.Sprintf(` AND billing_account_id = "%s"`, data.BillingAccountID)
	}

	reportFields := []string{"cost", "usage", "savings"}
	groupByFields := []string{
		"export_time", "usage_date_time", "usage_start_time", "usage_end_time", "location_agg_key",
	}
	groupByFields = append(groupByFields, data.AdditionalFields...)

	var (
		reportGroupByFields []string
	)

	reportGroupByFields = []string{"credit", "savings_description", "r.ext_metric.key", "r.ext_metric.type"}

	for i, r := range reportFields {
		reportFields[i] = fmt.Sprintf("SUM(IFNULL(r.%s, 0)) AS %s", r, r)
	}

	r := []string{
		"{full_table_name}", data.FullBillingDataTableFullName,
		"{select_fields}", strings.Join(data.AdditionalFields, ",\n\t\t"),
		"{where_clause}", whereClause,
		"{group_by_clause}", strings.Join(groupByFields, ", "),
		"{interval}", data.AggregationInterval,
		"{null_tags_field}", domain.NullTags,
		"{report_group_by_clause}", strings.Join(reportGroupByFields, ", "),
	}

	if data.Cloud == common.Assets.GoogleCloud {
		if !data.IsCSP {
			// TODO: After savings_description added to GCP, move this and AWS to the reportFields definition
			reportFields = append(reportFields, "CAST(NULL AS STRING) AS savings_description")
		}

		r = append(r, "{project_ancestry_names}", "ANY_VALUE(project.ancestry_names)")
		r = append(r, "{report_select_fields}", strings.Join(reportFields, ",\n\t\t\t"))
	} else {
		if !data.IsCSP {
			reportFields = append(reportFields, "r.savings_description AS savings_description")
		}

		r = append(r, "{project_ancestry_names}", "ANY_VALUE(CAST(NULL AS STRING))")
		r = append(r, "{report_select_fields}", strings.Join(reportFields, ",\n\t\t\t"))
	}

	return strings.NewReplacer(r...).Replace(query)
}

func getAggregatedTableQueryCSP(allPartitions bool, data *tablemanagementDomain.AggregatedQueryData) string {
	query := `WITH grouped_data AS (
	SELECT
		{select_fields},
		TIMESTAMP_TRUNC(export_time, {interval}) AS export_time,
		DATETIME_TRUNC(usage_date_time, {interval}) AS usage_date_time,
		TIMESTAMP_TRUNC(usage_start_time, {interval}) AS usage_start_time,
		TIMESTAMP_TRUNC(usage_end_time, {interval}) AS usage_end_time,

		-- project fields
		ANY_VALUE(project.id) AS id,
		ANY_VALUE(project.number) AS number,
		ANY_VALUE(project.name) AS name,
		ANY_VALUE(project.ancestry_numbers) AS ancestry_numbers,
		{project_ancestry_names} AS ancestry_names,

		NULL AS project,
		NULL AS labels,
		NULL AS system_labels,

		-- location fields
		location.location,
		location.country,
		location.region,
		location.zone,

		SUM(CASE WHEN r.ext_metric.key IS NULL THEN t.cost ELSE 0 END) AS cost,

		-- usage fields
		NULL AS usage,
		SUM(CASE WHEN r.ext_metric.key IS NULL THEN t.usage.amount ELSE 0 END) AS amount,
		ANY_VALUE(t.usage.unit) AS unit,
		SUM(CASE WHEN r.ext_metric.key IS NULL THEN t.usage.amount_in_pricing_units ELSE 0 END) AS amount_in_pricing_units,
		ANY_VALUE(t.usage.pricing_unit) AS pricing_unit,

		ANY_VALUE(invoice.month) AS month,
		NULL AS invoice,

		-- report fields
		NULL AS report,
		{report_select_fields},
		r.credit AS r_credit,
		r.ext_metric.key AS r_key,
		SUM(IFNULL(r.ext_metric.value, 0)) AS r_value,
		r.ext_metric.type AS r_type,

		NULL AS tags,
		{csp_select_fields}
	FROM
		{full_table_name} t, UNNEST(report) r
	WHERE
		{where_clause}
	GROUP BY
		ALL
), aggregated_report AS (
	SELECT
		*
		EXCEPT( {temp_report_fields}, r_credit, r_key, r_value, r_type)
		REPLACE(
			ARRAY_AGG(
				STRUCT(
				r_cost AS cost,
				r_usage AS usage,
				r_savings AS savings,
				r_margin AS margin,
				r_credit AS credit,
				STRUCT ( r_key AS key,
					r_value AS value,
					r_type AS type) AS ext_metric
				)
			) AS report,
			SUM(cost) AS cost,
			SUM(amount) AS amount,
			SUM(amount_in_pricing_units) AS amount_in_pricing_units
		)
	FROM
		grouped_data
	GROUP BY
		ALL
)

SELECT
	*
	EXCEPT(country, region, zone, id, number, name, ancestry_numbers, amount, unit, amount_in_pricing_units, pricing_unit, month)
	REPLACE(
		STRUCT(
			location AS location,
			country AS country,
			region AS region,
			zone AS zone
		) AS location,
		STRUCT(
			id AS id,
			number AS number,
			name AS name,
			CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>) AS labels,
			ancestry_numbers AS ancestry_numbers,
			CAST(NULL AS STRING) AS ancestry_names
		) AS project,
		CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>) AS labels,
		CAST(NULL AS ARRAY<STRUCT<key STRING, value STRING>>) AS system_labels,
		STRUCT(
			month AS month
		) AS invoice,
		STRUCT(
			amount,
			unit,
			amount_in_pricing_units,
			pricing_unit
		) AS usage,
		{null_tags_field}
		)
FROM
	aggregated_report`

	var whereClause string
	if allPartitions {
		whereClause = `DATE(export_time) >= DATE("2018-01-01")`
	} else {
		whereClause = "DATE(export_time) = DATE(@partition)"
	}

	if data.BillingAccountID != "" {
		whereClause += fmt.Sprintf(` AND billing_account_id = "%s"`, data.BillingAccountID)
	}

	reportFields := []string{"cost", "usage", "savings"}
	tempFields := []string{} // used in EXCEPT clause

	var (
		cspReportFields []string
	)

	reportFields = append(reportFields, "margin")
	cspReportFields = []string{
		domain.FieldPrimaryDomain,
		domain.FieldClassification,
		domain.FieldTerritory,
		domain.FieldPayeeCountry,
		domain.FieldPayerCountry,
		domain.FieldFSR,
		domain.FieldSAM,
		domain.FieldTAM,
		domain.FieldCSM,
	}

	for i, r := range reportFields {
		reportFields[i] = fmt.Sprintf("SUM(IFNULL(r.%s, 0)) AS r_%s", r, r)
		tempFields = append(tempFields, fmt.Sprintf("r_%s", r))
	}

	r := []string{
		"{full_table_name}", data.FullBillingDataTableFullName,
		"{select_fields}", strings.Join(data.AdditionalFields, ",\n\t\t"),
		"{csp_select_fields}", strings.Join(cspReportFields, ",\n\t"),
		"{where_clause}", whereClause,
		"{interval}", data.AggregationInterval,
		"{null_tags_field}", domain.NullTags,
	}

	if data.Cloud == common.Assets.GoogleCloud {
		r = append(r, "{project_ancestry_names}", "ANY_VALUE(project.ancestry_names)")
	} else {
		r = append(r, "{project_ancestry_names}", "ANY_VALUE(CAST(NULL AS STRING))")
	}

	r = append(r, "{report_select_fields}", strings.Join(reportFields, ",\n\t\t\t"))
	r = append(r, "{temp_report_fields}", strings.Join(tempFields, ", "))

	return strings.NewReplacer(r...).Replace(query)
}

package bqlens

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	bigqueryUtils "github.com/doitintl/hello/scheduled-tasks/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	reportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	bqJobCompletedEventField  = "protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent"
	sinkTableBaseID           = "cloudaudit_googleapis_com_data_access"
	customerBQLogsSinkDataset = "doitintl_cmp_bq"

	bqCustomClientNilTemplate = "failed to get bq client for customer %s"

	editionsReservationSelectStatement = "protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.reservation AS reservation"
	reservationMappingSelectStatement  = "reservation_mapping.reservation AS reservation"
	editionsSlotMsToSlotHour           = "(1000 * 60 * 60)"

	// 30.437 is the number used by Google to compute the slot hour cost across every month.
	legacyFlatRateSlotMsToSlotMonth = "(30.437 * 24 * 1000 * 60 * 60)"
)

func GetBQDiscountQuery(withClauseDiscountsTable string) (discountsQuery string) {
	return fmt.Sprintf(`WITH %s
	SELECT
		MIN(IFNULL(SAFE_DIVIDE(cost, cost_at_list), 1.0) * IFNULL(discount.value, 1.0)) AS total_discount,
		NULL AS error_checks
	FROM
		raw_data
	WHERE
		DATE(export_time) >= DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY)
		AND cost > 0
		AND service_description = 'BigQuery'
		AND sku_description LIKE '%%Analysis%%'
  `, withClauseDiscountsTable)
}

func GetBQAuditLogsTableSubQuery(
	ctx context.Context,
	bqLensQueryArgs *BQLensQueryArgs,
) (string, error) {
	editionsAnalysisPricesSubQuery, err := GetBQEditionsAnalysisPrices(bqLensQueryArgs)
	if err != nil {
		return "", err
	}

	const bqAuditLogsSubQuery = `
	SELECT
		*,
		CASE
			WHEN reservation != "unreserved" THEN totalSlotMs / {slot_cost_unit_divisor}  * {editions_analysis_prices}
			ELSE usageTb * {on_demand_analysis_prices}
			END
		AS cost
	FROM (
		SELECT
			"{bq_access_audit_logs_source}" AS cloud_provider,
			"BigQuery" AS service_description,
			timestamp,
			protopayload_auditlog.requestMetadata.callerIp,
			protopayload_auditlog.authenticationInfo.principalEmail AS user,
			{bq_job_completed_event_field}.eventName,
			{bq_job_completed_event_field}.job.jobName.projectId,
			{reservation_selector},
			LOWER({bq_job_completed_event_field}.job.jobName.location) AS location,
			{bq_job_completed_event_field}.job.jobStatistics.startTime,
			{bq_job_completed_event_field}.job.jobStatistics.totalBilledBytes,
			{bq_job_completed_event_field}.job.jobStatistics.totalProcessedBytes,
			{bq_job_completed_event_field}.job.jobStatistics.totalLoadOutputBytes,
			{bq_job_completed_event_field}.job.jobStatistics.totalTablesProcessed,
			{bq_job_completed_event_field}.job.jobStatistics.totalBilledBytes / POW(1024,4) AS usageTb,
			{bq_job_completed_event_field}.job.jobStatistics.totalSlotMs AS totalSlotMs,
			{bq_job_completed_event_field}.job.jobConfiguration.labels,
			{bq_job_completed_event_field}.job.jobConfiguration.query.queryPriority,
			{bq_job_completed_event_field}.job.jobConfiguration.query.statementType,
			{bq_job_completed_event_field}.job.jobStatus.state AS jobStatus,
			TIMESTAMP_DIFF({bq_job_completed_event_field}.job.jobStatistics.endTime, {bq_job_completed_event_field}.job.jobStatistics.startTime, MILLISECOND) AS executionTimeMs,
			protopayload_auditlog.resourceName AS resource_id,
			NULL AS pricing_unit
		FROM {table}
		{reservation_mapping_join}
		WHERE {bq_job_completed_event_field}.job.jobName.jobId NOT LIKE 'script_job_%'
	)`

	const reservationMappingJoinClause = `
	LEFT JOIN
  		reservation_mapping
	ON
  		reservation_mapping.project_id=protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId`

	reservationSelect := editionsReservationSelectStatement
	reservationMappingJoin := ""

	if bqLensQueryArgs.ReservationMappingWithClause != "" {
		reservationSelect = reservationMappingSelectStatement
		reservationMappingJoin = reservationMappingJoinClause
	}

	slotCostUnitDivisor := editionsSlotMsToSlotHour

	if len(bqLensQueryArgs.FlatRateUsageTypes) > 0 {
		slotCostUnitDivisor = legacyFlatRateSlotMsToSlotMonth
	}

	return strings.NewReplacer(
		"{table}", bqLensQueryArgs.CustomerBQLogsTableID,
		"{on_demand_analysis_prices}", GetBQOnDemandAnalysisPrices(),
		"{editions_analysis_prices}", editionsAnalysisPricesSubQuery,
		"{bq_access_audit_logs_source}", string(reportDomain.DataSourceBQLens),
		"{bq_job_completed_event_field}", bqJobCompletedEventField,
		"{reservation_selector}", reservationSelect,
		"{reservation_mapping_join}", reservationMappingJoin,
		"{slot_cost_unit_divisor}", slotCostUnitDivisor,
	).Replace(bqAuditLogsSubQuery), nil
}

func GetBQOnDemandAnalysisPrices() string {
	return `CASE
		WHEN location = "asia-east1" THEN 5.75
		WHEN location = "asia-east2" THEN 7
		WHEN location = "asia-northeast1" THEN 6
		WHEN location = "asia-northeast2" THEN 6
		WHEN location = "asia-northeast3" THEN 6
		WHEN location = "asia-south1" THEN 6
		WHEN location = "asia-south2" THEN 6
		WHEN location = "asia-southeast1" THEN 6.75
		WHEN location = "asia-southeast2" THEN 6
		WHEN location = "australia-southeast1" THEN 6.5
		WHEN location = "australia-southeast2" THEN 6.5
		WHEN location = "europe-central2" THEN 6.5
		WHEN location = "europe-north1" THEN 6
		WHEN location = "europe-southwest1" THEN 6.25
		WHEN location = "europe-west1" THEN 6
		WHEN location = "europe-west2" THEN 6.25
		WHEN location = "europe-west3" THEN 6.5
		WHEN location = "europe-west4" THEN 6
		WHEN location = "europe-west6" THEN 7
		WHEN location = "europe-west8" THEN 6.25
		WHEN location = "europe-west9" THEN 6.25
		WHEN location = "northamerica-northeast1" THEN 5.25
		WHEN location = "northamerica-northeast2" THEN 5.25
		WHEN location = "southamerica-east1" THEN 9
		WHEN location = "southamerica-west1" THEN 7.15
		WHEN location = "us-central1" THEN 5
		WHEN location = "us-east1" THEN 5
		WHEN location = "us-east4" THEN 5
		WHEN location = "us-west1" THEN 5
		WHEN location = "us-west2" THEN 6.75
		WHEN location = "us-west3" THEN 6.75
		WHEN location = "us-west4" THEN 5
		WHEN location = "eu" THEN 5
		WHEN location = "us" THEN 6.25
		WHEN location = "aws-us-east-1" THEN 7.82
		WHEN location = "azure-eastus2" THEN 9.13
		WHEN location = "aws-ap-northeast-2" THEN 10
		ELSE 7.5
	END`
}

func GetBQAuditLogsFields(discount string) (orderedFields []string, nonNullFieldsMapping map[string]string) {
	// Define report field with all its metrics
	FieldLogsReport := strings.Replace(`STRUCT(cost * @discount AS cost, usageTb AS usage, CAST(cost * (1.0 - @discount) AS FLOAT64) AS savings, CASE WHEN @discount < 1.0 THEN "discount" ELSE "" END AS savings_description, CAST(NULL AS STRING) AS credit, CAST(NULL AS STRUCT<key STRING, value FLOAT64, type STRING>) AS ext_metric)`, "@discount", discount, -1)
	FieldLogsReportExtendedMetricTotalProcessedBytes := fmt.Sprintf(queryDomain.FieldLogsReportExtendedMetricTemplate, `"total_processed_bytes"`, "totalProcessedBytes")
	FieldLogsReportExtendedMetricTotalBilledBytes := fmt.Sprintf(queryDomain.FieldLogsReportExtendedMetricTemplate, `"total_billed_bytes"`, "totalBilledBytes")
	FieldLogsReportExtendedMetricTotalTablesProcessed := fmt.Sprintf(queryDomain.FieldLogsReportExtendedMetricTemplate, `"total_tables_processed"`, "totalTablesProcessed")
	FieldLogsReportExtendedMetricTotalLoadOutputBytes := fmt.Sprintf(queryDomain.FieldLogsReportExtendedMetricTemplate, `"total_load_output_bytes"`, "totalLoadOutputBytes")
	FieldLogsReportExtendedMetricTotalSlotsMs := fmt.Sprintf(queryDomain.FieldLogsReportExtendedMetricTemplate, `"total_slots_ms"`, "totalSlotMs")
	FieldLogsReportExtendedMetricSlotsUsed := fmt.Sprintf(queryDomain.FieldLogsReportExtendedMetricTemplate, `"slots_usage"`, "SAFE_DIVIDE(totalSlotMs,executionTimeMs)")

	fieldReport := fmt.Sprintf(`[%s, %s, %s, %s, %s, %s, %s]`,
		FieldLogsReport,
		FieldLogsReportExtendedMetricTotalProcessedBytes,
		FieldLogsReportExtendedMetricTotalBilledBytes,
		FieldLogsReportExtendedMetricTotalTablesProcessed,
		FieldLogsReportExtendedMetricTotalLoadOutputBytes,
		FieldLogsReportExtendedMetricTotalSlotsMs,
		FieldLogsReportExtendedMetricSlotsUsed,
	)

	orderedFields = []string{
		queryDomain.FieldProjectID,
		queryDomain.FieldProjectNameCF,
		queryDomain.FieldProject,
		queryDomain.FieldServiceDescription,
		queryDomain.FieldUsageDateTime,
		queryDomain.FieldExportTime,
		queryDomain.FieldLocation,
		queryDomain.FieldLabels,
		queryDomain.FieldUsage,
		queryDomain.FieldBillingReport,
		queryDomain.FieldResourceID,
		queryDomain.FieldLogsUser,
		queryDomain.FieldLogsCallerIP,
		queryDomain.FieldLogsEventName,
		queryDomain.FieldLogsQueryPriority,
		queryDomain.FieldLogsStatementType,
		queryDomain.FieldLogsJobStatus,
		queryDomain.FieldReservation,
	}

	nonNullFieldsMapping = map[string]string{
		queryDomain.FieldProjectID:          "projectId",
		queryDomain.FieldProjectNameCF:      "projectId",
		queryDomain.FieldProject:            "STRUCT(projectId AS id, NULL AS number, projectId AS name, NULL AS labels, NULL AS ancestry_numbers, NULL AS ancestry_names)",
		queryDomain.FieldServiceDescription: "service_description",
		queryDomain.FieldUsageDateTime:      "CAST(startTime AS DATETIME)",
		queryDomain.FieldExportTime:         "timestamp",
		queryDomain.FieldLocation:           "STRUCT(NULL AS location, NULL AS country, NULL AS region, NULL AS zone)",
		queryDomain.FieldLabels:             "labels",
		queryDomain.FieldUsage:              "STRUCT(pricing_unit AS pricing_unit, pricing_unit AS unit, usageTb AS amount_in_pricing_units)",
		queryDomain.FieldBillingReport:      fieldReport,
		queryDomain.FieldResourceID:         "resource_id",
		queryDomain.FieldLogsUser:           "user",
		queryDomain.FieldLogsCallerIP:       "callerIp",
		queryDomain.FieldLogsEventName:      "eventName",
		queryDomain.FieldLogsQueryPriority:  "queryPriority",
		queryDomain.FieldLogsStatementType:  "statementType",
		queryDomain.FieldLogsJobStatus:      "jobStatus",
		queryDomain.FieldReservation:        "reservation",
	}

	return orderedFields, nonNullFieldsMapping
}

func GetCustomerBQClient(ctx context.Context, fs *firestore.Client, customerID string) (*bigquery.Client, error) {
	docSnaps, err := bigqueryUtils.GetCustomerBQLensCloudConnectDocs(ctx, fs, customerID)
	if err != nil {
		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, fmt.Errorf(bqCustomClientNilTemplate, customerID)
	}

	// There should be only 1 relevant service account.
	// In the future could consider multiple org case and using filters to specify which to use.
	var cloudConnect common.GoogleCloudCredential

	if err := docSnaps[0].DataTo(&cloudConnect); err != nil {
		return nil, err
	}

	cred, err := common.NewGcpCustomerAuthService(&cloudConnect).GetClientOption()
	if err != nil {
		return nil, err
	}

	customerGCPProjectID, err := extractProjectIDFromSA(cloudConnect.ClientEmail)
	if err != nil {
		return nil, err
	}

	customerBQClient, err := bigquery.NewClient(ctx, customerGCPProjectID, cred)
	if err != nil {
		return nil, err
	}

	return customerBQClient, nil
}

func extractProjectIDFromSA(serviceAccountEmail string) (string, error) {
	parts := strings.Split(serviceAccountEmail, "@")
	if len(parts) < 2 {
		return "", errors.New("invalid service account email")
	}

	// Example: service-account-name@<project_id>.iam.gserviceaccount.com
	return strings.Split(parts[1], ".")[0], nil
}

func GetCustomerBQLogsSinkTable(ctx context.Context, customerBQClient *bigquery.Client) (sinkTableName string, err error) {
	ts := customerBQClient.Dataset(customerBQLogsSinkDataset).Tables(ctx)

	for {
		t, err := ts.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return sinkTableName, err
		}

		if strings.HasPrefix(t.TableID, sinkTableBaseID) {
			sinkTableName = sinkTableBaseID // old format
			if strings.HasPrefix(t.TableID, fmt.Sprintf("%s_", sinkTableBaseID)) {
				sinkTableName = fmt.Sprintf("%s_*", sinkTableBaseID) // new format
			}

			break
		}
	}

	return fmt.Sprintf(consts.FullTableTemplateQuotes, customerBQClient.Project(), customerBQLogsSinkDataset, sinkTableName), nil
}

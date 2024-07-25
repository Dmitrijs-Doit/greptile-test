package utils

import (
	"fmt"
	"strings"
	"time"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
)

type SegmentLength int

const (
	SegmentLengthInvalid SegmentLength = iota
	SegmentLengthHour
	SegmentLengthDay
	SegmentLengthMonth
)

func GetInternalUpdateQuery(md *dataStructures.InternalTaskMetadata) (string, error) {
	tableName, err := GetFullTableName(md.BQTable)

	var customerType string
	if md.BillingAccount == googleCloudConsts.MasterBillingAccount {
		customerType = string(common.AssetTypeResold)
	} else if md.Dummy {
		customerType = consts.CustomerTypeDummy
	} else {
		customerType = string(common.AssetTypeStandalone)
	}

	if err != nil {
		return "", err
	}

	query := fmt.Sprintf(
		"SELECT %d as iteration, \"%s\" as customer_type, * FROM `%s` WHERE export_time > \"%s\"",
		md.Iteration, customerType, tableName, md.Segment.StartTime.Format(consts.ExportTimeLayoutWithMillis))
	if md.Segment.EndTime != nil {
		query = fmt.Sprintf("%s AND export_time <= \"%s\"", query, md.Segment.EndTime.Format(consts.ExportTimeLayoutWithMillis))
	}

	return query, nil
}

func GetAlternativeInternalUpdateQuery(md *dataStructures.InternalTaskMetadata) (string, error) {
	tableName, err := GetFullTableName(&dataStructures.BillingTableInfo{
		ProjectID: md.BQTable.ProjectID,
		DatasetID: consts.AlternativeLocalBillingDataset,
		TableID:   GetAlternativeLocalCopyAccountTableName(md.BillingAccount),
	})

	var customerType string
	if md.BillingAccount == googleCloudConsts.MasterBillingAccount {
		customerType = string(common.AssetTypeResold)
	} else if md.Dummy {
		customerType = consts.CustomerTypeDummy
	} else {
		customerType = string(common.AssetTypeStandalone)
	}

	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SELECT %d as iteration, \"%s\" as customer_type, * FROM `%s`", md.Iteration, customerType, tableName)

	return query, nil
}

func GetLocalLatestExportTimeQuery(billingAccountID string) string {
	return fmt.Sprintf(
		"SELECT export_time FROM `%s` WHERE export_time <= CURRENT_TIMESTAMP() ORDER BY export_time DESC LIMIT 1",
		GetLocalCopyAccountTableFullName(billingAccountID))
}

func GetExportDataToBucketQuery(table *dataStructures.BillingTableInfo, bucketName string, Segment *dataStructures.Segment) (string, error) {
	tableName, err := GetFullTableName(table)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("EXPORT DATA OPTIONS(uri='%s', format='JSON', compression='GZIP', overwrite=true) AS SELECT * FROM `%s` WHERE export_time > '%s' AND export_time <= '%s'", bucketName, tableName, Segment.StartTime.Format(consts.ExportTimeLayoutWithMillis), Segment.EndTime.Format(consts.ExportTimeLayoutWithMillis)), nil
}

func GetTableOldestRecordTimeQuery(md *dataStructures.InternalTaskMetadata) (string, error) {
	fullTableName, err := GetFullTableName(md.BQTable)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" GROUP BY 1 ORDER BY 1 ASC LIMIT 1",
		fullTableName, time.Now().Format(consts.PartitionFieldFormat)), nil
}

func GetDeleteRowsFromUnifiedByBA(billingAccount string) string {
	if billingAccount == googleCloudConsts.MasterBillingAccount {
		return fmt.Sprintf("DELETE  FROM `%s` WHERE customer_type=\"%s\"", GetUnifiedTableFullName(), common.AssetTypeResold)
	} else {
		return fmt.Sprintf("DELETE  FROM `%s` WHERE billing_account_id=\"%s\"", GetUnifiedTableFullName(), billingAccount)
	}
}

type order string

const (
	DESC order = "DESC"
	ASC  order = "ASC"
)

func GetUnifiedTableOldestRecordByBA(md *dataStructures.InternalTaskMetadata) string {
	return getUnifiedTableMostExtremeRecordByBA(md, ASC)
}

func GetUnifiedTableNewestRecordByBA(md *dataStructures.InternalTaskMetadata) string {
	return getUnifiedTableMostExtremeRecordByBA(md, DESC)
}

func getUnifiedTableMostExtremeRecordByBA(md *dataStructures.InternalTaskMetadata, order order) string {
	if md.BillingAccount == googleCloudConsts.MasterBillingAccount {
		return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" AND customer_type=\"%s\" GROUP BY 1 ORDER BY 1 %s LIMIT 1",
			GetUnifiedTableFullName(), time.Now().Format(consts.PartitionFieldFormat), common.AssetTypeResold, order)
	} else {
		return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" AND billing_account_id =\"%s\" GROUP BY 1 ORDER BY 1 %s LIMIT 1",
			GetUnifiedTableFullName(), time.Now().Format(consts.PartitionFieldFormat), md.BillingAccount, order)
	}
}

func GetCustomerTableOldestRecordTimeQuery(t *dataStructures.BillingTableInfo) (string, error) {
	fullTableName, err := GetFullTableName(t)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" GROUP BY 1 ORDER BY 1 ASC LIMIT 1",
		fullTableName, time.Now().Format(consts.PartitionFieldFormat)), nil
}

func GetTableNewestRecordTime(md *dataStructures.InternalTaskMetadata) (string, error) {
	fullTableName, err := GetFullTableName(md.BQTable)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" GROUP BY 1 ORDER BY 1 DESC LIMIT 1",
		fullTableName, time.Now().Format(consts.PartitionFieldFormat)), nil
}

func GetCustomerRowCountQuery(table *dataStructures.BillingTableInfo, startRange, endRange *time.Time) (string, error) {
	fullTableName, err := GetFullTableName(table)
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SELECT COUNT(*) as rows_count FROM `%s`", fullTableName)
	if startRange != nil && endRange != nil {
		query = fmt.Sprintf("%s WHERE export_time < \"%s\" AND export_time >= \"%s\"", query, endRange.Format(consts.PartitionFieldFormat), startRange.Format(consts.PartitionFieldFormat))
	}

	return query, nil
}

func GetLocalRowCountQuery(billingAccount string, startRange, endRange *time.Time) (string, error) {
	fullTableName, err := GetFullTableName(GetLocalTableByBillingAccount(billingAccount))
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SELECT COUNT(*) as rows_count FROM `%s`", fullTableName)
	if startRange != nil && endRange != nil {
		query = fmt.Sprintf("%s WHERE export_time < \"%s\" AND export_time >= \"%s\"", query, endRange.Format(consts.PartitionFieldFormat), startRange.Format(consts.PartitionFieldFormat))
	}

	return query, nil
}

func GetAlternativeLocalRowCountQuery(billingAccount string, startRange, endRange *time.Time) (string, error) {
	fullTableName, err := GetFullTableName(&dataStructures.BillingTableInfo{
		ProjectID: GetProjectName(),
		DatasetID: consts.AlternativeLocalBillingDataset,
		TableID:   GetAlternativeLocalCopyAccountTableName(billingAccount),
	})
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SELECT COUNT(*) as rows_count FROM `%s`", fullTableName)
	if startRange != nil && endRange != nil {
		query = fmt.Sprintf("%s WHERE export_time < \"%s\" AND export_time >= \"%s\"", query, endRange.Format(consts.PartitionFieldFormat), startRange.Format(consts.PartitionFieldFormat))
	}

	return query, nil
}

func GetFromUnifiedRowCountQuery(billingAccount string, startRange, endRange *time.Time) (string, error) {
	query := fmt.Sprintf("SELECT COUNT(*) as rows_count FROM `%s`", GetUnifiedTableFullName())
	if billingAccount == googleCloudConsts.MasterBillingAccount {
		query = fmt.Sprintf("%s WHERE customer_type=\"%s\"", query, common.AssetTypeResold)
	} else {
		query = fmt.Sprintf("%s  WHERE billing_account_id=\"%s\"", query, billingAccount)
	}

	if startRange != nil && endRange != nil {
		query = fmt.Sprintf("%s AND export_time < \"%s\" AND export_time >= \"%s\"", query, endRange.Format(consts.PartitionFieldFormat), startRange.Format(consts.PartitionFieldFormat))
	}

	return query, nil
	// return fmt.Sprintf("SELECT COUNT(*) as rows_count FROM `%s` WHERE export_time < \"%s\" AND export_time >= \"%s\" AND customer_type=\"%s\"",
	//
	//	//return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" GROUP BY 1 ORDER BY 1 DESC LIMIT 1",
	//	GetUnifiedTableFullName(), endRange.Format(consts.PartitionFieldFormat), startRange.Format(consts.PartitionFieldFormat), common.AssetTypeResold), nil
}

func GetUnifiedTableFullName() string {
	return fmt.Sprintf("%s.%s.%s", GetProjectName(), consts.UnifiedGCPBillingDataset, consts.UnifiedGCPRawTable)
}

func GetCustomerTableNewestRecordTime(t *dataStructures.BillingTableInfo) (string, error) {
	fullTableName, err := GetFullTableName(t)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SELECT export_time FROM `%s` WHERE DATE(export_time) <= \"%s\" GROUP BY 1 ORDER BY 1 DESC LIMIT 1",
		fullTableName, time.Now().Format(consts.PartitionFieldFormat)), nil
}

func GetTableOldestRecordTimeNewerThan(t *dataStructures.BillingTableInfo, minExportTime time.Time) (string, error) {
	fullTableName, err := GetFullTableName(t)
	if err != nil {
		return "", err
	}

	resoldParam := ""
	if fullTableName == GetUnifiedTableFullName() {
		resoldParam = "AND customer_type = \"resold\" "
	}

	return fmt.Sprintf("SELECT export_time FROM `%s` WHERE export_time > \"%s\" %sGROUP BY 1 ORDER BY 1 ASC LIMIT 1",
		fullTableName, minExportTime.Format(consts.ExportTimeLayoutWithMillis), resoldParam), nil
}

func GetRawBillingNewestRecordTime() string {
	return fmt.Sprintf("SELECT export_time FROM `%s.%s.%s` WHERE DATE(export_time) <= \"%s\" GROUP BY 1 ORDER BY 1 DESC LIMIT 1",
		consts.BillingProjectProd, consts.ResellRawBillingDataset, consts.ResellRawBillingTable, time.Now().Format(consts.PartitionFieldFormat))
}

func GetRawBillingOldestRecordTime(long bool) string {
	var upperLimit string
	if long {
		upperLimit = consts.OldestRawBillingPartition
	} else {
		upperLimit = time.Now().UTC().Format(consts.PartitionFieldFormat)
	}

	return fmt.Sprintf("SELECT export_time FROM `%s.%s.%s` WHERE DATE(export_time) <= \"%s\"  GROUP BY 1 ORDER BY 1 ASC LIMIT 1",
		consts.BillingProjectProd, consts.ResellRawBillingDataset, consts.ResellRawBillingTable, upperLimit)
}

func GetMarkTmpTableBillingRowsAsVerifiedQuery(md *dataStructures.InternalTaskMetadata) string {
	if md.BillingAccount == googleCloudConsts.MasterBillingAccount {
		return fmt.Sprintf("UPDATE `%s` SET verified=true WHERE customer_type=\"%s\"", GetUnifiedTempTableFullName(md.Iteration), common.AssetTypeResold)
	} else {
		return fmt.Sprintf("UPDATE `%s` SET verified=true WHERE billing_account_id=\"%s\"", GetUnifiedTempTableFullName(md.Iteration), md.BillingAccount)
	}
}

func GetCopyFromTmpTableAllRowsQuery(iteration int64, itms []*dataStructures.InternalTaskMetadata) string {
	query := strings.Builder{}
	query.WriteString(fmt.Sprintf("SELECT CURRENT_TIMESTAMP() as doit_export_time, * FROM `%s`", GetUnifiedTempTableFullName(iteration)))

	for iteration, itm := range itms {
		if iteration == 0 {
			query.WriteString(" WHERE ")
		} else {
			query.WriteString(" OR ")
		}

		if itm.BillingAccount == googleCloudConsts.MasterBillingAccount {
			query.WriteString(fmt.Sprintf("customer_type=\"%s\"", common.AssetTypeResold))
		} else {
			query.WriteString(fmt.Sprintf("billing_account_id=\"%s\"", itm.BillingAccount))
		}
	}

	return query.String()
}

func GetRowsCountQuery(table *dataStructures.BillingTableInfo, billingAccountID string, segment *dataStructures.Segment) (string, SegmentLength, error) {
	tableName, err := GetFullTableName(table)
	if err != nil {
		return "", SegmentLengthInvalid, err
	}

	var timestampTrunc string

	var segmentLength SegmentLength

	if segment.EndTime.Sub(*segment.StartTime) <= 24*time.Hour {
		timestampTrunc = "HOUR"
		segmentLength = SegmentLengthHour
	} else if segment.EndTime.Sub(*segment.StartTime) < 60*24*time.Hour {
		timestampTrunc = "DAY"
		segmentLength = SegmentLengthDay
	} else {
		timestampTrunc = "MONTH"
		segmentLength = SegmentLengthMonth
	}

	query := fmt.Sprintf("SELECT TIMESTAMP_TRUNC(export_time,%s) time_stamp, COUNT(1) rows_count FROM `%s` WHERE export_time <= \"%s\" AND export_time > \"%s\"",
		timestampTrunc, tableName, segment.EndTime.Format(consts.ExportTimeLayoutWithMillis), segment.StartTime.Format(consts.ExportTimeLayoutWithMillis))

	if billingAccountID != "" {
		if billingAccountID != googleCloudConsts.MasterBillingAccount {
			query = fmt.Sprintf("%s AND billing_account_id=\"%s\"", query, billingAccountID)
		} else {
			query = fmt.Sprintf("%s AND %s=\"%s\"", query, consts.CustomerTypeField, common.AssetTypeResold)
		}
	}

	query = fmt.Sprintf("%s GROUP BY time_stamp ORDER BY time_stamp", query)

	return query, segmentLength, nil
}

func GetRowsCountByExportTimeQuery(table *dataStructures.BillingTableInfo, billingAccountID string, segment *dataStructures.Segment) (string, error) {
	tableName, err := GetFullTableName(table)
	if err != nil {
		return "", err
	}

	query := fmt.Sprintf("SELECT export_time, count(1) as rows_count  FROM `%s` WHERE export_time <= \"%s\" AND export_time > \"%s\"",
		tableName, segment.EndTime.Format(consts.ExportTimeLayoutWithMillis), segment.StartTime.Format(consts.ExportTimeLayoutWithMillis))

	if billingAccountID != "" {
		if billingAccountID != googleCloudConsts.MasterBillingAccount {
			query = fmt.Sprintf("%s AND billing_account_id=\"%s\"", query, billingAccountID)
		} else {
			query = fmt.Sprintf("%s AND %s=\"%s\"", query, consts.CustomerTypeField, common.AssetTypeResold)
		}
	}

	query = fmt.Sprintf("%s GROUP BY 1 ORDER BY 1 ASC", query)

	return query, nil
}

func GetDetailedTableRewritesMappingQuery() string {
	return fmt.Sprintf(
		`SELECT
			TIMESTAMP_TRUNC(CURRENT_DATETIME(), HOUR) run_time,
			TIMESTAMP_TRUNC(export_time,HOUR) time_stamp,
			SUM(cost) total_cost,
			COUNT(1) rows_count
		FROM %s.%s.%s
		WHERE DATE(export_time) >=  DATETIME_SUB(CURRENT_DATETIME(), %s)
		GROUP BY time_stamp ORDER BY time_stamp`,
		consts.BillingProjectProd, consts.ResellRawBillingDataset, consts.ResellRawBillingDetailedTable, consts.AnalyticsLoobackInterval)
}

func GetDetailedTableUsageAndExportTimeDifferential() string {
	return fmt.Sprintf(
		`WITH time_data AS
		(SELECT
			TIMESTAMP_TRUNC(usage_start_time, HOUR) usage_start_time_hour,
			TIMESTAMP_TRUNC(export_time,HOUR) export_time_hour,
			-- delay might be negative because the export time is effectively truncated to day, but usage start is not.
			IF(TIMESTAMP_DIFF(export_time, usage_start_time, HOUR) >= 0, TIMESTAMP_DIFF(export_time, usage_start_time, HOUR),0) AS delay,
			FROM %s.%s.%s
			WHERE DATE(export_time) >=  DATETIME_SUB(CURRENT_DATETIME(), %s)
			GROUP BY usage_start_time_hour, export_time_hour, delay ORDER BY usage_start_time_hour)
		SELECT
			CURRENT_DATETIME() AS report_time_stamp,
			MIN(export_time_hour) AS min_export_time,
			MAX(export_time_hour) AS max_export_time,
			MIN(usage_start_time_hour) AS min_usage_start_time,
			MAX(usage_start_time_hour) AS max_usage_start_time,
			MIN(delay) AS min_delay_hour,
			MAX(delay) AS max_delay_hour,
			AVG(delay) AS avg_delay_hour
			FROM time_data`,
		consts.BillingProjectProd, consts.ResellRawBillingDataset, consts.ResellRawBillingDetailedTable, consts.AnalyticsLoobackInterval)
}

func GetDetailedTableAnalyticsTableName() *dataStructures.BillingTableInfo {
	t := &dataStructures.BillingTableInfo{
		ProjectID: common.ProjectID,
		DatasetID: consts.AnalyticsDataset,
	}
	if common.ProjectID == consts.BillingProjectMeDoitIntlCom {
		t.TableID = consts.AnalyticsRewritesMappingTableProd
	} else {
		t.TableID = consts.AnalyticsRewritesMappingTableNonProd
	}

	return t
}

func GetFreshnessReportTableName() *dataStructures.BillingTableInfo {
	t := &dataStructures.BillingTableInfo{
		ProjectID: common.ProjectID,
		DatasetID: consts.AnalyticsDataset,
	}
	if common.ProjectID == consts.BillingProjectMeDoitIntlCom {
		t.TableID = consts.AnaliyticsFreshnessReportTableProd
	} else {
		t.TableID = consts.AnaliyticsFreshnessReportTableNonProd
	}

	return t
}

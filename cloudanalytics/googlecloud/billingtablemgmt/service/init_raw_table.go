package googlecloud

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	udfProjectReassignments string = "`doitintl-cmp-gcp-data.gcp_billing.UDF_PROJECT_REASSIGNMENTS_V1`"
	udfShouldExcludeCost    string = "`doitintl-cmp-gcp-data.gcp_billing.UDF_SHOULD_EXCLUDE_COST_V1`"

	// Google started offering resource billing in July, 2021, it is effected from the start of the previous month
	// for reference: https://cloud.google.com/billing/docs/how-to/export-data-bigquery-tables#resource-usage-cost-data-schema
	resourceBillingStartDate string = "2021-09-01"
)

func (s *BillingTableManagementService) getGoogleCloudBillingExportV1(billingAccountID string) string {
	return fmt.Sprintf("gcp_billing_export_v1_%s", strings.Replace(billingAccountID, "-", "_", -1))
}

func (s *BillingTableManagementService) getGoogleCloudResourceBillingExportV1(billingAccountID string) string {
	return fmt.Sprintf("gcp_billing_export_resource_v1_%s", strings.Replace(billingAccountID, "-", "_", -1))
}

func (s *BillingTableManagementService) InitRawTable(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	dstTable := bq.DatasetInProject(domain.GetBillingProject(), domain.BillingDataset).Table("gcp_raw_billing")

	billingExportV1 := s.getGoogleCloudBillingExportV1(googleCloudConsts.MasterBillingAccount)
	queryTemplate := `SELECT
	*
	REPLACE(
		{udf_project_reassignments}(billing_account_id, project.id, usage_start_time) AS billing_account_id,
		IF({udf_should_exclude_cost}(service, sku), 0, cost) AS cost
	)
FROM
	{table}`
	query := strings.NewReplacer(
		"{table}",
		fmt.Sprintf(consts.FullTableTemplate, domain.ResellerBillingExportProject, domain.ResellerBillingExportDataset, billingExportV1),
		"{udf_project_reassignments}",
		udfProjectReassignments,
		"{udf_should_exclude_cost}",
		udfShouldExcludeCost,
	).Replace(queryTemplate)

	queryJob := bq.Query(query)
	queryJob.Dst = dstTable
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.CreateDisposition = bigquery.CreateIfNeeded
	queryJob.WriteDisposition = bigquery.WriteEmpty
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: "cloud_analytics_gcp_init", AddJobIDSuffix: true}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "export_time"}
	queryJob.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseData.String(),
		common.LabelKeyFeature.String(): common.FeatureCloudAnalytics.String(),
		common.LabelKeyModule.String():  common.ModuleTableManagement.String(),
	}

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	status, err := job.Status(ctx)
	if err != nil {
		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	return nil
}

func (s *BillingTableManagementService) InitRawResourceTable(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	billingExportV1 := s.getGoogleCloudBillingExportV1(googleCloudConsts.MasterBillingAccount)
	resourceBillingExportV1 := s.getGoogleCloudResourceBillingExportV1(googleCloudConsts.MasterBillingAccount)

	dstTable := bq.DatasetInProject(domain.GetBillingProject(), domain.BillingDataset).Table(domain.GetRawBillingTableName(true))

	replacer := strings.NewReplacer(
		"{resource_table}",
		fmt.Sprintf(consts.FullTableTemplate, domain.ResellerBillingExportProject, domain.ResellerBillingExportDataset, resourceBillingExportV1),
		"{table}",
		fmt.Sprintf(consts.FullTableTemplate, domain.ResellerBillingExportProject, domain.ResellerBillingExportDataset, billingExportV1),
		"{udf_project_reassignments}",
		udfProjectReassignments,
		"{udf_should_exclude_cost}",
		udfShouldExcludeCost,
		"{start_date}",
		resourceBillingStartDate,
	)

	queryTemplate := `SELECT
	*
	EXCEPT(
		resource
	)
	REPLACE(
		{udf_project_reassignments}(billing_account_id, project.id, usage_start_time) AS billing_account_id,
		_PARTITIONTIME AS export_time,
		IF({udf_should_exclude_cost}(service, sku), 0, cost) AS cost
	),
	resource
FROM
	{resource_table}
WHERE
	_PARTITIONDATE >= "{start_date}"

UNION ALL

SELECT
	*
	REPLACE(
		{udf_project_reassignments}(billing_account_id, project.id, usage_start_time) AS billing_account_id,
		IF({udf_should_exclude_cost}(service, sku), 0, cost) AS cost
	),
	NULL AS resource
FROM
	{table}
WHERE
	_PARTITIONDATE < "{start_date}"`

	query := replacer.Replace(queryTemplate)
	queryJob := bq.Query(query)
	queryJob.Dst = dstTable
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.CreateDisposition = bigquery.CreateIfNeeded
	queryJob.WriteDisposition = bigquery.WriteEmpty
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: "cloud_analytics_gcp_resource_init", AddJobIDSuffix: true}
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "export_time"}
	queryJob.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseData.String(),
		common.LabelKeyFeature.String(): common.FeatureCloudAnalytics.String(),
		common.LabelKeyModule.String():  common.ModuleTableManagement.String(),
	}

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	status, err := job.Status(ctx)
	if err != nil {
		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	return nil
}

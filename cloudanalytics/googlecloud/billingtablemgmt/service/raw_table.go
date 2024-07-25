package googlecloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *BillingTableManagementService) UpdateRawTableLastPartition(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	numPartitions := 1
	today := time.Now().UTC().Truncate(time.Hour * 24)
	partition := today.Add(time.Hour * time.Duration(-24*numPartitions))

	for !partition.After(today) {
		l.Info(partition)

		queryJob := s.newBillingCopyParitionJob(bq, partition)
		queryJob.DryRun = false
		queryJob.UseLegacySQL = false
		queryJob.AllowLargeResults = true
		queryJob.DisableFlattenedResults = true
		queryJob.CreateDisposition = bigquery.CreateIfNeeded
		queryJob.WriteDisposition = bigquery.WriteTruncate
		queryJob.SchemaUpdateOptions = []string{"ALLOW_FIELD_ADDITION"}
		queryJob.Priority = bigquery.InteractivePriority
		queryJob.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "export_time"}
		queryJob.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}
		queryJob.Parameters = []bigquery.QueryParameter{{Name: "partition", Value: partition}}

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

		partition = partition.Add(time.Hour * time.Duration(24))
	}

	return nil
}

func (s *BillingTableManagementService) UpdateRawResourceTableLastPartition(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	numPartitions := 1
	today := time.Now().UTC().Truncate(time.Hour * 24)
	partition := today.Add(time.Hour * time.Duration(-24*numPartitions))

	for !partition.After(today) {
		l.Info(partition)

		queryJob := s.newResourceBillingCopyParitionJob(bq, partition)
		queryJob.DryRun = false
		queryJob.UseLegacySQL = false
		queryJob.AllowLargeResults = true
		queryJob.DisableFlattenedResults = true
		queryJob.CreateDisposition = bigquery.CreateIfNeeded
		queryJob.WriteDisposition = bigquery.WriteTruncate
		queryJob.SchemaUpdateOptions = []string{"ALLOW_FIELD_ADDITION"}
		queryJob.Priority = bigquery.InteractivePriority
		queryJob.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "export_time"}
		queryJob.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}
		queryJob.Parameters = []bigquery.QueryParameter{{Name: "partition", Value: partition}}

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

		partition = partition.Add(time.Hour * time.Duration(24))
	}

	return nil
}

func (s *BillingTableManagementService) newResourceBillingCopyParitionJob(bq *bigquery.Client, partition time.Time) *bigquery.Query {
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
	{table}
WHERE
	_PARTITIONDATE = DATE(@partition)`
	query := strings.NewReplacer(
		"{table}",
		fmt.Sprintf(consts.FullTableTemplate, domain.ResellerBillingExportProject, domain.ResellerBillingExportDataset, s.getGoogleCloudResourceBillingExportV1(googleCloudConsts.MasterBillingAccount)),
		"{udf_project_reassignments}",
		udfProjectReassignments,
		"{udf_should_exclude_cost}",
		udfShouldExcludeCost,
	).Replace(queryTemplate)

	queryJob := bq.Query(query)
	queryJob.Dst = bq.DatasetInProject(domain.BillingProjectProd, domain.BillingDataset).Table(fmt.Sprintf("%s$%s", domain.GetRawBillingTableName(true), partition.Format("20060102")))
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: "cloud_analytics_gcp_resource_table_copy_partition", AddJobIDSuffix: true}
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseData.String(),
		common.LabelKeyFeature.String(): common.FeatureCloudAnalytics.String(),
		common.LabelKeyModule.String():  common.ModuleTableManagement.String(),
	}

	return queryJob
}

func (s *BillingTableManagementService) newBillingCopyParitionJob(bq *bigquery.Client, partition time.Time) *bigquery.Query {
	queryTemplate := `SELECT
	*
	REPLACE(
		{udf_project_reassignments}(billing_account_id, project.id, usage_start_time) AS billing_account_id,
		IF({udf_should_exclude_cost}(service, sku), 0, cost) AS cost
	)
FROM
	{table}
WHERE
	_PARTITIONDATE = DATE(@partition)`
	query := strings.NewReplacer(
		"{table}",
		fmt.Sprintf(consts.FullTableTemplate, "billing-explorer", "gcp", s.getGoogleCloudBillingExportV1(googleCloudConsts.MasterBillingAccount)),
		"{udf_project_reassignments}",
		udfProjectReassignments,
		"{udf_should_exclude_cost}",
		udfShouldExcludeCost,
	).Replace(queryTemplate)

	queryJob := bq.Query(query)
	queryJob.Dst = bq.DatasetInProject(domain.BillingProjectProd, domain.BillingDataset).Table(fmt.Sprintf("%s$%s", "gcp_raw_billing", partition.Format("20060102")))
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: "cloud_analytics_gcp_table_copy_partition", AddJobIDSuffix: true}
	queryJob.Labels = map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   common.HouseData.String(),
		common.LabelKeyFeature.String(): common.FeatureCloudAnalytics.String(),
		common.LabelKeyModule.String():  common.ModuleTableManagement.String(),
	}

	return queryJob
}

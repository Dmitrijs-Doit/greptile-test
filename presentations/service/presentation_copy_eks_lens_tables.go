package service

import (
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	awsCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

func (p *PresentationService) UpdateCustomerEKSLensBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	_, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return err
	}

	return p.copyEKSBillingTable(ctx, customerID, incrementalUpdate)
}

func (p *PresentationService) UpdateEKSLensBillingData(ctx *gin.Context, incrementalUpdate bool) error {
	logger := p.Logger(ctx)
	logger.SetLabel(log.LabelPresentationUpdateStage.String(), "presentation-eks")

	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.AmazonWebServices, func(ctx *gin.Context, customerID string) error {
		return p.copyEKSBillingTable(ctx, customerID, incrementalUpdate)
	}); len(errors) > 0 {
		return fmt.Errorf("failed to update AWS billing data for some presentation customers: %v", errors)
	}

	return nil
}

func (p *PresentationService) copyEKSBillingTable(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.Infof("EQS billing update data for customer: %s, incrementalUpdate: %t", customerID, incrementalUpdate)

	bq := p.conn.Bigquery(ctx)
	connatix := getBqTable(bq, awsCloudConsts.EksProjectProd, connatixCustomerID, func(_ string) string { return awsCloudConsts.EksDataset }, func(suffix string, customerID string) string {
		return querytable.GetEksTableName(connatixCustomerID)
	})

	quinx := getBqTable(bq, awsCloudConsts.EksProjectProd, quinxCustomerID, func(_ string) string { return awsCloudConsts.EksDataset }, func(suffix string, customerID string) string {
		return querytable.GetEksTableName(quinxCustomerID)
	})

	destination, err := createTableIfNoneExists(ctx, bq, querytable.GetEksProject(), awsCloudConsts.EksDataset, fmt.Sprintf("%s%s", awsCloudConsts.EksTable, customerID), connatix.Table)
	if err != nil {
		l.Error(err)
		return err
	}

	exportTimeThreshold := getIncrementalUpdateThreshold(incrementalUpdate, destination)

	replacer := strings.NewReplacer(
		"{connatix_table}", connatix.FullTableName,
		"{quinx_table}", quinx.FullTableName,
		"{aws_demo_billing_id}", awsDemoBillingAccountID,
		"{moonActive_billing_id}", moonActiveCustomerID,
		"{connatix_billing_id}", connatixCustomerID,
		"{quinx_billing_id}", quinxCustomerID,
		"{aws_query_fields}", awsQueryFields,
		"{project_id_anonymizer}", createAwsProjectIDAnonymizer(customerID),
		"{resource_anonymizer}", resourceAnonymizer,
		"{labels_generator}", createLabelGenerator(),
		"{export_time_threshold}", exportTimeThreshold,
		"{choose_cluster_name}", eKSClusterNameAnonymizer,
		"{choose_pod_owner}", choosePodOwner,
	)

	q := bq.Query(replacer.Replace(`
	{project_id_anonymizer}
	{resource_anonymizer}
	{labels_generator}
    {choose_cluster_name}
    {choose_pod_owner}

	CREATE TEMP FUNCTION choose_namespace(value INT64) RETURNS STRING
	AS (
	  CASE
		WHEN MOD(value, 3) = 0 THEN 'Production Namespace'
		WHEN MOD(value, 3) = 1 THEN 'Staging Namespace'
		ELSE 'Development Namespace'
	  END
	);

	CREATE TEMP FUNCTION choose_label_k8(value INT64) RETURNS STRING
	AS (
	  CASE
		WHEN MOD(value, 3) = 0 THEN 'aws-node-1'
		WHEN MOD(value, 3) = 1 THEN 'aws-node-2'
		ELSE 'aws-node-3'
	  END
	);

	WITH raw_data AS (
		SELECT {aws_query_fields} FROM {connatix_table}
		WHERE billing_account_id = "{connatix_billing_id}"
			AND export_time > ({export_time_threshold}) AND export_time < CURRENT_TIMESTAMP()
		UNION ALL
			SELECT {aws_query_fields} FROM {quinx_table}
		WHERE billing_account_id = "{quinx_billing_id}"
			AND export_time > ({export_time_threshold}) AND export_time < CURRENT_TIMESTAMP()
	)
	SELECT
		*
		REPLACE (
			STRUCT(
				AWSProjectIdAnonymizer(project_id) AS id,
				AWSProjectIdAnonymizer(project_id) AS name,
				LabelGenerator(project_id) AS labels,
				CAST(NULL AS STRING) AS ancestry_numbers,
				CAST(NULL AS STRING) AS number
			) AS project,
			LabelGenerator(project_id) AS labels,
			ARRAY(
				SELECT AS STRUCT sl.key key,
				CASE
					WHEN sl.key = 'eks:cluster-name' THEN choose_cluster_name(ABS(FARM_FINGERPRINT(sl.value)))
					WHEN sl.key = 'EKS_namespace' THEN choose_namespace(ABS(FARM_FINGERPRINT(sl.value)))
					WHEN sl.key = 'EKS_label_k8s_app' THEN choose_label_k8(ABS(FARM_FINGERPRINT(sl.value)))
					WHEN sl.key = 'EKS_pod_owner_name' AND sl.value <> 'UNREQUESTED_COSTS' THEN choose_pod_owner(ABS(FARM_FINGERPRINT(sl.value)))
					ELSE sl.value
				END AS value from UNNEST(system_labels) AS sl where
				REGEXP_CONTAINS(sl.key, r"^cmp\/|^aws\/|^compute\.googleapis\.com/|eks:cluster-name|EKS_label_k8s_app|EKS_namespace|EKS_pod_owner_name")) AS system_labels,
			AWSProjectIdAnonymizer(project_id) AS project_id,
			"{aws_demo_billing_id}" AS billing_account_id,
			CAST(NULL AS STRING) AS customer_type,
			resourceAnonymizer(resource_id) AS resource_id
		),
	FROM raw_data
	`))

	l.Info("EKS billing querying data with query: ", q.QueryConfig.Q)

	q.CreateDisposition = bigquery.CreateIfNeeded
	q.Dst = destination.Table

	if incrementalUpdate {
		q.WriteDisposition = bigquery.WriteAppend
	} else {
		q.WriteDisposition = bigquery.WriteTruncate
	}

	job, err := q.Run(ctx)

	if err != nil {
		return err
	}

	l.Infof("EKS billing data update jobID: %s", job.ID())
	status, err := job.Wait(ctx)

	if err != nil {
		l.Error(fmt.Errorf("EKS billing query run failed for job %s. Caused by %s", job.ID(), err))
		return err
	}

	if status.Err() != nil {
		return status.Err()
	}

	return nil
}

var choosePodOwner = fmt.Sprintf(
	`CREATE TEMP FUNCTION choose_pod_owner(value INT64) RETURNS STRING AS (%s);`,
	pickVariantByModulo(tags, "value"))

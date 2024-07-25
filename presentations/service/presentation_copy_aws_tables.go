package service

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	analyticsAWS "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

func (p *PresentationService) UpdateCustomerAWSBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "billing")

	_, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return err
	}

	return p.copyAwsBillingTable(ctx, customerID, incrementalUpdate)
}

func (p *PresentationService) UpdateAWSBillingData(ctx *gin.Context, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "billing")

	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.AmazonWebServices, func(ctx *gin.Context, customerID string) error {
		return p.copyAwsBillingTable(ctx, customerID, incrementalUpdate)
	}); len(errors) > 0 {
		return fmt.Errorf("failed to update AWS billing data for some presentation customers: %v", errors)
	}

	return nil
}

func (p *PresentationService) copyAwsBillingTable(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.Infof("AWS billing update data for customer: %s; incrementalUpdate: %t", customerID, incrementalUpdate)

	bq := p.conn.Bigquery(ctx)

	moonActive := getBqTable(bq, analyticsAWS.BillingProjectProd, moonActiveCustomerID, analyticsAWS.GetCustomerBillingDataset, analyticsAWS.GetCustomerBillingTable)
	connatix := getBqTable(bq, analyticsAWS.BillingProjectProd, connatixCustomerID, analyticsAWS.GetCustomerBillingDataset, analyticsAWS.GetCustomerBillingTable)
	quinx := getBqTable(bq, analyticsAWS.BillingProjectProd, quinxCustomerID, analyticsAWS.GetCustomerBillingDataset, analyticsAWS.GetCustomerBillingTable)

	destination, err := createTableIfNoneExists(ctx, bq, analyticsAWS.GetBillingProject(), analyticsAWS.GetCustomerBillingDataset(customerID), analyticsAWS.GetCustomerBillingTable(customerID, ""), moonActive.Table)
	if err != nil {
		l.Error(err)
		return err
	}

	exportTimeThreshold := getIncrementalUpdateThreshold(incrementalUpdate, destination)

	replacer := strings.NewReplacer(
		"{moonactive_table}", moonActive.FullTableName,
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
		"{aws_org_tags_generator}", createAwsOrgTagsGenerator(),
		"{export_time_threshold}", exportTimeThreshold,
		"{choose_cluster_name}", eKSClusterNameAnonymizer,
	)

	q := getQueryWithLabels(ctx, bq, replacer.Replace(`
	{project_id_anonymizer}
	{resource_anonymizer}
	{labels_generator}
	{aws_org_tags_generator}
    {choose_cluster_name}

	WITH raw_data AS (
		SELECT {aws_query_fields} FROM {moonactive_table}
		WHERE billing_account_id = "{moonActive_billing_id}"
			AND export_time > ({export_time_threshold}) AND export_time < CURRENT_TIMESTAMP()
		UNION ALL
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
				ARRAY_CONCAT(LabelGenerator(project_id), AwsOrgTagsGenerator(project_id)) AS labels,
				CAST(NULL AS STRING) AS ancestry_numbers,
				CAST(NULL AS STRING) AS number
			) AS project,
			LabelGenerator(project_id) AS labels,
			ARRAY(
				SELECT AS STRUCT sl.key key, CASE
					WHEN sl.key="aws/payer_account_id" THEN ""
					WHEN sl.key ="eks:cluster-name" THEN choose_cluster_name(ABS(FARM_FINGERPRINT(sl.value)))
					ELSE sl.value
				END AS value from UNNEST(system_labels) AS sl where
					REGEXP_CONTAINS(sl.key, r"^cmp\/|^aws\/|^compute\.googleapis\.com/|^eks:cluster-name")
			) AS system_labels,
			AWSProjectIdAnonymizer(project_id) AS project_id,
			"{aws_demo_billing_id}" AS billing_account_id,
			CAST(NULL AS STRING) AS customer_type,
			resourceAnonymizer(resource_id) AS resource_id
		),
	FROM raw_data
	`), customerID)

	l.Info("AWS billing querying data with query: ", q.QueryConfig.Q)

	q.CreateDisposition = bigquery.CreateIfNeeded
	q.Dst = destination.Table

	if incrementalUpdate {
		q.WriteDisposition = bigquery.WriteAppend
	} else {
		q.WriteDisposition = bigquery.WriteTruncate
	}

	job, err := q.Run(ctx)
	if err != nil {
		l.Error(fmt.Errorf("AWS billing query run failed for job %s. Caused by %s", job.ID(), err))
		return err
	}

	l.Infof("AWS billing data update jobID: %s", job.ID())

	status, err := job.Wait(ctx)
	if err != nil {
		l.Error(fmt.Errorf("AWS billing unable to wait for job %s. Caused by %s", job.ID(), err))
		return err
	}

	if err := status.Err(); err != nil {
		l.Error(fmt.Errorf("AWS billing failed update for job %s. Caused by %s", job.ID(), err))
		return err
	}

	return nil
}

func createAwsProjectIDAnonymizer(customerID string) string {
	hash := sha1.New()
	hash.Write([]byte(customerID))
	projectIDPrefixAsHex := hash.Sum(nil)[0:4]
	projectIDPrefix := strconv.FormatInt(int64(binary.BigEndian.Uint32(projectIDPrefixAsHex)), 10)[0:4]

	return fmt.Sprintf(`CREATE TEMP FUNCTION AWSProjectIdAnonymizer(project_id STRING) RETURNS STRING AS
		(CONCAT(
			"%s",
			LEFT(ARRAY_TO_STRING(ARRAY(
				SELECT CAST(ASCII(c) AS STRING) FROM UNNEST(SPLIT(TO_BASE64(SHA1(project_id)), "")) AS c
			), ""), 12)
		));`, projectIDPrefix)
}

func createAwsOrgTagsGenerator() string {
	return fmt.Sprintf(`CREATE TEMP FUNCTION AwsOrgTagsGenerator(project_id STRING) RETURNS ARRAY<STRUCT<key STRING, value STRING>> AS (
		[
			STRUCT("aws-org/owner", %s),
			STRUCT("aws-org/playground", %s),
			STRUCT("aws-org/customer", %s),
			STRUCT("aws-org/cohort", %s)
		]
	);`,
		pickVariantByModulo(owners, projectIDNumericalHash),
		pickVariantByModulo(playground, projectIDNumericalHash),
		pickVariantByModulo(customers, firstHalfIDNumericalHash),
		pickVariantByModulo(cohorts, secondHalfIDNumericalHash),
	)
}

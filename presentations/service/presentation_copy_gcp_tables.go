package service

import (
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

func (p *PresentationService) UpdateCustomerGCPBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "billing")
	if _, err := p.getDemoCustomerFromID(ctx, customerID); err != nil {
		return err
	}

	return p.copyGCPBillingData(ctx, customerID, incrementalUpdate)
}

func (p *PresentationService) UpdateGCPBillingData(ctx *gin.Context, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "billing")
	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.GoogleCloud, func(ctx *gin.Context, customerID string) error {
		return p.copyGCPBillingData(ctx, customerID, incrementalUpdate)
	}); len(errors) > 0 {
		return fmt.Errorf("failed to update GCP billing data for some presentation customers: %v", errors)
	}

	return nil
}

func (p *PresentationService) copyGCPBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	billingAccountID := domain.HashCustomerIdIntoABillingAccountId(customerID)

	l := p.Logger(ctx)
	l.Infof("GCP billing update data for billing account: %s; incrementalUpdate: %t", billingAccountID, incrementalUpdate)

	bq := p.conn.Bigquery(ctx)

	destinationDataset := bq.DatasetInProject(gcpTableMgmtDomain.GetBillingProject(), gcpTableMgmtDomain.GetCustomerBillingDataset(billingAccountID))

	var (
		values []string
		keys   []string
	)

	gcpDemoTables := getDemoTableNames(gcpDemoBillingIds, bq)

	destination, err := createTableIfNoneExists(ctx, bq, gcpTableMgmtDomain.GetBillingProject(), gcpTableMgmtDomain.GetCustomerBillingDataset(billingAccountID), gcpTableMgmtDomain.GetCustomerBillingTable(billingAccountID, ""), gcpDemoTables["moonactive"].Table)
	if err != nil {
		l.Error(err)
		return err
	}

	exportTimeThreshold := getIncrementalUpdateThreshold(incrementalUpdate, destination)

	keys = append(keys, "{gcp_demo_billing_id}")
	keys = append(keys, "{raw_data}")
	keys = append(keys, "{create_gcp_label_generator}")
	keys = append(keys, additionalKeys...)

	values = append(values, billingAccountID)
	values = append(values, getSQLUnionQuery(getCustomerNames(gcpDemoBillingIds), gcpDemoTables, exportTimeThreshold))
	values = append(values, createLabelGenerator())
	values = append(values, additionalValues...)

	replacerArgs := make([]string, 0, len(keys)*2)
	for i, key := range keys {
		replacerArgs = append(replacerArgs, key, values[i])
	}

	replacer := strings.NewReplacer(replacerArgs...)

	q := getQueryWithLabels(ctx, bq, replacer.Replace(`
	{create_gcp_label_generator}
	{ancestry_names_anonymizer}
	{kubernetes_cluster_name_anonymizer}
	{kubernetes_namespace_anonymizer}
	{resource_anonymizer}
	{project_id_generator}

	WITH
		raw_data AS (
			{raw_data}
		)
	SELECT
			*
			REPLACE (
				STRUCT(
					ProjectIdGenerator(project_id) AS id,
					CAST(NULL AS STRING) AS number,
					CAST(NULL AS STRING) AS name,
				    LabelGenerator(project_id) AS labels,
					CAST(NULL AS STRING) AS ancestry_numbers,
					AncestryNamesAnonymizer(project.ancestry_names) AS ancestry_names
				) AS project,
        		LabelGenerator(project_id) AS labels,
				"{gcp_demo_billing_id}" AS billing_account_id,
				ARRAY(
					SELECT AS STRUCT sl.key key, sl.value value  FROM UNNEST(system_labels) AS sl WHERE
						REGEXP_CONTAINS(sl.key, r"^cmp\/|^aws\/|^compute\.googleapis\.com/")
				) AS system_labels,
				KubernetesClusterNameAnonymizer(kubernetes_cluster_name) AS kubernetes_cluster_name,
				KubernetesNamespaceAnonymizer(kubernetes_namespace) AS kubernetes_namespace,
				resourceAnonymizer(resource_id) AS resource_id,
				CAST(NULL AS STRING) AS resource_global_id,
				{null_tags_field},
				ProjectIdGenerator(project_id) AS project_id,
				CAST(NULL AS STRING) AS seller_name,
                CAST(NULL AS STRUCT<instance_id STRING>) AS subscription,
				STRUCT(CAST(price_book.discount AS FLOAT64) AS discount, CAST(price_book.unit_price AS FLOAT64) AS unit_price) AS price_book,
				STRUCT(CAST(discount.value AS FLOAT64) AS value, CAST(discount.rebase_modifier AS FLOAT64) AS rebase_modifier, discount.allow_preemptible as allow_preemptible, discount.is_commitment as is_commitment ) AS discount
			)

		FROM
			raw_data
		WHERE
			export_time > "2024-01-01"
`), customerID)

	l.Infof("GCP billing querying data with query: ", q.QueryConfig.Q)

	q.CreateDisposition = bigquery.CreateIfNeeded
	q.Dst = destinationDataset.Table(gcpTableMgmtDomain.GetCustomerBillingTable(billingAccountID, ""))

	if incrementalUpdate {
		q.WriteDisposition = bigquery.WriteAppend
	} else {
		q.WriteDisposition = bigquery.WriteTruncate
	}

	job, err := q.Run(ctx)
	if err != nil {
		l.Error(fmt.Errorf("GCP billing query run failed for job %s. Caused by %s", job.ID(), err))
		return err
	}

	l.Infof("GCP billing data update jobID: %s", job.ID())

	status, err := job.Wait(ctx)
	if err != nil {
		l.Error(fmt.Errorf("GCP billing unable to wait for job %s. Caused by %s", job.ID(), err))
		return err
	}

	if err := status.Err(); err != nil {
		l.Error(fmt.Errorf("GCP billing failed update for job %s. Caused by %s", job.ID(), err))
		return err
	}

	return nil
}

package service

import (
	"fmt"
	"math/rand"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	analyticsAzure "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

var (
	azureProductNameAnonymizer = fmt.Sprintf(
		`CREATE TEMP FUNCTION AzureProductNameAnonymizer(product_name STRING) RETURNS STRING AS (%s);`,
		pickVariantByModulo(azureProductNames, "ABS(FARM_FINGERPRINT(product_name))"))

	azureResourceAnonymizer = `CREATE TEMP FUNCTION AzureResourceAnonymizer(resource_id STRING) RETURNS STRING AS(
			ARRAY_TO_STRING(ARRAY(SELECT (CASE
				WHEN MOD(row_number, 2) = 0 THEN token
				WHEN token = "" THEN ""
				WHEN row_number = 3 THEN AzureUuidV4Anonymizer(token)
				WHEN REGEXP_CONTAINS(token, r"^(?i)microsoft\.") THEN token
				ELSE CommonNameAnonymizer(token)
			END) FROM (
			SELECT token, ROW_NUMBER() OVER () AS row_number  FROM UNNEST(SPLIT(resource_id, "/")) as token
			)), "/")
		);`
)

func (p *PresentationService) UpdateCustomerAzureBillingData(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "billing")
	_, err := p.getDemoCustomerFromID(ctx, customerID)

	if err != nil {
		return err
	}

	return p.copyAzureBillingTable(ctx, customerID, incrementalUpdate)
}

func (p *PresentationService) UpdateAzureBillingData(ctx *gin.Context, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "billing")

	if errors := p.runForEachPresentationCustomerWithAssetType(ctx, common.Assets.MicrosoftAzure, func(ctx *gin.Context, customerID string) error {
		return p.copyAzureBillingTable(ctx, customerID, incrementalUpdate)
	}); len(errors) > 0 {
		return fmt.Errorf("failed to update Azure billing data for some presentation customers: %v", errors)
	}

	return nil
}

func (p *PresentationService) copyAzureBillingTable(ctx *gin.Context, customerID string, incrementalUpdate bool) error {
	l := p.Logger(ctx)
	l.Infof("Azure billing update data for customer: %s; incrementalUpdate: %t", customerID, incrementalUpdate)

	bq := p.conn.Bigquery(ctx)

	taboola := getBqTable(bq, analyticsAzure.BillingProjectProd, taboolaCustomerID, analyticsAzure.GetCustomerBillingDataset, analyticsAzure.GetCustomerBillingTable)
	connatix := getBqTable(bq, analyticsAzure.BillingProjectProd, connatixCustomerID, analyticsAzure.GetCustomerBillingDataset, analyticsAzure.GetCustomerBillingTable)
	aigoAi := getBqTable(bq, analyticsAzure.BillingProjectProd, aigoAiCustomerID, analyticsAzure.GetCustomerBillingDataset, analyticsAzure.GetCustomerBillingTable)

	destination, err := createTableIfNoneExists(ctx, bq, analyticsAzure.GetBillingProject(), analyticsAzure.GetCustomerBillingDataset(customerID), analyticsAzure.GetCustomerBillingTable(customerID, ""), connatix.Table)
	if err != nil {
		l.Error(err)
		return err
	}

	exportTimeThreshold := getIncrementalUpdateThreshold(incrementalUpdate, destination)

	replacer := strings.NewReplacer(
		"{taboola_table}", taboola.FullTableName,
		"{connatix_table}", connatix.FullTableName,
		"{aigoai_table}", aigoAi.FullTableName,
		"{azure_query_fields}", azureQueryFields,
		"{export_time_threshold}", exportTimeThreshold,
		"{common_name_anonymizer}", commonNameAnonymizer,
		"{project_id_anonymizer}", createAzureProjectIDAnonymizer(customerID),
		"{product_name_anonymizer}", azureProductNameAnonymizer,
		"{resource_anonymized}", azureResourceAnonymizer,
		"{uuid_v4_anonymizer}", createAzureUUIDv4Anonymizer(customerID),
		"{system_label_anonymizer}", createAzureSystemLabelAnonymizer(),
		"{labels_generator}", createLabelGenerator(),
	)

	q := getQueryWithLabels(ctx, bq, replacer.Replace(`
	{common_name_anonymizer}
	{project_id_anonymizer}
	{product_name_anonymizer}
	{uuid_v4_anonymizer}
	{resource_anonymized}
	{system_label_anonymizer}
	{labels_generator}

	WITH raw_data AS (
		SELECT {azure_query_fields} FROM {taboola_table}
		WHERE export_time > ({export_time_threshold}) AND export_time < CURRENT_TIMESTAMP()
		UNION ALL
		SELECT {azure_query_fields} FROM {connatix_table}
		WHERE export_time > ({export_time_threshold}) AND export_time < CURRENT_TIMESTAMP()
		UNION ALL
		SELECT {azure_query_fields} FROM {aigoai_table}
		WHERE export_time > ({export_time_threshold}) AND export_time < CURRENT_TIMESTAMP()
	)
	SELECT
		*
		REPLACE (
			STRUCT(
				AzureProjectIdAnonymizer(project_id) AS id,
				AzureProjectIdAnonymizer(project_id) AS name,
				LabelGenerator(project_id) AS labels,
				CAST(NULL AS STRING) AS ancestry_numbers,
				CAST(NULL AS STRING) AS number
			) AS project,
			LabelGenerator(project_id) AS labels,
			ARRAY(
				SELECT AS STRUCT key, value FROM
					(SELECT * REPLACE(AnonymizeSystemLabel(billing_account_id, key, value) as value) FROM UNNEST(system_labels))
			) AS system_labels,
			AzureProjectIdAnonymizer(project_id) AS project_id,
			AzureUuidV4Anonymizer(billing_account_id) AS billing_account_id,
			CAST(NULL AS STRING) AS customer_type,
			AzureResourceAnonymizer(resource_id) AS resource_id,
			AzureUuidV4Anonymizer(tenant_id) AS tenant_id
		),
	FROM raw_data
	`), customerID)

	l.Info("Azure billing querying data with query: ", q.QueryConfig.Q)

	q.CreateDisposition = bigquery.CreateIfNeeded
	q.Dst = destination.Table

	if incrementalUpdate {
		q.WriteDisposition = bigquery.WriteAppend
	} else {
		q.WriteDisposition = bigquery.WriteTruncate
	}

	job, err := q.Run(ctx)
	if err != nil {
		l.Error(fmt.Errorf("Azure billing query run failed for job %s. Caused by %s", job.ID(), err))
		return err
	}

	l.Infof("Azure billing data update jobID: %s", job.ID())

	status, err := job.Wait(ctx)
	if err != nil {
		l.Error(fmt.Errorf("Azure billing unable to wait for job %s. Caused by %s", job.ID(), err))
		return err
	}

	if err := status.Err(); err != nil {
		l.Error(fmt.Errorf("Azure billing failed update for job %s. Caused by %s", job.ID(), err))
		return err
	}

	return nil
}

func createAzureProjectIDAnonymizer(customerID string) string {
	return fmt.Sprintf(`CREATE TEMP FUNCTION AzureProjectIdAnonymizer(project_id STRING) RETURNS STRING AS
		(ARRAY_TO_STRING(
			[
				%s,
				%s,
				RIGHT(CAST(ABS(FARM_FINGERPRINT(CONCAT(project_id, "%s"))) AS STRING), 4)
			], "-"
		));`,
		pickVariantByModulo(projectPrefixes, firstHalfIDNumericalHash),
		pickVariantByModulo(projectLabels, projectIDNumericalHash),
		customerID)
}

func createAzureUUIDv4Anonymizer(customerID string) string {
	hexLetters := []string{"a", "b", "c", "d", "e", "f"}
	r := rand.New(rand.NewSource(domain.Hash(customerID)))
	r.Shuffle(len(hexLetters), func(i, j int) { hexLetters[i], hexLetters[j] = hexLetters[j], hexLetters[i] })

	return fmt.Sprintf(`CREATE TEMP FUNCTION AzureUuidV4Anonymizer(billing_account_id STRING) RETURNS STRING AS
		(
			TRANSLATE(billing_account_id, "0123456789abcdef", CONCAT(RIGHT(CAST(ABS(FARM_FINGERPRINT("%s")) AS STRING), 10), "%s"))
		);`,
		customerID,
		strings.Join(hexLetters, ""),
	)
}

func createAzureSystemLabelAnonymizer() string {
	replacer := strings.NewReplacer(
		"{project}", pickVariantByModulo(projectSuffixes, "ABS(FARM_FINGERPRINT(billing_account_id))"),
		"{common_name}", pickVariantByModulo(commonNames, "ABS(FARM_FINGERPRINT(value))"),
	)

	return replacer.Replace(`CREATE TEMP FUNCTION AnonymizeSystemLabel(billing_account_id STRING, key STRING, value STRING) RETURNS STRING AS (
			CASE
				WHEN REGEXP_CONTAINS(key, r"^azure/publisher_name$") THEN value
				WHEN REGEXP_CONTAINS(key, r"_id$|Id$") THEN AzureUuidV4Anonymizer(value)
				WHEN REGEXP_CONTAINS(key, r"_uri$|Uri$") THEN AzureResourceAnonymizer(value)
				WHEN REGEXP_CONTAINS(key, r"^azure/billing_profile_name$|^azure/partner_name$") THEN "DoiT International"
				WHEN key = "azure/subscription_name" THEN ARRAY_TO_STRING(["azure", LOWER({project}), "com"], ".")
				WHEN key = "azure/invoice_section_name" THEN {project}
				WHEN key = "azure/azure/product_order_name" THEN AzureProductNameAnonymizer(value)
				WHEN REGEXP_CONTAINS(key, r"_name$|Name$|_info$|Info|Tags") THEN {common_name}
				ELSE value END
			);`)
}

package service

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

const (
	projectIDNumericalHash    = "ABS(FARM_FINGERPRINT(project_id))"
	firstHalfIDNumericalHash  = "ABS(FARM_FINGERPRINT(SUBSTR(project_id, 1, DIV(LENGTH(project_id),2))))"
	secondHalfIDNumericalHash = "ABS(FARM_FINGERPRINT(SUBSTR(project_id, DIV(LENGTH(project_id),2))))"
)

type bqTable struct {
	Table         *bigquery.Table
	FullTableName string
}

func getBqTable(bq *bigquery.Client, projectID string, customerID string, GetCustomerBillingDataset func(suffix string) string, GetCustomerBillingTable func(customerID string, tableSuffix string) string) *bqTable {
	table := bq.DatasetInProject(projectID, GetCustomerBillingDataset(customerID)).Table(GetCustomerBillingTable(customerID, ""))

	return &bqTable{
		Table:         table,
		FullTableName: strings.Join([]string{table.ProjectID, table.DatasetID, table.TableID}, "."),
	}
}

func pickVariantByModulo(variants []string, value string) string {
	formattedVariants := make([]string, 0)
	mod := len(variants)

	for _, variant := range variants {
		formattedVariants = append(formattedVariants, fmt.Sprintf("'%s'", variant))
	}

	return fmt.Sprintf("[%s][OFFSET(MOD(%s, %d))]", strings.Join(formattedVariants, ","), value, mod)
}

func createLabelGenerator() string {
	return fmt.Sprintf(`CREATE TEMP FUNCTION LabelGenerator(project_id STRING) RETURNS ARRAY<STRUCT<key STRING, value STRING>> AS (
		[
			STRUCT("app", %s),
			STRUCT("customer", %s),
			STRUCT("env", %s),
			STRUCT("product", %s),
			STRUCT("project", %s)
		]
	);`,
		pickVariantByModulo(projectSuffixes, secondHalfIDNumericalHash),
		pickVariantByModulo(customerLabels, projectIDNumericalHash),
		pickVariantByModulo(projectPrefixes, firstHalfIDNumericalHash),
		pickVariantByModulo(projectSuffixes, projectIDNumericalHash),
		pickVariantByModulo(projectLabels, projectIDNumericalHash))
}

const resourceAnonymizer string = `CREATE TEMP FUNCTION resourceAnonymizer(resource_id STRING) RETURNS STRING AS(
	ARRAY_TO_STRING(ARRAY(
		SELECT
			LEFT(REVERSE(n),3) FROM UNNEST(SPLIT(REGEXP_REPLACE(resource_id,"[:/]", ""), "-")) AS n
	), "-")
);`

var commonNameAnonymizer = fmt.Sprintf(
	`CREATE TEMP FUNCTION CommonNameAnonymizer(name STRING) RETURNS STRING AS (%s);`,
	pickVariantByModulo(commonNames, "ABS(FARM_FINGERPRINT(name))"))

func getCustomerNames(m map[string]string) []string {
	keys := make([]string, 0, len(m))

	for key := range m {
		keys = append(keys, key)
	}

	return keys
}

func getSQLUnionQuery(customerNames []string, demoTables map[string]*bqTable, exportTimeThreshold string) string {
	var sqlStrings []string

	for _, customer := range customerNames {
		tableName := demoTables[customer].FullTableName
		selectStatement := fmt.Sprintf("SELECT %s FROM %s WHERE export_time > (%s) AND export_time < CURRENT_TIMESTAMP()", gcpQueryFields, tableName, exportTimeThreshold)
		sqlStrings = append(sqlStrings, selectStatement)
	}

	return strings.Join(sqlStrings, "\n\tUNION ALL\n\t")
}

func getDemoTableNames(demoBillingIds map[string]string, bq *bigquery.Client) map[string]*bqTable {
	gcpDemoTables := make(map[string]*bqTable)

	for demoCustomerName, demoBillingID := range demoBillingIds {
		gcpDemoTables[demoCustomerName] = getBqTable(bq, gcpTableMgmtDomain.BillingProjectProd, demoBillingID, gcpTableMgmtDomain.GetCustomerBillingDataset, gcpTableMgmtDomain.GetCustomerBillingTable)
	}

	return gcpDemoTables
}

const projectIdGenerator string = `
	CREATE TEMP FUNCTION ProjectIdGenerator(project_id STRING) RETURNS STRING AS(
	CONCAT((SELECT value FROM UNNEST(LabelGenerator(project_id)) WHERE key = 'env'), '-', (SELECT value FROM UNNEST(LabelGenerator(project_id)) WHERE key = 'project'), "-", LEFT(REGEXP_REPLACE(LOWER(TO_BASE64(SHA1(project_id))), r"\W", ""), LEAST(8, LENGTH(project_id))))
);
`

const AncestryNamesAnonymizer string = `CREATE TEMP FUNCTION AncestryNamesAnonymizer(name STRING) RETURNS STRING AS(
	ARRAY_TO_STRING(ARRAY(
		SELECT
			LEFT(REVERSE(n),3) FROM UNNEST(SPLIT(name, "/")) AS n
	), "/")
);`

const KubernetesClusterNameAnonymizer string = `CREATE TEMP FUNCTION KubernetesClusterNameAnonymizer(cluster_name STRING) RETURNS STRING AS (
	CASE MOD(UNICODE(cluster_name), 4)
		WHEN 0 THEN 'prod-cluster'
		WHEN 1 THEN 'dev-cluster'
		WHEN 2 THEN 'stage-cluster'
		WHEN 3 THEN 'beta-cluster'
	ELSE NULL
	END
);`

const KubernetesNamespaceAnonymizer string = `CREATE TEMP FUNCTION KubernetesNamespaceAnonymizer(namespace STRING) RETURNS STRING AS (
	IF(namespace LIKE 'kube:%' OR namespace LIKE 'goog-k8s-%' OR namespace IN ('kube-system', 'web', 'default'), namespace,
		CASE MOD(UNICODE(namespace), 4)
			WHEN 0 THEN 'adventure-game'
			WHEN 1 THEN 'action-game'
			WHEN 2 THEN 'childrens-game'
			WHEN 3 THEN 'shooter-game'
		ELSE NULL
		END
	)
);`

func getQueryWithLabels(ctx context.Context, bq *bigquery.Client, query string, customerID string) *bigquery.Query {
	q := bq.Query(query)
	log.AddQueryLabels(ctx, q, customerID)

	return q
}

const eKSClusterNameAnonymizer string = `CREATE TEMP FUNCTION choose_cluster_name(value INT64) RETURNS STRING
	AS (
	  CASE
		WHEN MOD(value, 9) = 0 THEN 'Production CloudCosmos'
		WHEN MOD(value, 9) = 1 THEN 'Production NebulaCraft'
		WHEN MOD(value, 9) = 2 THEN 'Production AstroCloud'
		WHEN MOD(value, 9) = 3 THEN 'Staging CloudCosmos'
		WHEN MOD(value, 9) = 4 THEN 'Staging NebulaCraft'
		WHEN MOD(value, 9) = 5 THEN 'Staging AstroCloud'
		WHEN MOD(value, 9) = 6 THEN 'Sandbox CyberOrbit'
		WHEN MOD(value, 9) = 7 THEN 'Sandbox CloudSwiftX'
		WHEN MOD(value, 9) = 8 THEN 'Sandbox NexusStellar'
		ELSE 'Production Final'
	  END
	);`

func createTableIfNoneExists(ctx context.Context, bqClient *bigquery.Client, projectID string, datasetID string, tableID string, sourceTable *bigquery.Table) (*bqTable, error) {
	dataset := bqClient.DatasetInProject(projectID, datasetID)
	table := dataset.Table(tableID)

	exists, _, err := common.BigQueryDatasetExists(ctx, bqClient, projectID, datasetID)
	if !exists && err == nil {
		md := bigquery.DatasetMetadata{
			Location: "US",
		}
		if err := dataset.Create(ctx, &md); err != nil {
			return nil, err
		}
	}

	exists, _, err = common.BigQueryTableExists(ctx, bqClient, projectID, datasetID, tableID)
	if !exists && err == nil {
		md, err := sourceTable.Metadata(ctx)
		if err != nil {
			return nil, err
		}

		newTableMetadata := bigquery.TableMetadata{
			Schema:            md.Schema,
			TimePartitioning:  md.TimePartitioning,
			RangePartitioning: md.RangePartitioning,
			Clustering:        md.Clustering,
			Description:       md.Description,
			ExpirationTime:    md.ExpirationTime,
			Labels:            log.DefaultPresentationLogFields,
		}

		err = table.Create(ctx, &newTableMetadata)
		if err != nil {
			return nil, err
		}
	}

	return &bqTable{
		Table:         table,
		FullTableName: strings.Join([]string{table.ProjectID, table.DatasetID, table.TableID}, "."),
	}, nil
}

func getIncrementalUpdateThreshold(incrementalUpdate bool, destination *bqTable) string {
	exportTimeThreshold := "TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 DAY)"
	if incrementalUpdate {
		exportTimeThreshold = fmt.Sprintf("SELECT COALESCE(MAX(export_time),%s) FROM `%s` where export_time > '2024-01-01'", exportTimeThreshold, destination.FullTableName)
	}

	return exportTimeThreshold
}

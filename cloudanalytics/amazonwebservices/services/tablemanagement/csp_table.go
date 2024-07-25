package tablemanagement

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	cspReportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport/domain"
	queryPkg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/querytable"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *BillingTableManagementService) UpdateCSPAccounts(
	ctx context.Context,
	allPartitions bool,
	startDate string,
	numPartitions int,
) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	// if allPartitions is false
	// startDate == "YYYY-MM-DD" -> Update all table partitions starting YYYY-MM-DD till now
	// startDate == "" (or not valid date value) -> (default) Update from beginning of the previous month
	if !allPartitions {
		startDate = service.CalculateStartDate(startDate)
	}

	if tableExists, err := CheckDestinationTable(ctx, bq, &domain.BigQueryTableUpdateRequest{
		DestinationProjectID: utils.GetBillingProject(),
		DestinationDatasetID: utils.GetCSPBillingDataset(),
		DestinationTableName: utils.GetCSPBillingTableName(),
	}); err != nil {
		l.Error(err)
		return err
	} else if !tableExists {
		allPartitions = true
		numPartitions = 0
		startDate = ""
	}

	data := &domain.BigQueryTableUpdateRequest{
		DefaultProjectID:      utils.GetBillingProject(),
		DefaultDatasetID:      utils.BillingDataset,
		DestinationProjectID:  utils.GetBillingProject(),
		DestinationDatasetID:  utils.GetCSPBillingDataset(),
		DestinationTableName:  utils.GetCSPFullBillingTableName(),
		AllPartitions:         allPartitions,
		WriteDisposition:      bigquery.WriteTruncate,
		ConfigJobID:           "cloud_analytics_csp_aws_accounts",
		WaitTillDone:          true,
		CSP:                   true,
		Clustering:            service.GetTableClustering(true),
		FromDate:              startDate,
		FromDateNumPartitions: numPartitions,

		House:   common.HouseAdoption,
		Feature: common.FeatureCloudAnalytics,
		Module:  common.ModuleTableManagementCsp,
	}

	rawBillingDataTable := utils.GetRawBillingTable()
	enhDataquery, selectFrom := GetCSPDiscountQuery(rawBillingDataTable)
	query := cspreport.GetCSPTableMetadataQuery(!data.AllPartitions,
		&cspReportDomain.CSPMetadataQueryData{
			Cloud:                          common.Assets.AmazonWebServices,
			BillingDataTableFullName:       rawBillingDataTable,
			MetadataTableFullName:          fmt.Sprintf("%s.%s.%s", cspreport.GetCSPMetadataProject(), cspreport.GetCSPMetadataDataset(), cspreport.GetCSPMetadataTable()),
			BindIDField:                    "project_id",
			MetadataBindIDField:            "id",
			EnchancedBillingDataQuery:      enhDataquery,
			EnchancedBillingDataSelectFrom: selectFrom,
		})
	l.Infof("AWS Update Query: %s", query)

	if err := service.RunBillingTableUpdateQuery(ctx, bq, query, data); err != nil {
		l.Error(err)
		return err
	}

	metadataJoinFrom := `
	(SELECT
        customer_id,
        MIN(type) as type,
        MIN(primary_domain) AS primary_domain,
        MIN(classification) AS classification,
        MIN(payee_country) AS payee_country,
        MIN(payer_country) AS payer_country,
        MIN(territory) AS territory,
        MIN(field_sales_representative) AS field_sales_representative,
        MIN(strategic_account_manager) AS strategic_account_manager,
        MIN(technical_account_manager) AS technical_account_manager
    FROM
        (SELECT DISTINCT customer_id, type, primary_domain, classification, payee_country, payer_country, territory, field_sales_representative, strategic_account_manager, technical_account_manager
        FROM ` + fmt.Sprintf("%s.%s.%s", cspreport.GetCSPMetadataProject(), cspreport.GetCSPMetadataDataset(), cspreport.GetCSPMetadataTable()) + `
        WHERE type = "amazon-web-services")
    GROUP BY customer_id)`
	dataSelectFrom := `(SELECT ` + queryPkg.AWSLineItemsNonNullFields() + ` FROM ` + querytable.GetAWSLineItemsTable() + `)`

	awsLineItemsQuery := cspreport.GetCSPTableMetadataQuery(!data.AllPartitions,
		&cspReportDomain.CSPMetadataQueryData{
			Cloud:                          common.Assets.AmazonWebServices,
			BillingDataTableFullName:       querytable.GetAWSLineItemsTable(),
			MetadataTableFullName:          metadataJoinFrom,
			BindIDField:                    domainQuery.FieldBillingAccountID,
			MetadataBindIDField:            "customer_id",
			EnchancedBillingDataQuery:      "",
			EnchancedBillingDataSelectFrom: dataSelectFrom,
		})
	l.Infof("AWS Line Items (FlexSave) Update Query: %s", awsLineItemsQuery)

	data.WriteDisposition = bigquery.WriteAppend
	if err := service.RunBillingTableUpdateQuery(ctx, bq, awsLineItemsQuery, data); err != nil {
		return err
	}

	if err := s.reportStatusService.UpdateReportStatus(ctx, domainQuery.CSPCustomerID, common.ReportStatus{
		Status: map[string]common.StatusInfo{
			string(common.AWSReportStatus): {
				LastUpdate: time.Now(),
			},
		},
	}); err != nil {
		l.Error(err)
	}

	if err := createCSPAggregatedTableTask(ctx, allPartitions, startDate); err != nil {
		return err
	}

	return nil
}

// GetCSPDiscountQuery - builds AWS CSP dicounts query that will enhance raw AWS
// billing data with discount column
func GetCSPDiscountQuery(rawBillingTable string) (string, string) {
	withData := "enhanced_data"
	query := `WITH {with_data} as (
		SELECT
		T.*,
			IFNULL(ARRAY(
				SELECT STRUCT(
					CAST(D.is_commitment as STRING) as is_commitment
				)
				FROM
					{discounts_table} AS D
				WHERE
					T.project_id = D.project_id
					AND DATE(T.usage_date_time) >= D.start_date
					AND (D.end_date IS NULL OR DATE(T.usage_date_time) < D.end_date)
					AND DATE(_PARTITIONTIME) <= CURRENT_DATE()
				)[SAFE_OFFSET(0)], STRUCT(NULL as is_commitment)
			)
			AS discount
		FROM {raw_billing_table} AS T
	)`
	query = strings.NewReplacer(
		"{with_data}", withData,
		"{discounts_table}", utils.GetFullDiscountsTable(),
		"{raw_billing_table}", rawBillingTable,
	).Replace(query)

	return query, withData
}

func createCSPAggregatedTableTask(ctx context.Context, allPartitions bool, startDate string) error {
	baseURI := "/tasks/analytics/amazon-web-services/csp-accounts-aggregate?"
	if allPartitions {
		baseURI += "allPartitions=true"
	} else if startDate != "" {
		baseURI += "from=" + startDate
	}

	config := common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_POST,
		Path:         baseURI,
		Queue:        common.TaskQueueCloudAnalyticsCSP,
		Body:         nil,
		ScheduleTime: nil,
	}

	_, err := common.CreateCloudTask(ctx, &config)

	return err
}

func (s *BillingTableManagementService) UpdateCSPAggregatedTable(
	ctx context.Context,
	allPartitions bool,
	startDate string,
	numPartitions int,
) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	// if allPartitions is false
	// startDate == "YYYY-MM-DD" -> Update all table partitions strating YYYY-MM-DD till now
	// startDate == "" (or not valid date value) -> (default) Update from beginning of the previous month
	if !allPartitions {
		startDate = service.CalculateStartDate(startDate)
	}

	query := getAggregatedQuery(allPartitions, "", true, domainQuery.BillingTableSuffixDay)
	l.Debugf("AWS aggregated table query: %s\n", query)

	err := service.RunBillingTableUpdateQuery(ctx, bq, query,
		&domain.BigQueryTableUpdateRequest{
			DefaultProjectID:      utils.GetBillingProject(),
			DefaultDatasetID:      utils.GetCSPBillingDataset(),
			DestinationProjectID:  utils.GetBillingProject(),
			DestinationDatasetID:  utils.GetCSPBillingDataset(),
			DestinationTableName:  utils.GetCSPBillingTableName(),
			AllPartitions:         allPartitions,
			WriteDisposition:      bigquery.WriteTruncate,
			ConfigJobID:           "cloud_analytics_csp_aws_aggregated",
			WaitTillDone:          false,
			CSP:                   true,
			Clustering:            service.GetTableClustering(true),
			FromDate:              startDate,
			FromDateNumPartitions: numPartitions,

			House:   common.HouseAdoption,
			Feature: common.FeatureCloudAnalytics,
			Module:  common.ModuleTableManagementCsp,
		})
	if err != nil {
		l.Errorf("Error updating aggregated AWS table: %s\n", err)
		return err
	}

	l.Debug("Aggregated AWS table updated.\n")

	return nil
}

func CheckDestinationTable(ctx context.Context, bq *bigquery.Client, data *domain.BigQueryTableUpdateRequest) (bool, error) {
	dstDataset := bq.DatasetInProject(data.DestinationProjectID, data.DestinationDatasetID)
	dstTable := dstDataset.Table(data.DestinationTableName)

	// Check dataset existence
	datasetExists, _, err := common.BigQueryDatasetExists(ctx, bq, dstDataset.ProjectID, dstDataset.DatasetID)
	if err != nil {
		return false, err
	}

	if !datasetExists {
		if err := dstDataset.Create(ctx, &data.DatasetMetadata); err != nil {
			return false, err
		}
	}

	// Check table existence
	tableExists, md, err := common.BigQueryTableExists(ctx, bq, dstTable.ProjectID, dstTable.DatasetID, dstTable.TableID)
	if err != nil {
		return false, err
	}

	if !tableExists {
		return false, nil
	}

	if !md.RequirePartitionFilter {
		mdtu := bigquery.TableMetadataToUpdate{
			RequirePartitionFilter: true,
		}
		if _, err := dstTable.Update(ctx, mdtu, md.ETag); err != nil {
			return true, err
		}
	}

	return true, nil
}

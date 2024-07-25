package tablemanagement

import (
	"context"
	"os"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

func (s *BillingTableManagementService) UpdateCSPTable(ctx context.Context, startDate, endDate string) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	queryBytes, err := os.ReadFile("cloudanalytics/scheduledQueries/csp-azure-update.sql")
	if err != nil {
		return err
	}

	query := string(queryBytes)

	endDate = calculateEndDate(endDate, startDate)
	startDate = service.CalculateStartDate(startDate)

	queryJob := bq.Query(query)
	queryJob = setBaseBigQueryJobConfig(queryJob, startDate, endDate)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	return nil
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

	// if allPartitions is false, then startDate is required
	// startDate == "YYYY-MM-DD" -> Update all table partitions starting YYYY-MM-DD till now
	// startDate == "" (or not valid date value) -> (default) Update from beginning of the previous month
	if !allPartitions {
		startDate = service.CalculateStartDate(startDate)
	}

	query := getAggregatedQuery(allPartitions, "", true, domainQuery.BillingTableSuffixDay)

	l.Debug(query)

	tableUpdateParams := &domain.BigQueryTableUpdateRequest{
		DefaultProjectID:      microsoftazure.GetBillingProject(),
		DefaultDatasetID:      microsoftazure.GetCSPBillingDataset(),
		DestinationProjectID:  microsoftazure.GetBillingProject(),
		DestinationDatasetID:  microsoftazure.GetCSPBillingDataset(),
		DestinationTableName:  microsoftazure.GetCSPBillingTableName(),
		AllPartitions:         allPartitions,
		WriteDisposition:      bigquery.WriteTruncate,
		ConfigJobID:           "cloud_analytics_csp_azure_aggregated",
		WaitTillDone:          false,
		CSP:                   true,
		Clustering:            service.GetTableClustering(true),
		FromDate:              startDate,
		FromDateNumPartitions: numPartitions,

		House:   common.HouseAdoption,
		Feature: common.FeatureCloudAnalytics,
		Module:  common.ModuleTableManagementCsp,
	}

	err := service.RunBillingTableUpdateQuery(ctx, bq, query, tableUpdateParams)
	if err != nil {
		l.Errorf("Failed updating CSP Azure aggregated table with error: %s", err)
		return err
	}

	return nil
}

// calculateEndDate calculates the end date for the query.
// if endDate is provided return it as is.
// If startDate is not a valid date, then it will return the last day of the current month.
// If startDate is a valid date, then it will return the current date.
func calculateEndDate(endDate, startDate string) string {
	if endDate != "" {
		return endDate
	}

	if _, err := time.Parse(times.YearMonthDayLayout, startDate); err != nil {
		today := times.CurrentDayUTC()
		return today.AddDate(0, 1, -today.Day()).Format(times.YearMonthDayLayout)
	}

	return time.Now().Format(times.YearMonthDayLayout)
}

func setBaseBigQueryJobConfig(queryJob *bigquery.Query, startDate, endDate string) *bigquery.Query {
	timeParams := []bigquery.QueryParameter{
		{
			Name:  "start_date",
			Value: startDate,
		},
		{
			Name:  "end_date",
			Value: endDate,
		},
	}

	queryJob.Parameters = timeParams
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.DefaultProjectID = microsoftazure.GetBillingProject()
	queryJob.DefaultDatasetID = microsoftazure.GetCSPBillingDataset()
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true
	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: "cloud_analytics_csp_azure_account", AddJobIDSuffix: true}

	return queryJob
}

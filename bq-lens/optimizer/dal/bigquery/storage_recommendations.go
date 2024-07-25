package bq

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	doitBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func (d *BigqueryDAL) RunStorageRecommendationsQuery(
	ctx context.Context,
	bq *bigquery.Client,
	replacements domain.Replacements,
	now time.Time,
) ([]bqmodels.StorageRecommendationsResult, error) {
	jobsDeduplicated, err := replaceJobsDeduplicatedWithClause(replacements, now, false)
	if err != nil {
		return nil, err
	}

	replacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", replacements.ProjectID,
		"{datasetIdPlaceHolder}", replacements.DatasetID,
		"{tablesDiscoveryTable}", replacements.TablesDiscoveryTable,
		"{jobsDeduplicatedWithClause}", jobsDeduplicated,
		"{getTableIdBaseName}", bqmodels.GetTableIdBaseName,
	)

	query := replacer.Replace(bqmodels.TablesRecommendations)
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.StorageRecommendationsResult](iter)
}

func (d *BigqueryDAL) RunTotalScanPricePerPeriod(
	ctx context.Context,
	bq *bigquery.Client,
	replacements domain.Replacements,
	now time.Time,
) ([]bqmodels.ScanPricePerPeriod, error) {
	jobsDeduplicated, err := replaceJobsDeduplicatedWithClause(replacements, now, true)
	if err != nil {
		return nil, err
	}

	totalScanReplacer := strings.NewReplacer(
		"{jobsDeduplicatedWithClause}", jobsDeduplicated,
	)

	query := totalScanReplacer.Replace(bqmodels.TotalScan)
	queryJob := bq.Query(query)
	queryJob.Location = replacements.Location

	iter, err := d.RunQuery(ctx, bq, query)
	if err != nil {
		return nil, err
	}

	return doitBQ.LoadRows[bqmodels.ScanPricePerPeriod](iter)
}

func replaceJobsDeduplicatedWithClause(replacements domain.Replacements, now time.Time, isScanPriceRun bool) (string, error) {
	historicalMinDateNeeded := now.AddDate(0, 0, -30)

	allProjectsWithReservations := replacements.ProjectsWithReservations
	for _, editionProjects := range replacements.ProjectsByEdition {
		allProjectsWithReservations = append(allProjectsWithReservations, editionProjects...)
	}

	historicalJobs, err := domain.ReplaceHistoricalJobs(bqmodels.TotalScanPrice, replacements, allProjectsWithReservations, historicalMinDateNeeded)
	if err != nil {
		return "", err
	}

	projectsWithReservations := `("")`

	if isScanPriceRun {
		projectsWithReservations = domain.GetProjectReservations(allProjectsWithReservations)
	}

	deduplicationReplacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", replacements.ProjectID,
		"{datasetIdPlaceHolder}", replacements.DatasetID,
		"{tablesDiscoveryTable}", replacements.TablesDiscoveryTable,
		"{queryPlaceholder}", "NULL",
		"{modePlaceholder}", "NOT",
		"{projectsWithReservations}", projectsWithReservations,
		"{startDate}", time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
		"{historicalJobsPlaceholder}", historicalJobs,
	)

	return deduplicationReplacer.Replace(bqmodels.JobsDeduplicatedWithClause), nil
}

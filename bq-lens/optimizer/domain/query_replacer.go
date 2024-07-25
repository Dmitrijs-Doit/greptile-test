package domain

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery/reservation/apiv1/reservationpb"

	"github.com/doitintl/errors"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

// queryReplacer is a general function used for all queries within the Optimizer
func QueryReplacer(queryID bqmodels.QueryName, queryValue string, replacements Replacements, timeRange bqmodels.TimeRange, now time.Time) (string, error) {
	if replacements.ProjectID == "" || replacements.DatasetID == "" || replacements.TablesDiscoveryTable == "" {
		return "", fmt.Errorf("detected empty values for required field (projectID, datasetID or tablesDiscovery): %+v", replacements)
	}

	day, err := GetDayBasedOnTimeRange(timeRange)
	if err != nil {
		return "", err
	}

	replacements.StartDate = replacements.MaxDate.AddDate(0, 0, -day).Format(times.YearMonthDayLayout)
	replacements.EndDate = replacements.MaxDate.Format(times.YearMonthDayLayout)

	historicalMinDateNeeded := now.AddDate(0, 0, -day)

	jobsDeduplication, err := replaceJobsDeduplicatedWithClause(queryID, replacements, historicalMinDateNeeded)
	if err != nil {
		return "", errors.Wrapf(err, "replaceJobsDeduplicatedWithClause() failed for queryValue '%s' and time range '%s'", queryValue, timeRange)
	}

	replacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", replacements.ProjectID,
		"{datasetIdPlaceHolder}", replacements.DatasetID,
		"{tablesDiscoveryTable}", replacements.TablesDiscoveryTable,
		"{jobsDeduplicatedWithClause}", jobsDeduplication,
		"{getTableIdBaseName}", bqmodels.GetTableIdBaseName,
		"{startDate}", replacements.StartDate,
		"{scanAttributionWithClause}", replaceScanAttributionWithClause(replacements),
	)

	return replacer.Replace(queryValue), nil
}

func replaceScanAttributionWithClause(replacements Replacements) string {
	replacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", replacements.ProjectID,
		"{datasetIdPlaceHolder}", replacements.DatasetID,
		"{tablesDiscoveryTable}", replacements.TablesDiscoveryTable,
	)

	return replacer.Replace(bqmodels.ScanAttributionWithClause)
}

func replaceJobsDeduplicatedWithClause(queryID bqmodels.QueryName, replacements Replacements, historicalMinDateNeeded time.Time) (string, error) {
	modePlaceholder, err := getModePlaceholder(queryID, false)
	if err != nil {
		return "", err
	}

	var projectsWithReservations []string

	switch queryID {
	case bqmodels.StandardScheduledQueriesMovement,
		bqmodels.StandardSlotsExplorer,
		bqmodels.StandardUserSlots,
		bqmodels.StandardBillingProjectSlots:
		projectsWithReservations = replacements.ProjectsByEdition[reservationpb.Edition_STANDARD]
	case bqmodels.EnterpriseScheduledQueriesMovement,
		bqmodels.EnterpriseSlotsExplorer,
		bqmodels.EnterpriseUserSlots,
		bqmodels.EnterpriseBillingProjectSlots:
		projectsWithReservations = replacements.ProjectsByEdition[reservationpb.Edition_ENTERPRISE]
	case bqmodels.EnterprisePlusScheduledQueriesMovement,
		bqmodels.EnterprisePlusSlotsExplorer,
		bqmodels.EnterprisePlusUserSlots,
		bqmodels.EnterprisePlusBillingProjectSlots:
		projectsWithReservations = replacements.ProjectsByEdition[reservationpb.Edition_ENTERPRISE_PLUS]
	default:
		projectsWithReservations = replacements.ProjectsWithReservations
	}

	historicalJobs, err := ReplaceHistoricalJobs(queryID, replacements, projectsWithReservations, historicalMinDateNeeded)
	if err != nil {
		return "", err
	}

	replacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", replacements.ProjectID,
		"{datasetIdPlaceHolder}", replacements.DatasetID,
		"{tablesDiscoveryTable}", replacements.TablesDiscoveryTable,
		"{queryPlaceholder}", getQueryPlaceholder(queryID),
		"{modePlaceholder}", modePlaceholder,
		"{projectsWithReservations}", GetProjectReservations(projectsWithReservations),
		"{startDate}", replacements.StartDate,
		"{historicalJobsPlaceholder}", historicalJobs,
	)

	return replacer.Replace(bqmodels.JobsDeduplicatedWithClause), nil
}

func ReplaceHistoricalJobs(
	queryID bqmodels.QueryName,
	replacements Replacements,
	projectsWithReservations []string,
	historicalMinDateNeeded time.Time,
) (string, error) {
	if replacements.MinDate.Before(historicalMinDateNeeded) {
		return "", nil
	}

	modePlaceholder, err := getModePlaceholder(queryID, true)
	if err != nil {
		return "", err
	}

	replacer := strings.NewReplacer(
		"{projectIdPlaceHolder}", replacements.ProjectID,
		"{datasetIdPlaceHolder}", replacements.DatasetID,
		"{historicalJobsModePlaceholder}", modePlaceholder,
		"{projectsWithReservations}", GetProjectReservations(projectsWithReservations),
		"{startDate}", replacements.StartDate,
		"{endDate}", replacements.MinDate.Format(times.YearMonthDayLayout),
	)

	return replacer.Replace(bqmodels.HistoricalJobsUnion), nil
}

func GetProjectReservations(projects []string) string {
	if len(projects) == 0 {
		return `("")`
	}

	quotedProjects := make([]string, len(projects))
	for i, project := range projects {
		// Enclose each project in double quotes
		quotedProjects[i] = fmt.Sprintf(`"%s"`, project)
	}

	result := fmt.Sprintf("(%s)", strings.Join(quotedProjects, ","))

	return result
}

func getQueryPlaceholder(queryID bqmodels.QueryName) string {
	switch queryID {
	case bqmodels.ClusterTables, bqmodels.PartitionTables, bqmodels.UsePartitionField:
		return "protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobConfiguration.queryValue.queryValue"

	default:
		return "NULL"
	}
}

func getModePlaceholder(queryID bqmodels.QueryName, isHistoricalJobs bool) (string, error) {
	var modePlaceholder string

	if queryID == bqmodels.TotalScanPrice || queryID == bqmodels.StorageSavings {
		return "NOT", nil
	}

	mode, err := getQueryMode(queryID)
	if err != nil {
		return "", err
	}

	switch mode {
	case bqmodels.FlatRate,
		bqmodels.StandardEdition,
		bqmodels.EnterpriseEdition,
		bqmodels.EnterprisePlusEdition:
		modePlaceholder = ""
	case bqmodels.OnDemand:
		modePlaceholder = "NOT"

	default:
		modePlaceholder = "IS NOT NULL OR projectId"

		if !isHistoricalJobs {
			modePlaceholder = "IS NOT NULL OR protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId"
		}
	}

	return modePlaceholder, nil
}

func getQueryMode(queryID bqmodels.QueryName) (bqmodels.Mode, error) {
	for mode, queries := range bqmodels.QueriesPerMode {
		for name := range queries {
			if name == queryID {
				return mode, nil
			}
		}
	}

	return "", errors.New("queryValue provided is not within existing modes (hybrid, flat-rate or on-demand)")
}

func GetDayBasedOnTimeRange(t bqmodels.TimeRange) (int, error) {
	switch t {
	case bqmodels.TimeRangeMonth:
		return 30, nil
	case bqmodels.TimeRangeWeek:
		return 7, nil
	case bqmodels.TimeRangeDay:
		return 1, nil

	default:
		return 0, errors.Errorf("unknown time range provided '%s'", t)
	}
}

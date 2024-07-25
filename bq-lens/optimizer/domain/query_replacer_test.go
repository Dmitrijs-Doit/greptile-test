package domain

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

//go:embed testData/*.sql
var testData embed.FS

var (
	projectID               = "mock-project-id"
	datasetID               = "mock-dataset-id"
	tableDiscovery          = "mock-table-discovery"
	projects                = []string{"project1", "project2", "project3"}
	mostRecentDate          = time.Date(2024, 03, 27, 0, 0, 0, 0, time.UTC)
	oldestDate              = mostRecentDate.AddDate(0, 0, -1)
	historicalMinDateNeeded = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)
)

func TestQueryReplacer(t *testing.T) {
	var (
		mockTime = time.Date(2022, 01, 01, 12, 0, 0, 0, time.UTC)
	)

	type args struct {
		queryID      bqmodels.QueryName
		queryValue   string
		replacements Replacements
		timeRange    bqmodels.TimeRange
	}

	tests := []struct {
		name         string
		args         args
		expectedFile string
		wantErr      bool
	}{
		{
			name: "costFromTableTypes with projects reservations",
			args: args{
				queryID:    bqmodels.CostFromTableTypes,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.CostFromTableTypes],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					HistoricalJobs:           nil,
					ProjectsWithReservations: projects,
					MinDate:                  time.Time{},
					MaxDate:                  time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "costFromTableTypes",
		},
		{
			name: "tableStoragePrice",
			args: args{
				queryID:    bqmodels.TableStoragePrice,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.TableStoragePrice],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
					MinDate:              time.Time{},
					MaxDate:              time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "tableStoragePrice",
		},
		{
			name: "datasetStoragePrice",
			args: args{
				queryID:    bqmodels.DatasetStoragePrice,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.DatasetStoragePrice],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
					MinDate:              time.Time{},
					MaxDate:              time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "datasetStoragePrice",
		},
		{
			name: "projectStoragePrice",
			args: args{
				queryID:    bqmodels.ProjectStoragePrice,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.ProjectStoragePrice],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
					MinDate:              time.Time{},
					MaxDate:              time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "projectStoragePrice",
		},
		{
			name: "projectStorageTB",
			args: args{
				queryID:    bqmodels.ProjectStorageTB,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.ProjectStorageTB],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
					MinDate:              time.Time{},
					MaxDate:              time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "projectStorageTB",
		},
		{
			name: "datasetStorageTB",
			args: args{
				queryID:    bqmodels.DatasetStorageTB,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.DatasetStorageTB],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
					MinDate:              time.Time{},
					MaxDate:              time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "datasetStorageTB",
		},
		{
			name: "tableStorageTB",
			args: args{
				queryID:    bqmodels.TableStorageTB,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.TableStorageTB],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
					MinDate:              time.Time{},
					MaxDate:              time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "tableStorageTB",
		},
		{
			name: "scheduledQueriesMovement",
			args: args{
				queryID:    bqmodels.ScheduledQueriesMovement,
				queryValue: bqmodels.QueriesPerMode[bqmodels.FlatRate][bqmodels.ScheduledQueriesMovement],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "scheduledQueriesMovement",
		},
		{
			name: "billingProjectSlots",
			args: args{
				queryID:    bqmodels.BillingProjectSlots,
				queryValue: bqmodels.QueriesPerMode[bqmodels.FlatRate][bqmodels.BillingProjectSlots],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "billingProjectSlots",
		},
		{
			name: "billingProjectSlotsTopUsers",
			args: args{
				queryID:    bqmodels.BillingProjectSlots,
				queryValue: bqmodels.BillingProjectSlotsQueries[bqmodels.BillingProjectSlotsTopUsers],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "billingProjectSlotsTopUsers",
		},
		{
			name: "billingProjectSlotsTopQueries",
			args: args{
				queryID:    bqmodels.BillingProjectSlots,
				queryValue: bqmodels.BillingProjectSlotsQueries[bqmodels.BillingProjectSlotsTopQueries],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "billingProjectSlotsTopQueries",
		},
		{
			name: "flat rate slots explorer",
			args: args{
				queryID:    bqmodels.SlotsExplorerFlatRate,
				queryValue: bqmodels.QueriesPerMode[bqmodels.FlatRate][bqmodels.SlotsExplorerFlatRate],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
					MaxDate:                  mockTime,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "flatrate_slotsExplorer",
		},
		{
			name: "on demand slots explorer",
			args: args{
				queryID:    bqmodels.SlotsExplorerOnDemand,
				queryValue: bqmodels.QueriesPerMode[bqmodels.OnDemand][bqmodels.SlotsExplorerOnDemand],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
					MaxDate:                  mockTime,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "ondemand_slotsExplorer",
		},
		{
			name: "on demand billing project",
			args: args{
				queryID:    bqmodels.BillingProject,
				queryValue: bqmodels.QueriesPerMode[bqmodels.OnDemand][bqmodels.BillingProject],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
					MaxDate:                  mockTime,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "ondemand_billingProject",
		},
		{
			name: "on demand billing project top queries",
			args: args{
				queryID:    bqmodels.BillingProject,
				queryValue: bqmodels.OnDemandBillingProjectQueries[bqmodels.BillingProjectTopQueries],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
					MaxDate:                  mockTime,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "ondemand_billingProject_topQueries",
		},
		{
			name: "on demand billing project top users",
			args: args{
				queryID:    bqmodels.BillingProject,
				queryValue: bqmodels.OnDemandBillingProjectQueries[bqmodels.BillingProjectTopUsers],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					ProjectsWithReservations: projects,
					MaxDate:                  mockTime,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			expectedFile: "ondemand_billingProject_topUsers",
		},
		{
			name: "validation error empty table discovery value",
			args: args{
				queryID:    bqmodels.CostFromTableTypes,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.CostFromTableTypes],
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     "",
					HistoricalJobs:           nil,
					ProjectsWithReservations: nil,
					MinDate:                  time.Time{},
					MaxDate:                  time.Time{},
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			wantErr: true,
		},
		{
			name: "failure due to unknown timeRange provided",
			args: args{
				queryID:    bqmodels.CostFromTableTypes,
				queryValue: bqmodels.QueriesPerMode[bqmodels.Hybrid][bqmodels.CostFromTableTypes],
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
				},
				timeRange: bqmodels.TimeRange("past-0-day"),
			},
			wantErr: true,
		},
		{
			name: "failure due to unknown queryValue name",
			args: args{
				queryID: "unknow queryValue name",
				replacements: Replacements{
					ProjectID:            projectID,
					DatasetID:            datasetID,
					TablesDiscoveryTable: tableDiscovery,
				},
				timeRange: bqmodels.TimeRangeMonth,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := QueryReplacer(tt.args.queryID,
				tt.args.queryValue,
				tt.args.replacements,
				tt.args.timeRange,
				mockTime,
			)
			if err != nil {
				assert.True(t, tt.wantErr)
			} else {
				data, err := testData.ReadFile(fmt.Sprintf("testData/%s.sql", tt.expectedFile))
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}

				assert.Equal(t, normalizeSpace(string(data)), normalizeSpace(got))
			}
		})
	}
}

func Test_replaceJobsDeduplicatedWithClause(t *testing.T) {
	type args struct {
		query        bqmodels.QueryName
		replacements Replacements
		timeRange    bqmodels.TimeRange
	}

	tests := []struct {
		name         string
		args         args
		expectedFile string
		wantErr      bool
	}{
		{
			name: "replaces all fields correctly",
			args: args{
				query:     bqmodels.CostFromTableTypes,
				timeRange: bqmodels.TimeRangeMonth,
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					StartDate:                mostRecentDate.Format(times.YearMonthDayLayout),
					HistoricalJobs:           nil,
					ProjectsWithReservations: projects,
					MinDate:                  oldestDate,
					MaxDate:                  mostRecentDate,
				},
			},
			expectedFile: "jobsDeduplicatedWithClause",
		},
		{
			name: "failed to replace string due to unknown queryValue",
			args: args{
				query:     "unknown queryValue",
				timeRange: bqmodels.TimeRangeMonth,
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					StartDate:                mostRecentDate.Format(times.YearMonthDayLayout),
					HistoricalJobs:           nil,
					ProjectsWithReservations: projects,
					MinDate:                  oldestDate,
					MaxDate:                  mostRecentDate,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceJobsDeduplicatedWithClause(tt.args.query, tt.args.replacements, historicalMinDateNeeded)
			if err != nil {
				assert.True(t, tt.wantErr)
			} else {
				data, err := testData.ReadFile(fmt.Sprintf("testData/%s.sql", tt.expectedFile))
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}

				assert.Equal(t, normalizeSpace(string(data)), normalizeSpace(got))
			}

			fmt.Print(got)

		})
	}
}

func Test_replaceHistoricalJobs(t *testing.T) {
	type args struct {
		queryName    bqmodels.QueryName
		replacements Replacements
	}

	tests := []struct {
		name         string
		args         args
		expectedFile string
		wantErr      bool
	}{
		{
			name: "replaces fields for query correctly",
			args: args{
				queryName: bqmodels.CostFromTableTypes,
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					StartDate:                mostRecentDate.Format(times.YearMonthDayLayout),
					HistoricalJobs:           nil,
					ProjectsWithReservations: projects,
					MinDate:                  oldestDate,
					MaxDate:                  mostRecentDate,
				},
			},
			expectedFile: "historicalJobsUnion",
		},
		{
			name: "failed due to unknown queryValue",
			args: args{
				queryName: "unknown",
				replacements: Replacements{
					ProjectID:                projectID,
					DatasetID:                datasetID,
					TablesDiscoveryTable:     tableDiscovery,
					StartDate:                mostRecentDate.Format(times.YearMonthDayLayout),
					HistoricalJobs:           nil,
					ProjectsWithReservations: projects,
					MinDate:                  oldestDate,
					MaxDate:                  mostRecentDate,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplaceHistoricalJobs(tt.args.queryName, tt.args.replacements, projects, historicalMinDateNeeded)
			if err != nil {
				assert.True(t, tt.wantErr)
			} else {
				data, err := testData.ReadFile(fmt.Sprintf("testData/%s.sql", tt.expectedFile))
				if err != nil {
					t.Fatalf("Failed to read file: %v", err)
				}

				assert.Equal(t, normalizeSpace(string(data)), normalizeSpace(got))
			}
		})
	}
}

func Test_getModePlaceholder(t *testing.T) {
	type args struct {
		queryName        bqmodels.QueryName
		isHistoricalJobs bool
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test OnDemand",
			args: args{
				queryName:        bqmodels.LimitingJobsSavings,
				isHistoricalJobs: false,
			},
			want: "NOT",
		},
		{
			name: "Test FlatRate",
			args: args{
				queryName:        bqmodels.ScheduledQueriesMovement,
				isHistoricalJobs: false,
			},
		},
		{
			name: "Test Hybrid with historical jobs",
			args: args{
				queryName:        bqmodels.TableStorageTB,
				isHistoricalJobs: true,
			},
			want: "IS NOT NULL OR projectId",
		},
		{
			name: "Test Hybrid without historical jobs",
			args: args{
				queryName:        bqmodels.TableStorageTB,
				isHistoricalJobs: false,
			},
			want: "IS NOT NULL OR protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.projectId",
		},
		{
			name: "Test unknown queryValue",
			args: args{
				queryName:        "unknownQuery",
				isHistoricalJobs: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getModePlaceholder(tt.args.queryName, tt.args.isHistoricalJobs)
			if (err != nil) != tt.wantErr {
				t.Errorf("getModePlaceholder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("getModePlaceholder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getProjectReservations(t *testing.T) {
	type args struct {
		projects []string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no projects",
			args: args{projects: []string{}},
			want: `("")`,
		},
		{
			name: "single project",
			args: args{projects: []string{"project1"}},
			want: `("project1")`,
		},
		{
			name: "multiple projects",
			args: args{projects: []string{"project1", "project2", "project3"}},
			want: `("project1","project2","project3")`,
		},
		{
			name: "with empty project names",
			args: args{projects: []string{"", "project2", ""}},
			want: `("","project2","")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetProjectReservations(tt.args.projects); got != tt.want {
				t.Errorf("GetProjectReservations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func normalizeSpace(str string) string {
	// Matches one or more whitespace characters (space, tab, newline, etc.)
	spacePattern := regexp.MustCompile(`\s+`)
	// Replace all sequences of whitespace with a single space
	normalized := spacePattern.ReplaceAllString(str, " ")
	// Trim leading and trailing spaces
	return strings.TrimSpace(normalized)
}

package service

import (
	"context"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	crmIface "github.com/doitintl/cloudresourcemanager/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/dal"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	bqLensDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal"
	bqLensIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal/iface"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	cloudConnectServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudconnect/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BackfillScheduler struct {
	loggerProvider        logger.Provider
	jobsSinksMetadata     bqLensIface.JobsSinksMetadata
	tasksWriter           bqLensIface.TaskCreator
	cloudConnect          cloudConnectServiceIface.CloudConnectService
	doitCmpHistoricalJobs iface.DoitCmpHistoricalJobs
}

func NewBackfillScheduler(loggerProvider logger.Provider, conn *connection.Connection) *BackfillScheduler {
	jobsSinksMetadata := bqLensDal.NewJobsSinksMetadataDal(conn.Firestore(context.Background()))
	tasksWriter := bqLensDal.NewCloudTaskDal(conn.CloudTaskClient)
	historicalJobs := dal.NewHistoricalJobs(loggerProvider)
	cloudConnect := cloudconnect.NewCloudConnectService(loggerProvider, conn)

	return &BackfillScheduler{
		loggerProvider:        loggerProvider,
		jobsSinksMetadata:     jobsSinksMetadata,
		tasksWriter:           tasksWriter,
		cloudConnect:          cloudConnect,
		doitCmpHistoricalJobs: historicalJobs,
	}
}

func (s *BackfillScheduler) ScheduleBackfill(ctx context.Context, sinkID string, testMode bool) error {
	// get sink metadata by sinkID -> superQuery/jobs-sinks/jobsSinksMetadata/$sinkID
	sink, err := s.jobsSinksMetadata.GetSinkMetadata(ctx, sinkID)
	if err != nil {
		return err
	}

	customerID := sink.Customer.ID

	// get bq client for customer
	connect, _, err := s.cloudConnect.NewGCPClients(ctx, customerID)
	if err != nil {
		return err
	}

	bq := connect.BQ.BigqueryService
	defer bq.Close()

	crm := connect.CRM

	// ensure doitCmpHistoricalJobsTable is correct
	table := bq.DatasetInProject(bq.Project(), bqLensDomain.DoitCmpDatasetID).
		Table(bqLensDomain.DoitCmpHistoricalJobsTable)

	if _, err := bqLensDal.EnsureTableIsCorrect(
		ctx,
		table,
		bqLensDomain.DoitCmpHistoricalJobsTableMetadata); err != nil {
		return err
	}

	// Get date for 30 days ago to know which customers should be backfilled
	date30DaysAgo := time.Now().UTC().
		Truncate(time.Duration(24 * time.Hour)).
		Add(-30 * 24 * time.Hour)

	// Get first record time from customer (to know how far back to backfill)
	sinkFirstRecordTime, err := s.doitCmpHistoricalJobs.GetSinkFirstRecordTime(ctx, bq, sink.ProjectLocation, sink.ProjectID)
	if err != nil {
		return err
	}

	if sinkFirstRecordTime.After(date30DaysAgo) {
		if err := s.scheduleBackfill(ctx, bq, crm, sinkID, customerID, date30DaysAgo, sinkFirstRecordTime, testMode); err != nil {
			return err
		}
	}

	return nil
}

func (s *BackfillScheduler) scheduleBackfill(ctx context.Context, client *bigquery.Client, crm crmIface.CloudResourceManager, sinkID, customerID string, date30DaysAgo, sinkFirstRecordTime time.Time, testMode bool) error {
	// Get all dates to backfill
	dates := getDatesByRange(date30DaysAgo, sinkFirstRecordTime)

	// Get all projects to backfill
	var projects []string

	plist, err := crm.ListProjects(ctx, "")
	if err != nil {
		return err
	}

	for _, project := range plist {
		projects = append(projects, project.ID)
	}

	// Filter projects to be backfilled based on historical jobs time range
	projectsToBeBackfilled, err := s.filterProjectsToBackfill(ctx, client, sinkID, projects, date30DaysAgo, sinkFirstRecordTime)
	if err != nil {
		return err
	}

	// Save projects to be backfilled in FS
	if err := s.jobsSinksMetadata.UpdateBackfillProgress(ctx, sinkID, projectsToBeBackfilled); err != nil {
		return err
	}

	// schedule backfill for relevant projects
	if !testMode {
		if err := s.scheduleBackfillForProjects(ctx, sinkID, customerID, projectsToBeBackfilled, date30DaysAgo, sinkFirstRecordTime, dates); err != nil {
			return err
		}
	}

	return nil
}

func (s *BackfillScheduler) filterProjectsToBackfill(ctx context.Context, client *bigquery.Client, sinkID string, projects []string, date30DaysAgo, sinkFirstRecordTime time.Time) ([]string, error) {
	var projectsToBackfill []string

	var (
		mu sync.Mutex     // protects projectsToBackfill slice from concurrent writes
		wg sync.WaitGroup // wait for all projects to be checked
	)

	for _, project := range projects {
		wg.Add(1)

		// Check if project has bq usage
		// If it does -> add to projectsToBackfill and save backfill progress in FS
		go func(wg *sync.WaitGroup, mu *sync.Mutex, project string) {
			defer wg.Done()

			// l := s.loggerProvider(ctx)
			l, err := logger.NewLogger(ctx.(*gin.Context)) // TODO: we need to fix logger to make it thread safe
			if err != nil {
				return
			}

			l.SetLabels(map[string]string{
				"house":   "adoption",
				"feature": "bq-lens",
				"module":  "backfill",
				"service": "BackfillScheduler",
				"sinkID":  sinkID,
				"project": project,
			})

			addBackfill, err := s.decideToScheduleBackfillForProject(ctx, client, project, date30DaysAgo, sinkFirstRecordTime)
			if err != nil {
				l.SetLabel("function", "decideToScheduleBackfillForProject")
				l.Error(err)

				return
			}

			if addBackfill {
				// Save backfill progress for a project in FS
				if err := s.jobsSinksMetadata.UpdateSinkProjectProgress(ctx, sinkID, project, 0); err != nil {
					l.SetLabel("function", "UpdateSinkProjectProgress")
					l.Error(err)

					return
				}

				mu.Lock()
				projectsToBackfill = append(projectsToBackfill, project)
				mu.Unlock()
			}
		}(&wg, &mu, project)
	}

	wg.Wait()

	return projectsToBackfill, nil
}

func (s *BackfillScheduler) decideToScheduleBackfillForProject(ctx context.Context, client *bigquery.Client, project string, date30DaysAgo, sinkFirstRecordTime time.Time) (bool, error) {
	return s.doitCmpHistoricalJobs.CheckIfProjectHasBQUsage(ctx, client, project, date30DaysAgo, sinkFirstRecordTime)
}

func (s *BackfillScheduler) scheduleBackfillForProjects(ctx context.Context, sinkID, customerID string, projects []string, date30DaysAgo, sinkFirstRecordTime time.Time, dates []time.Time) error {
	var (
		wg sync.WaitGroup
	)

	// schedule backfill for each project
	for _, project := range projects {
		// schedule backfill for each project date
		for _, date := range dates {
			wg.Add(1)

			go func(wg *sync.WaitGroup, project string, date time.Time) {
				defer wg.Done()

				if err := s.scheduleBackfillForProjectDate(ctx, sinkID, customerID, project, date, date30DaysAgo, sinkFirstRecordTime); err != nil {
					// l := s.loggerProvider(ctx)
					l, err := logger.NewLogger(ctx.(*gin.Context)) // TODO: we need to fix logger to make it thread safe
					if err != nil {
						return
					}

					l.SetLabels(map[string]string{
						"house":    "adoption",
						"feature":  "bq-lens",
						"module":   "backfill",
						"service":  "BackfillScheduler",
						"function": "scheduleBackfillForProjectDate",
						"sinkID":   sinkID,
					})

					l.Error(err)
				}
			}(&wg, project, date)
		}
	}

	wg.Wait()

	return nil
}

func (s *BackfillScheduler) scheduleBackfillForProjectDate(ctx context.Context, sinkID, customerID, project string, date, date30DaysAgo, sinkFirstRecordTime time.Time) error {
	// Update one day's backfill metadata in Firestore
	dateBackInfo := getDayBackfillInfo(date, date30DaysAgo, sinkFirstRecordTime)

	if err := s.jobsSinksMetadata.UpdateBackfillForProjectAndDate(ctx, sinkID, project, date, dateBackInfo); err != nil {
		return err
	}

	// Schedule backfill job
	return s.tasksWriter.CreateBackfillTask(ctx, *dateBackInfo, date, project, customerID, sinkID)
}

func getDatesByRange(startDate, endDate time.Time) []time.Time {
	dates := make([]time.Time, 0)

	startDate = startDate.Truncate(time.Duration(24) * time.Hour)
	endDate = endDate.Truncate(time.Duration(24)*time.Hour).AddDate(0, 0, 1)

	// get all dates between start and end date
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}

	return dates
}

func getDayBackfillInfo(date, date30DaysAgo, sinkFirstRecordTime time.Time) *domain.DateBackfillInfo {
	dayStart := date.Truncate(time.Duration(24) * time.Hour)
	dayEnd := dayStart.AddDate(0, 0, 1)

	// Don't go further back as date30DaysAgo
	if date30DaysAgo.After(dayStart) {
		dayStart = date30DaysAgo
	}

	// Don't fetch jobs which have been executed later than sinkFirstRecordTime
	if sinkFirstRecordTime.Before(dayEnd) {
		dayEnd = sinkFirstRecordTime
	}

	return &domain.DateBackfillInfo{
		BackfillMinCreationTime: dayStart,
		BackfillMaxCreationTime: dayEnd,
		// BackfillInitialMaxCreationTime: dayEnd,
		BackfillDone: false,
	}
}

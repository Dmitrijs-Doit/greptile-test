package service

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/dal"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/dal/iface"
	backfill "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/domain"
	bqLensDal "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal"
	bqLensIface "github.com/doitintl/hello/scheduled-tasks/bq-lens/dal/iface"
	bqLensDomain "github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	cloudConnectServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudconnect/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BackfillService struct {
	loggerProvider        logger.Provider
	jobsSinksMetadata     bqLensIface.JobsSinksMetadata
	cloudConnect          cloudConnectServiceIface.CloudConnectService
	doitCmpHistoricalJobs iface.DoitCmpHistoricalJobs
}

func NewBackfillService(loggerProvider logger.Provider, conn *connection.Connection) *BackfillService {
	jobsSinksMetadata := bqLensDal.NewJobsSinksMetadataDal(conn.Firestore(context.Background()))
	historicalJobs := dal.NewHistoricalJobs(loggerProvider)
	cloudConnect := cloudconnect.NewCloudConnectService(loggerProvider, conn)

	return &BackfillService{
		loggerProvider:        loggerProvider,
		jobsSinksMetadata:     jobsSinksMetadata,
		cloudConnect:          cloudConnect,
		doitCmpHistoricalJobs: historicalJobs,
	}
}

func (s *BackfillService) Backfill(
	ctx context.Context,
	sinkID string,
	customerID string,
	backfillProject string,
	backfillDate time.Time,
	backfillInfo backfill.DateBackfillInfo,
) error {
	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":           "adoption",
		"feature":         "bq-lens",
		"module":          "backfill",
		"service":         "Backfill",
		"backfillDate":    backfillDate.Format("2006-01-02"),
		"backfillProject": backfillProject,
		"customerID":      customerID,
		"sinkID":          sinkID,
	})

	// get bq client for customer
	connect, _, err := s.cloudConnect.NewGCPClients(ctx, customerID)
	if err != nil {
		l.Errorf("failed to get gcp clients: %v", err)

		return err
	}

	bq := connect.BQ.BigqueryService
	defer bq.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel() // if error occurs, cancel the read write context

	// get jobs stream
	jobs, err := s.doitCmpHistoricalJobs.GetJobsList(ctx, bq, backfillProject, backfillInfo.BackfillMinCreationTime, backfillInfo.BackfillMaxCreationTime)
	if err != nil {
		l.Errorf("failed to get jobs list: %v", err)

		return err
	}

	// target table
	table := bq.DatasetInProject(bq.Project(), bqLensDomain.DoitCmpDatasetID).
		Table(bqLensDomain.DoitCmpHistoricalJobsTable)

	// save jobs from stream
	err = s.doitCmpHistoricalJobs.SaveJobs(ctx, table, jobs)
	if err != nil {
		l.Errorf("failed to save jobs: %v", err)

		return err
	}

	// mark backfill as done
	now := time.Now().UTC()
	backfillInfo.BackfillDone = true
	backfillInfo.BackfillProcessEndTime = now
	backfillInfo.BackfillProcessLastUpdateTime = now

	// update project date backfill info in FS
	if err := s.jobsSinksMetadata.UpdateBackfillForProjectAndDate(ctx, sinkID, backfillProject, backfillDate, &backfillInfo); err != nil {
		l.Errorf("failed to update backfill for project and date in FS: %v", err)

		return err
	}

	l.Info("Project date backfill done")

	// update project progress in FS
	if err := s.updateSinkProjectProgress(ctx, sinkID, backfillProject); err != nil {
		l.Errorf("failed to update sink project progress in FS: %v", err)

		return err
	}

	// update sink progress in FS
	if err := s.updateSinkProgress(ctx, sinkID); err != nil {
		l.Errorf("failed to update sink progress in FS: %v", err)

		return err
	}

	return nil
}

func (s *BackfillService) updateSinkProjectProgress(ctx context.Context, sinkID, backfillProject string) error {
	projectDates, err := s.jobsSinksMetadata.GetSinkProjectDates(ctx, sinkID, backfillProject)
	if err != nil {
		return err
	}

	var projectDone int

	for _, date := range projectDates {
		if date.BackfillDone {
			projectDone++
		}
	}

	progress := int(float64(projectDone) / float64(len(projectDates)) * 100)

	if err := s.jobsSinksMetadata.UpdateSinkProjectProgress(ctx, sinkID, backfillProject, progress); err != nil {
		return err
	}

	return nil
}

func (s *BackfillService) updateSinkProgress(ctx context.Context, sinkID string) error {
	projects, err := s.jobsSinksMetadata.GetSinkProjects(ctx, sinkID)
	if err != nil {
		return err
	}

	inProgress := make([]string, 0)

	for _, project := range projects {
		if !project.BackfillDone {
			inProgress = append(inProgress, project.ProjectName)
		}
	}

	if err := s.jobsSinksMetadata.UpdateBackfillProgress(ctx, sinkID, inProgress); err != nil {
		return err
	}

	return nil
}

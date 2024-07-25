package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Job struct {
	Logger logger.Provider
	*connection.Connection
}

func NewJob(log logger.Provider, conn *connection.Connection) *Job {
	return &Job{
		log,
		conn,
	}
}

func (j *Job) GetJobByPrefix(ctx context.Context, bq *bigquery.Client, prefix string) (jobID string, err error) {
	logger := j.Logger(ctx)
	jobIterator := bq.Jobs(ctx)
	jobs := []*bigquery.Job{}

	for {
		job, err := jobIterator.Next()
		if err != nil {
			if err == iterator.Done {
				break
			} else {
				return "", fmt.Errorf("unable to find job with prefix %s. Caused by %s", prefix, err)
			}
		}

		if strings.Contains(job.ID(), prefix) {
			jobs = append(jobs, job)
		}
	}

	if len(jobs) == 0 {
		logger.Infof("unable to find job with prefix %s. Caused by no job with requested prefix where found", prefix)
		return "", nil
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].LastStatus().Statistics.EndTime.After(jobs[j].LastStatus().Statistics.EndTime)
	})

	return jobs[0].ID(), nil
}

func (j *Job) GetJobStatus(ctx context.Context, bq *bigquery.Client, jobID string, location string) (*bigquery.JobStatus, error) {
	logger := j.Logger(ctx)

	job, err := bq.JobFromIDLocation(ctx, jobID, location)
	if err != nil {
		err = fmt.Errorf("unable to get JobFromID for job %s. Caused by %s", jobID, err)
		logger.Error(err)

		return nil, err
	}

	jobStatus, err := job.Status(ctx)
	if err != nil {
		err = fmt.Errorf("unable to get job.Status for job %s. Caused by %s", jobID, err)
		logger.Error(err)

		return nil, err
	}

	return jobStatus, nil
}

func (j *Job) HandleRunningJob(ctx context.Context, job *bigquery.Job, ttl *time.Time, extraTime time.Duration) (err error) {
	logger := j.Logger(ctx)

	jobDoneCh := make(chan struct{})
	jobFailedCh := make(chan struct{})
	shouldCancelJob := make(chan struct{})
	jobCancelingCh := make(chan struct{})
	finishRunningCh := make(chan struct{})

	defer close(finishRunningCh)

	go func() {
		var js *bigquery.JobStatus

		for {
			time.Sleep(10 * time.Second)
			select {
			case <-finishRunningCh:
				close(jobDoneCh)
				close(jobFailedCh)
				close(jobCancelingCh)
				close(shouldCancelJob)

				return
			default:
				js, err = job.Status(ctx)
				if err != nil {
					logger.Errorf("unable to get job %s status. Caused by %s", job.ID(), err)
					shouldCancelJob <- struct{}{}
				}

				if js.Done() {
					err = js.Err()
					if err != nil {
						logger.Errorf("unable to exec query. Caused by %s", err)
						jobFailedCh <- struct{}{}
					} else {
						jobDoneCh <- struct{}{}
					}

					break
				}
			}
		}
	}()

	select {
	case <-jobFailedCh:
		err = common.NewJobExecutionFailure(fmt.Sprintf("job %s failed. Caused by %s", job.ID(), err))
		logger.Error(err)

		return err

	case <-jobDoneCh:
		logger.Infof("JOB %s is DONE", job.ID())
		return nil

	case <-ctx.Done():
		err = common.NewJobExecutionTimeout(fmt.Sprintf("context timeout while waiting for job %s", job.ID()))
		logger.Error(err)

		return err

	case <-time.After(time.Until(*ttl)):
		err = common.NewJobExecutionTimeout(fmt.Sprintf("timeout while waiting for job %s", job.ID()))
		logger.Error(err)

		ctx = context.Background()

	case <-shouldCancelJob:
		err = fmt.Errorf("unable to wait for job %s. Caused by %s", job.ID(), err)
		logger.Error(err)
	}

	logger.Infof("attempting to terminate job %s", job.ID())

	err = job.Cancel(ctx)
	if err != nil {
		err = common.NewJobExecutionStuck(fmt.Sprintf("unable to cancel for job %s. Caused by %s", job.ID(), err))
		logger.Error(err)

		return err
	}

	jobStuckCh := make(chan struct{})
	succeddedJobCh := make(chan struct{})
	canceledJobCh := make(chan struct{})

	finishCancelCh := make(chan struct{})
	defer close(finishCancelCh)

	go func() {
		var js *bigquery.JobStatus

		for {
			time.Sleep(10 * time.Second)
			select {
			case <-finishCancelCh:
				close(jobStuckCh)
				close(succeddedJobCh)
				close(canceledJobCh)

				return
			default:
				js, err = job.Status(ctx)
				if err != nil {
					logger.Errorf("unable to get job %s status. Caused by %s", job.ID(), err)
					jobStuckCh <- struct{}{}
				}

				if js.Done() {
					if js.Err() != nil {
						logger.Errorf("Job exited with err %s", js.Err())
						canceledJobCh <- struct{}{}
					} else {
						succeddedJobCh <- struct{}{}
					}

					break
				}
			}
		}
	}()

	select {
	case <-time.After(time.Until(ttl.Add(extraTime))):
		err = common.NewJobExecutionStuck(fmt.Sprintf("timeout while waiting for job %s", job.ID()))
		logger.Error(err)

		return err

	case <-jobStuckCh:
		err = common.NewJobExecutionStuck(fmt.Sprintf("unable to cancel job %s", job.ID()))
		logger.Error(err)

		return err

	case <-succeddedJobCh:
		err = common.NewJobCancelButFinished(job.ID())
		logger.Error(err)

		return err

	case <-canceledJobCh:
		err = common.NewJobCanceled(fmt.Sprintf("job %s canceled", job.ID()))
		logger.Error(err)

		return err
	}
}

func (j *Job) WaitUntilInternalJobIsDone(ctx context.Context, bq *bigquery.Client, jobID string, timeout time.Duration) error {
	logger := j.Logger(ctx)

	job, err := bq.JobFromID(ctx, jobID)
	if err != nil {
		//handle error
		err = fmt.Errorf("unable to exec JobFromID %s. Caused by %s", jobID, err)
		logger.Error(err)

		return err
	}

	doneCh := make(chan struct{})
	failureCh := make(chan struct{})
	waitFailureCh := make(chan struct{})

	defer func() {
		close(doneCh)
		close(failureCh)
		close(waitFailureCh)
	}()

	go func() {
		var js *bigquery.JobStatus

		js, err = job.Wait(ctx)
		if err != nil {
			logger.Errorf("unable to get job %s status. Caused by %s", jobID, err)
			waitFailureCh <- struct{}{}

			return
		}

		err = js.Err()
		if err != nil {
			logger.Errorf("unable to exec query. Caused by %s", err)
			failureCh <- struct{}{}

			return
		} else {
			doneCh <- struct{}{}
		}

		return
	}()

	select {
	case <-time.After(timeout):
		err = common.NewJobExecutionTimeout(fmt.Sprintf("timeout while waiting for job %s", jobID))
		logger.Error(err)

		return err

	case <-ctx.Done():
		err = common.NewJobExecutionTimeout(fmt.Sprintf("context timeout while waiting for job %s", jobID))
		logger.Error(err)

		return err

	case <-waitFailureCh:
		logger.Errorf("unable to wait for job %s. Caused by %s", jobID, err)
		return fmt.Errorf("unable to wait for job %s. Caused by %s", jobID, err)

	case <-failureCh:
		err = common.NewJobExecutionFailure(fmt.Sprintf("job %s failed. Caused by %s", jobID, err))
		logger.Error(err)

		return err

	case <-doneCh:
		logger.Infof("JOB %s is DONE", jobID)
		return nil
	}
}

func (j *Job) CancelRunningJob(ctx context.Context, bq *bigquery.Client, jobID string) error {
	logger := j.Logger(ctx)

	job, err := bq.JobFromID(ctx, jobID)
	if err != nil {
		err = fmt.Errorf("unable to get job %s. Caused by %s", jobID, err)
		logger.Error(err)

		return err
	}

	js, err := job.Status(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			err = fmt.Errorf("unable to cancel job %s. Caused by %s", jobID, gapiErr)
			logger.Error(err)
		}

		return err
	}

	logger.Infof("job status %s", js.State)

	err = job.Cancel(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			err = fmt.Errorf("unable to cancel job %s. Caused by %s", jobID, gapiErr)
			logger.Error(err)
		}

		return err
	}

	return nil
}

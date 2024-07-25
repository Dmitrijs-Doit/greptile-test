package scripts

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sync"

	"cloud.google.com/go/bigquery"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/workerpool"
	jobworker "github.com/doitintl/workerpool/job"
)

func bigqueryJobsCancel(ctx *gin.Context) []error {
	bq, err := bigquery.NewClient(ctx, "")
	if err != nil {
		return []error{err}
	}

	f, err := os.Open("./scripts/data/jobs.csv")
	if err != nil {
		return []error{err}
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	jobsCH := make(chan workerpool.Job)
	wp := jobworker.NewWorkerPool(ctx, jobsCH, 5)
	errs := make([]error, 0)
	mu := &sync.Mutex{}

	for i := 0; ; i++ {
		row, err := csvReader.Read()
		if err != nil {
			return errs
		}

		jobProject := row[0]
		jobID := row[1]
		j := NewJob(bq, &errs, mu, jobProject, jobID)

		wp.AddJob(j)
	}
}

type CancelPendingBQJob struct {
	errors    *[]error
	mutex     *sync.Mutex
	client    *bigquery.Client
	projectID string
	jobID     string
}

func NewJob(bq *bigquery.Client, errors *[]error, mu *sync.Mutex, projectID, jobID string) CancelPendingBQJob {
	return CancelPendingBQJob{
		client:    bq,
		errors:    errors,
		mutex:     mu,
		projectID: projectID,
		jobID:     jobID,
	}
}

func (j CancelPendingBQJob) Run(ctx context.Context) {
	job, err := j.client.JobFromProject(ctx, j.projectID, j.jobID, "US")
	if err != nil {
		fmterr := fmt.Errorf("failed to get job from project, job ID: %s\n%s", j.jobID, err)
		j.mutex.Lock()
		*j.errors = append(*j.errors, fmterr)
		j.mutex.Unlock()

		return
	}

	if job.LastStatus().State == bigquery.Pending {
		if err := job.Cancel(ctx); err != nil {
			fmterr := fmt.Errorf("failed to cancel job, job ID: %s\n%s", j.jobID, err)
			j.mutex.Lock()
			*j.errors = append(*j.errors, fmterr)
			j.mutex.Unlock()

			return
		}

		fmt.Println("job cancelled: ", job.ID())
	}
}

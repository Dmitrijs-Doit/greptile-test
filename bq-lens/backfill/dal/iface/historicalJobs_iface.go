package iface

import (
	"context"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
)

//go:generate mockery --name DoitCmpHistoricalJobs --output ../mocks --case=underscore
type DoitCmpHistoricalJobs interface {
	GetSinkFirstRecordTime(ctx context.Context, bq *bigquery.Client, projectLocation, projectID string) (time.Time, error)
	CheckIfProjectHasBQUsage(ctx context.Context, client *bigquery.Client, projectID string, timeRangeStart, timeRangeEnd time.Time) (bool, error)

	// GetJobsList returns a channel that receives maybe job or error if occurred during the retrieval
	GetJobsList(
		ctx context.Context,
		client *bigquery.Client,
		projectID string,
		minCreationTime time.Time,
		maxCreationTime time.Time) (chan domain.Maybe[*bigquery.Job], error)

	// SaveJobs saves jobs received from the channel to the destination table
	// jobs is a channel that receives maybe jobs to be saved to the destination table or error if occurred during the retrieval
	SaveJobs(ctx context.Context, destinationTable *bigquery.Table, jobs chan domain.Maybe[*bigquery.Job]) error
}

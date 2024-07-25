package dal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	bqIface "github.com/doitintl/bigquery/iface"
	"github.com/doitintl/buffer/arraybuff"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type HistoricalJobsRow struct {
	Ts                       bigquery.NullTimestamp `bigquery:"ts"`
	JobID                    bigquery.NullString    `bigquery:"jobId"`
	ProjectID                bigquery.NullString    `bigquery:"projectId"`
	Location                 bigquery.NullString    `bigquery:"location"`
	UserEmail                bigquery.NullString    `bigquery:"user_email"`
	State                    bigquery.NullString    `bigquery:"state"`
	CreationTime             bigquery.NullTimestamp `bigquery:"creationTime"`
	EndTime                  bigquery.NullTimestamp `bigquery:"endTime"`
	StartTime                bigquery.NullTimestamp `bigquery:"startTime"`
	Query                    bigquery.NullString    `bigquery:"query"`
	TotalBytesProcessed      bigquery.NullInt64     `bigquery:"totalBytesProcessed"`
	TotalBytesBilled         bigquery.NullInt64     `bigquery:"totalBytesBilled"`
	TotalSlotMs              bigquery.NullInt64     `bigquery:"totalSlotMs"`
	Property                 bigquery.NullString    `bigquery:"property"`
	JobType                  bigquery.NullString    `bigquery:"jobType"`
	BillingTier              bigquery.NullInt64     `bigquery:"billingTier"`
	CacheHit                 bigquery.NullBool      `bigquery:"cacheHit"`
	ReferencedTables         []ReferencedTable      `bigquery:"referencedTables"`
	TotalPartitionsProcessed bigquery.NullInt64     `bigquery:"totalPartitionsProcessed"`
	QueryPlan                []bigquery.NullString  `bigquery:"queryPlan"`
	Timeline                 bigquery.NullString    `bigquery:"timeline"`
	ReservationUsage         []ReservationUsage     `bigquery:"reservationUsage"`
	ReservationID            bigquery.NullString    `bigquery:"reservation_id"`
	Statistics               bigquery.NullString    `bigquery:"statistics"`
	Configuration            bigquery.NullString    `bigquery:"configuration"`
	Status                   bigquery.NullString    `bigquery:"status"`
	ErrorResult              ErrorResult            `bigquery:"errorResult"`
}

type ReferencedTable struct {
	ProjectID bigquery.NullString `bigquery:"projectId"`
	DatasetID bigquery.NullString `bigquery:"datasetId"`
	TableID   bigquery.NullString `bigquery:"tableId"`
}

type ReservationUsage struct {
	Name   bigquery.NullString `bigquery:"name"`
	SlotMs bigquery.NullInt64  `bigquery:"slotMs"`
}

type ErrorResult struct {
	Reason    bigquery.NullString `bigquery:"reason"`
	Message   bigquery.NullString `bigquery:"message"`
	Location  bigquery.NullString `bigquery:"location"`
	DebugInfo bigquery.NullString `bigquery:"debugInfo"`
}

const (
	inserterMaxRows = 500
)

type HistoricalJobs struct {
	loggerProvider logger.Provider
}

func NewHistoricalJobs(log logger.Provider) iface.DoitCmpHistoricalJobs {
	return &HistoricalJobs{
		loggerProvider: log,
	}
}

func (h *HistoricalJobs) GetSinkFirstRecordTime(ctx context.Context, client *bigquery.Client, projectLocation, projectID string) (time.Time, error) {
	q := `SELECT 
		MIN(protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobStatistics.createTime) as minTs
    FROM ` + "`%s.doitintl_cmp_bq.cloudaudit_googleapis_com_data_access`" + `
    WHERE 
		protopayload_auditlog.servicedata_v1_bigquery.jobCompletedEvent.job.jobName.jobId is not null
    	AND DATE(timestamp) >= DATE(DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 30 DAY))`

	q = fmt.Sprintf(q, projectID)

	query := client.Query(q)
	query.Location = projectLocation

	it, err := query.Read(ctx)
	if err != nil {
		return time.Time{}, err
	}

	var (
		minTs time.Time

		row struct {
			MinTs bigquery.NullTimestamp `bigquery:"minTs"`
		}
	)

	err = it.Next(&row)

	if err != nil {
		return time.Time{}, err
	}

	minTs = row.MinTs.Timestamp

	return minTs, nil
}

func (h *HistoricalJobs) CheckIfProjectHasBQUsage(ctx context.Context, client *bigquery.Client, projectID string, timeRangeStart, timeRangeEnd time.Time) (bool, error) {
	it := client.Jobs(ctx)

	it.MinCreationTime = timeRangeStart
	it.MaxCreationTime = timeRangeEnd
	it.ProjectID = projectID
	it.State = bigquery.Done
	it.AllUsers = true
	it.PageInfo().MaxSize = 1

	// get the one job
	job, err := it.Next()
	if errors.Is(err, iterator.Done) {
		return false, nil
	}

	if err != nil {
		// BQ is not enabled, there are no datasets
		// or we do not have access.
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusNotFound ||
				gapiErr.Code == http.StatusBadRequest ||
				gapiErr.Code == http.StatusForbidden {
				return false, nil
			}
		}

		return false, err
	}

	return job != nil, nil
}

func (h *HistoricalJobs) GetJobsList(
	ctx context.Context,
	client *bigquery.Client,
	projectID string,
	minCreationTime time.Time,
	maxCreationTime time.Time) (chan domain.Maybe[*bigquery.Job], error) {
	jobs := make(chan domain.Maybe[*bigquery.Job])

	it := client.Jobs(ctx)

	it.MinCreationTime = minCreationTime
	it.MaxCreationTime = maxCreationTime
	it.ProjectID = projectID
	it.State = bigquery.Done
	it.AllUsers = true

	go iterateJobs(it, jobs)

	return jobs, nil
}

func iterateJobs(it *bigquery.JobIterator, jobs chan domain.Maybe[*bigquery.Job]) {
	defer close(jobs)

	for {
		job, err := it.Next()
		if err != nil {
			if errors.Is(err, iterator.Done) {
				break
			}

			// if error occurred during iteration, send the error to the channel and break the loop
			jobs <- domain.Maybe[*bigquery.Job]{Err: err}

			break
		}

		jobs <- domain.Maybe[*bigquery.Job]{Value: job, Err: nil}
	}
}

func (h *HistoricalJobs) SaveJobs(ctx context.Context, destinationTable *bigquery.Table, jobs chan domain.Maybe[*bigquery.Job]) error {
	l := h.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		"house":     "adoption",
		"feature":   "bq-lens",
		"module":    "backfill",
		"service":   "SaveJobs",
		"dst_table": destinationTable.FullyQualifiedName(),
	})

	inserter := destinationTable.Inserter()

	// create a buffer for the rows to be inserted
	buffer, err := arraybuff.NewBuffer(ctx, inserterMaxRows, 0, func(ctx context.Context, rows []*HistoricalJobsRow) error {
		if err := insertRecords(ctx, inserter, rows); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// iterate over jobs and save them to BQ
	for maybeJob := range jobs {
		// if error occurred during retrieval
		if maybeJob.Err != nil {
			l.Error(maybeJob.Err)

			// flush the buffer to save the remaining rows
			if err := buffer.Flush(ctx); err != nil {
				return err
			}

			return maybeJob.Err
		}

		if maybeJob.Value == nil {
			continue
		}

		row, err := convertJobToHistoricalJobsRow(maybeJob.Value)
		if err != nil {
			l.Error(err)

			continue
		}

		if err := buffer.Add(row); err != nil {
			// if error occurred during flussing the buffer, return the error
			return err
		}
	}

	// flush the buffer to save the remaining rows
	return buffer.Flush(ctx)
}

// insertRecords inserts the rows into the destination table
// using the provided inserter
// The function is throttling the insertions in case of errors
// that require throttling
// The function is using a binary search algorithm to find the
// optimal size of the batch to be inserted
func insertRecords[T any](ctx context.Context, inserter bqIface.IfcInserter, rows []T) error {
	var err error

	start := 0
	end := len(rows)
	size := end - start

	calcSize := func(start, end int) int {
		size := (end - start) / 2
		if size < 1 {
			size = 1
		}

		return size
	}

	calcEnd := func(start, size int) int {
		end := start + size
		if end > len(rows) {
			end = len(rows)
		}

		return end
	}

	needsThrottling := func(err error) bool {
		if e, ok := err.(*googleapi.Error); ok {
			if e.Code == http.StatusRequestEntityTooLarge ||
				e.Code == http.StatusTooManyRequests {
				return true
			}
		}

		return false
	}

	for start < len(rows) {
		if err = inserter.Put(ctx, rows[start:end]); err != nil {
			if needsThrottling(err) {
				size = calcSize(start, end)

				end = calcEnd(start, size)

				continue
			}

			return err
		}

		start = end
		end = calcEnd(start, size)
	}

	return nil
}

package dal

import (
	"context"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Query struct {
	loggerProvider logger.Provider
	*connection.Connection
}

func NewQuery(log logger.Provider, conn *connection.Connection) *Query {
	return &Query{
		log,
		conn,
	}
}

func (q *Query) ReadQueryResult(ctx context.Context, bq *bigquery.Client, query string) (it *bigquery.RowIterator, err error) {
	logger := q.loggerProvider(ctx)
	it, err = bq.Query(query).Read(ctx)

	if err != nil {
		logger.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
		return nil, err
	}

	return it, nil
}

func (q *Query) ExecQueryAsync(ctx context.Context, bq *bigquery.Client, data *common.BQExecuteQueryData) (job *bigquery.Job, err error) {
	queryJob := bq.Query(data.Query)
	queryJob.Priority = bigquery.InteractivePriority
	queryJob.Priority = bigquery.BatchPriority

	if data.DestinationTable != nil && len(data.DestinationTable.TableID) > 0 && !data.Export {
		queryJob.Dst = bq.DatasetInProject(data.DestinationTable.ProjectID, data.DestinationTable.DatasetID).Table(data.DestinationTable.TableID)
		queryJob.TimePartitioning = &bigquery.TimePartitioning{Type: "DAY", Field: "export_time"}
		queryJob.Clustering = data.Clustering
		queryJob.SchemaUpdateOptions = []string{"ALLOW_FIELD_ADDITION"}
	}

	queryJob.DefaultProjectID = data.DefaultTable.ProjectID
	queryJob.DefaultDatasetID = data.DefaultTable.DatasetID
	queryJob.DryRun = false
	queryJob.UseLegacySQL = false
	queryJob.AllowLargeResults = true
	queryJob.DisableFlattenedResults = true

	if !data.Export {
		queryJob.CreateDisposition = bigquery.CreateIfNeeded
		queryJob.WriteDisposition = data.WriteDisposition
	}

	queryJob.JobIDConfig = bigquery.JobIDConfig{JobID: data.ConfigJobID, AddJobIDSuffix: true}

	return queryJob.Run(ctx)
}

////TODO lionel modify to async func
//func (q *Query) ExecuteQuery(ctx context.Context, bq *bigquery.Client, data *common.BQExecuteQueryData, taskInProcess dataStructures.State) (string, error) {
//	job, err := q.ExecQueryAsync(ctx, bq, data)
//	if err != nil {
//		return "", err
//	}
//
//	//move to the main task flow
//	if err := q.taskJobID.SaveJobId(ctx, data.BillingAccountID, data.Internal, job.ID(), taskInProcess); err != nil {
//		return "", err
//	}
//
//	go func(job *bigquery.Job) {
//		<-ctx.Done()
//		err = job.Cancel(ctx)
//		if err != nil {
//			q.loggerProvider(ctx).Error(err)
//		}
//	}(job)
//
//	if data.WaitTillDone {
//		status, err := job.Wait(ctx)
//		if err != nil {
//			return "", err
//		}
//		if err := status.Err(); err != nil {
//			return "", err
//		}
//	} else {
//		ctxWait, cancelWait := context.WithTimeout(ctx, time.Second*5)
//		defer cancelWait()
//		status, err := job.Wait(ctxWait)
//		if err != nil {
//			if err == context.DeadlineExceeded {
//				status, err := job.Status(ctx)
//				if err != nil {
//					return "", err
//				}
//				if err := status.Err(); err != nil {
//					return "", err
//				}
//			} else {
//				return "", err
//			}
//		} else if err := status.Err(); err != nil {
//			return "", err
//		}
//	}
//
//	return job.ID(), nil
//}
